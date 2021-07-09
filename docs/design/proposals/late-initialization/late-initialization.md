## Introduction

[K8s convention](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md#late-initialization) describes **Late Initialization** as “When resource fields are set by a system controller after an object is created/updated”

In the context of an ACK service controller, when an AWS resource is created or updated by the controller, the AWS service may set optional fields for a resource to a default value and return them as part of Create/Update output or even as Get output in some cases. 

This **late initialization** of optional fields in the AWS resource are not currently reflected in the desired state of the resource. This causes a difference to be detected between the stored desired state of the resource and the latest state fetched from the AWS service at the start of a reconciliation loop when we call the resource manager's ReadOne method. However, this difference is erroneous since the controller does not need to call the resource manager's Update method in order for the actual state to match the desired state of the resource.

This document will propose the solution for above problem and provide the details of implementing it.

## Requirements

A solution for this problem should :

1. Update the desired state of the K8s custom resource with late initialized fields which are optional and not provided by the K8s user. 

2. Provide configuration for AWS service teams to indicate which operation output returns the late initialized fields. i.e. Get/List/Create/Update output.

    > AWS APIs have different behavior for returning the late initialized to the user. Some services return these fields as part of Create output, while some services will return the late initialized fields as part of  ReadOne output after the resource creation.

3. Wait for asynchronous resource creation to populate late initialized fields.

    > For some services/resources the late initialized fields do not appear immediately.  This happens when there is asynchronous processing involved in resource creation. The proposed solution should allow waiting and retry until the late initialized fields appear in the mentioned API output.

4. Allow the AWS service teams to provide custom code using hooks for handing of late initialized fields.

## Solutions

### Solution 1

#### TL;DR: 

There will be a new `r.rd.AddLateInitializedFields()` method call added in the reconciliation loop, which will return `true` if the desired spec was updated with late initialized field(s) from the latest spec. If so, then reconciliation loop will patch the desired spec with new updates. 

See the following sections for detailed explanation of this solution: 

#### Reconciler Updates

* Create separate variables for `readOneLatest`, `createLatest` and `updateLatest` instead of a single `latest` variable which gets overwritten after readOne/Create/Update operation in the reconciliation loop.
* Create a `desiredBase` which is copy of `desired` object after the status is updated(`patchResource()`) in etcd by the reconciler. We will use this copy as base to patch the desired resource spec in etcd.
* Add `specUpdated, err := r.rd.AddLateInitializedFields(desired, readOneLatest, createLatest, updateLatest)`
    > If the service teams have provided the configuration for populating late initialized field into `generator.yaml`, this method will update the desired spec accordingly and return `true, nil`. Otherwise it will return `false, nil`.
* If `specUpdated` is `true`, reconciler will patch the resource in etcd using `desiredBase` as base object, otherwise no patching of K8s custom resource will take place and reconciliation will finish. 

#### Sdk.go Updates

* Allow `sdkCreate` and `sdkUpdate` to populate the `Spec` of `latest` object and not just the `Status`.
* This will allow capturing the late initialized fields from Create and Update API output.

#### ResourceDescriptor Updates

* A  new method will be added in `ResourceDescriptor`, which will look like
    `AddLateInitializedFields(desired, readOneLatest, createLatest, updateLatest)`
* The default implementation of this method will be to return `false, nil` if there is no entry in `generator.yaml` for handling late initialized fields.
* If AWS service team has mentioned some fields that are supposed to be late initialized, this method will perform
    `desired.ko.Spec.{{path_from_generator.yaml}} = {{operation_from_generator.yaml}}Latest.ko.Spec.{{path_from_ganarator.yaml}}`
* The above statement will be executed with proper non-nil check on latest object, proper non-nil check on desired object and making sure the existing entry does not exist in the desired spec (i.e. Only perform assignment if the value is absent in desired spec).
* If the `add_late_initialized_fields_override` hook is present, that template will be used as method body of `AddLateInitializedFields` method. This hook is mutually exclusive to `pre_add_late_initialized_fields` and `post_add_late_initialized_fields`.
* If the `pre_add_late_initialized_fields` hook is present, that template will be inserted at the beginning of method body of  `AddLateInitializedFields` i.e. before performing nil checks and copying the variables from latest to desired
* If the `post_add_late_initialized_fields` hook is present, that template will be inserted before the last statement (`return specUpdated, err`) of  `AddLateInitializedFields` method i.e. after copying the variables from latest to desired
* See **Hook Updates** section for more details on these hooks.

#### Generator Config Updates

A new `LateInitializationConfig` struct will be introduced which will become a member of `ResourceConfig` struct.

`LateInitializationConfig` struct will look like

```
type LateInitializationConfig struct { 
   
   // valid values: 'readOne, create, update'
   DefaultSourceMethod *string `json:"default_source_method"`
   
   LateInitializedFields []LateInitializedFieldConfig `json:"late_initialized_fields,omitempty"`
}


type LateInitializedFieldConfig struct {
   
   // valid values: 'readOne, create, update'
   // If this is not mentioned, then default_source_method is used
   SourceMethod *string `json:"source_method,omitempty"`
   
   // Path to the field relative to Spec, which will be late initialized
   // Example for primitives: "a" to replace "desired.ko.Spec.a" with "<source_method>Latest.ko.spec.a"
   // Example for struct: "a.b" to replace "desired.ko.Spec.a.b" with "<op_name>Latest.ko.spec.a.b"
   // Example for map: "a.b.[c]" to replace "desired.ko.Spec.a.b[c]" with "<op_name>Latest.ko.spec.a.b[c]"
   // TODO: support for List
   Path string `json:"path"`   
}
```

#### Hook Updates

There will be 3 new hooks added for dealing with late initialized fields

1. `add_late_initialized_fields_override` : The template used in this hook will be used as exact implementation of ResourceDescriptor’s `AddLateInitializedFields()` method

2. `pre_add_late_initialized_fields`: The template used in this hook will be inserted in beginning of the ResourceDescriptor’s `AddLateInitializedFields` method. The variables available to be used in the template will be `desired`, `readOneLatest`, `createLatest`, `updateLatest`, `specUpdated`(equal to false) and `err`(equal to nil)

3. `post_add_late_initialized_fields`: The template used in this hook will be inserted before the last `return specUpdated, err` statement of ResourceDescriptor’s `AddLateInitializedFields()` method. The variables available to be used in the template will be `desired`, `readOneLatest`, `createLatest`, `updateLatest`, `specUpdated` and `err`.


Based on the Requirements above, this mentioned solution will: 

1. Update the desired state of the K8s custom resource with late initialized fields.  ✅

2. Provide configuration for AWS service teams to indicate which operation output returns the late initialized fields. i.e. Get/List/Create/Update output. ✅

3. Wait for asynchronous resource creation. ✅

    > Service teams can add a hook which will look at the state of latest object and determine if the reconciliation loop should run again. If so, they can return an error from the “AddLateInitializedFields()” method and the reconciliation loop will be triggered again.

4. Allow the service teams to provide custom code using hooks for handing of late initialized fields. ✅

### Solution 2 (Preferred)

In this solution, first we will make following **two** changes in the existing reconciler code.

> NOTE: Initially, this solution will only support setting defaults after resource creation. The usecase of handling late initialized fields from Update calls will be added after this solution is implemented.

1. Change `AWSResourceManager.Create` method to **not** just update the status of latest object from the create output response but also update the `latest.ko.Spec` for any optional late initialized fields which were not provided by the k8s user. AWS service teams will provide the details of such fields in generator config.
    > Note: We cannot just consider all fields of `latest.ko.Spec` to be updated from Create output because `latest.ko.Spec` is originally created from Create input and for some AWS services the types of field, even with the same name, do not match in CreateInput and CreateOutput. Ex: apigatewayv2 Integration resource's TLSConfig shape.
    
    > Because of above reason, ACK code generator will rely on AWS service team's input on which spec fields should get populated from the create output.

2. `reconciler.patchResource()` method will be updated to not just update the Status of K8s object but also Spec.
    > The method signature and implementation will be updated to patch either Spec or Status or both based on the input parameters.

Once the existing source code is updated with above two changes, The solution for setting defaults on optional fields will look like following:

1. Introduce a new AWSResourceManager method `SetDefaults(AWSResource) (AWSResource, error)`.

    Based on the inputs provided by AWS service teams in generator config for `LateInitialization`, this method will set the optional fields, which are not present in the desired manifest, with their default values. The source of default values will be provided by AWS service teams in `LateInitialization` config.
    
2. `SetDefaults` method will be invoked after the `r.rm.Create()` call in the reconciler, to set default values after the creation of a resource.

3. The `generator config` and `hooks` from Solution 1 will remain the same in this solution. The `LateInitializationConfig` will tell reconciler which fields should be update with default values and three `hooks` from solution 1 will allow AWS service teams to add custom code for `AWSResourceManager.SetDefaults` method

4. If there is a delta present in `Spec` for `desired` and `latest` object after `Create` and `SetDefaults` calls, reconciler will patch the Spec of k8s object with values present in latest object, otherwise it will just patch the status of the k8s object.

**HOW WOULD SOLUTION 2 HANDLE ASYNC RESOURCE CREATION?**

**Context:** Some custom resources  do not return all the properties as output of Create operation because the resource is being created asynchronously on the server side.

To solve this problem, ACK controller will make use of `metadata.Annotations`, to make sure reconciler makes a call to `AWSResourceManager.SetDefaults` even when resource is not being created in the reconciliation loop.

In this scenario, AWS service teams will add a custom implementation using hooks for `SetDefaults` which will return an error when default values could not be found (resource is still under creation). The reconciler will see this specific error, add an annotation and requeue the request.

After multiple requeues, when SetDefaults successfully returns, the reconciler will remove the annotation and not requeue the reconcile request anymore.
