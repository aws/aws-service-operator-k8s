// Copyright Amazon.com Inc. or its affiliates. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License"). You may
// not use this file except in compliance with the License. A copy of the
// License is located at
//
//     http://aws.amazon.com/apache2.0/
//
// or in the "license" file accompanying this file. This file is distributed
// on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either
// express or implied. See the License for the specific language governing
// permissions and limitations under the License.

package config

// SourceFieldConfig instructs the code generator how to handle a field in the
// Resource's SpecFields/StatusFields collection that takes its value from an
// abnormal source -- in other words, not the Create operation's Input or
// Output shape.
//
// This additional field can source its value from a shape in a different API
// Operation entirely.
//
// The data type (Go type) that a field is assigned during code generation
// depends on whether the field is part of the Create Operation's Input shape
// which go into the Resource's Spec fields collection, or the Create
// Operation's Output shape which, if not present in the Input shape, means the
// field goes into the Resource's Status fields collection).
//
// Each Resource typically also has a ReadOne Operation. The ACK service
// controller will call this ReadOne Operation to get the latest observed state
// of a particular resource in the backend AWS API service. The service
// controller sets the observed Resource's Spec and Status fields from the
// Output shape of the ReadOne Operation. The code generator is responsible for
// producing the Go code that performs these "setter" methods on the Resource.
// The way the code generator determines how to set the Spec or Status fields
// from the Output shape's member fields is by looking at the data type of the
// Spec or Status field with the same name as the Output shape's member field.
//
// Importantly, in producing this "setter" Go code the code generator **assumes
// that the data types (Go types) in the source (the Output shape's member
// field) and target (the Spec or Status field) are the same**.
//
// There are some APIs, however, where the Go type of the field in the Create
// Operation's Input shape is actually different from the same-named field in
// the ReadOne Operation's Output shape. A good example of this is the Lambda
// CreateFunction API call, which has a `Code` member of its Input shape that
// looks like this:
//
// "Code": {
//   "ImageUri": "string",
//   "S3Bucket": "string",
//   "S3Key": "string",
//   "S3ObjectVersion": "string",
//   "ZipFile": blob
// },
//
// The GetFunction API call's Output shape has a same-named field called
// `Code` in it, but this field looks like this:
//
// "Code": {
//   "ImageUri": "string",
//   "Location": "string",
//   "RepositoryType": "string",
//   "ResolvedImageUri": "string"
// },
//
// This presents a conundrum to the ACK code generator, which, as noted above,
// assumes the data types of same-named fields in the Create Operation's Input
// shape and ReadOne Operation's Output shape are the same.
//
// The SourceFieldConfig struct allows us to explain to the code generator
// how to handle situations like this.
//
// For the Lambda Function Resource's `Code` field, we can inform the code
// generator to create three new Status fields (readonly) from the `Location`,
// `RepositoryType` and `ResolvedImageUri` fields in the `Code` member of the
// ReadOne Operation's Output shape:
//
// resources:
//   Function:
//     fields:
//       CodeLocation:
//         is_read_only: true
//         from:
//           operation: GetFunction
//           path: Code.Location
//       CodeRepositoryType:
//         is_read_only: true
//         from:
//           operation: GetFunction
//           path: Code.RepositoryType
//       CodeRegisteredImageURI:
//         is_read_only: true
//         from:
//           operation: GetFunction
//           path: Code.RegisteredImageUri
type SourceFieldConfig struct {
	// Operation refers to the ID of the API Operation where we will
	// determine the field's Go type.
	Operation string `json:"operation"`
	// Path refers to the field path of the member of the Input or Output
	// shape in the Operation identified by OperationID that we will take as
	// our additional spec/status field's value.
	Path string `json:"path"`
}

// FieldConfig contains instructions to the code generator about how
// to interpret the value of an Attribute and how to map it to a CRD's Spec or
// Status field
type FieldConfig struct {
	// IsAttribute informs the code generator that this field is part of an
	// "Attributes Map".
	//
	// Some resources for some service APIs follow a pattern or using an
	// "Attributes" `map[string]*string` that contains real, schema'd fields of
	// the primary resource, and that those fields should be "unpacked" from
	// the raw map and into CRD's Spec and Status struct fields.
	IsAttribute bool `json:"is_attribute"`
	// IsReadOnly indicates the field's value can not be set by a Kubernetes
	// user; in other words, the field should go in the CR's Status struct
	IsReadOnly bool `json:"is_read_only"`
	// IsPrintable determines whether the field should be included in the
	// AdditionalPrinterColumns list to be included in the `kubectl get`
	// response.
	IsPrintable bool `json:"is_printable"`
	// Required indicates whether this field is a required member or not.
	// This field is used to configure '+kubebuilder:validation:Required' on API object's members.
	IsRequired *bool `json:"is_required,omitempty"`
	// IsName indicates the field represents the name/string identifier field
	// for the resource.  This allows the generator config to override the
	// default behaviour of considering a field called "Name" or
	// "{Resource}Name" or "{Resource}Id" as the "name field" for the resource.
	IsName bool `json:"is_name"`
	// IsOwnerAccountID indicates the field contains the AWS Account ID
	// that owns the resource. This is a special field that we direct to
	// storage in the common `Status.ACKResourceMetadata.OwnerAccountID` field.
	IsOwnerAccountID bool `json:"is_owner_account_id"`
	// From instructs the code generator that the value of the field should
	// be retrieved from the specified operation and member path
	From *SourceFieldConfig `json:"from,omitempty"`
	// Type instructs the code generator that the value of the field should
	// be defined per the specified type.
	// For example: From field refers to a string property but a list of strings is desired
	Type string `json:"type,omitempty"`
}
