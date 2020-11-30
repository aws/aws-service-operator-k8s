// Copyright Amazon.com Inc. or its affiliates. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License"). You may
// not use this file except in compliance with the License. A copy of the
// License is located at
//
//	 http://aws.amazon.com/apache2.0/
//
// or in the "license" file accompanying this file. This file is distributed
// on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either
// express or implied. See the License for the specific language governing
// permissions and limitations under the License.

package generate_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/aws/aws-controllers-k8s/pkg/model"
	"github.com/aws/aws-controllers-k8s/pkg/testutil"
)

func TestSNS_Topic(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)

	g := testutil.NewGeneratorForService(t, "sns")

	crds, err := g.GetCRDs()
	require.Nil(err)

	crd := getCRDByName("Topic", crds)
	require.NotNil(crd)

	assert.Equal("Topic", crd.Names.Camel)
	assert.Equal("topic", crd.Names.CamelLower)
	assert.Equal("topic", crd.Names.Snake)

	// The GetTopicAttributes operation has the following definition:
	//
	//    "GetTopicAttributes":{
	//      "name":"GetTopicAttributes",
	//      "http":{
	//        "method":"POST",
	//        "requestUri":"/"
	//      },
	//      "input":{"shape":"GetTopicAttributesInput"},
	//      "output":{
	//        "shape":"GetTopicAttributesResponse",
	//        "resultWrapper":"GetTopicAttributesResult"
	//      },
	//      "errors":[
	//        {"shape":"InvalidParameterException"},
	//        {"shape":"InternalErrorException"},
	//        {"shape":"NotFoundException"},
	//        {"shape":"AuthorizationErrorException"},
	//        {"shape":"InvalidSecurityException"}
	//      ]
	//    },
	//
	// Where the NotFoundException shape looks like this:
	//
	//    "NotFoundException":{
	//      "type":"structure",
	//      "members":{
	//        "message":{"shape":"string"}
	//      },
	//      "error":{
	//        "code":"NotFound",
	//        "httpStatusCode":404,
	//        "senderFault":true
	//      },
	//      "exception":true
	//    },
	//
	// So, we expect that the normal logic in CRD.ExceptionCode(404)
	// identifies the above as the 404 Not Found error with a code of
	// "NotFound"
	assert.Equal("NotFound", crd.ExceptionCode(404))

	// The SNS Topic API is a little weird. There are Create and Delete
	// operations ("CreateTopic", "DeleteTopic") but there is no ReadOne
	// operation (there is a "GetTopicAttributes" call though) or Update
	// operation (there is a "SetTopicAttributes" call though). And there is a
	// ReadMany operation (ListTopics)
	require.NotNil(crd.Ops)

	assert.NotNil(crd.Ops.Create)
	assert.NotNil(crd.Ops.Delete)
	assert.NotNil(crd.Ops.ReadMany)
	assert.NotNil(crd.Ops.GetAttributes)
	assert.NotNil(crd.Ops.SetAttributes)

	assert.Nil(crd.Ops.ReadOne)
	assert.Nil(crd.Ops.Update)

	specFields := crd.SpecFields
	statusFields := crd.StatusFields

	// The SNS Topic uses an "Attributes" map[string]*string to masquerade
	// real fields. DeliveryPolicy, Policy, KMSMasterKeyID and DisplayName are
	// all examples of these masqueraded fields...
	expSpecFieldCamel := []string{
		"DeliveryPolicy",
		"DisplayName",
		"KMSMasterKeyID",
		"Name",
		"Policy",
		"Tags",
	}
	assert.Equal(expSpecFieldCamel, attrCamelNames(specFields))

	// The SNS Topic uses an "Attributes" map[string]*string to masquerade
	// real fields. EffectiveDeliveryPolicy and Owner are
	// examples of these masqueraded fields that are ReadOnly and thus belong
	// in the Status struct
	expStatusFieldCamel := []string{
		// "TopicARN" is in the output shape for CreateTopic, but it is removed
		// because it is the primary resource object's ARN field and the
		// SDKMapper has identified it as the source for the standard
		// Status.ACKResourceMetadata.ARN field
		"EffectiveDeliveryPolicy",
		"Owner",
	}
	assert.Equal(expStatusFieldCamel, attrCamelNames(statusFields))

	// The input shape for the Create operation is set from a variety of scalar
	// and non-scalar types and the SNS API features an Attributes parameter
	// that is actually a map[string]*string of real field values that are
	// unpacked by the code generator.
	expCreateInput := `
	attrMap := map[string]*string{}
	if r.ko.Spec.DeliveryPolicy != nil {
		attrMap["DeliveryPolicy"] = r.ko.Spec.DeliveryPolicy
	}
	if r.ko.Spec.DisplayName != nil {
		attrMap["DisplayName"] = r.ko.Spec.DisplayName
	}
	if r.ko.Spec.KMSMasterKeyID != nil {
		attrMap["KmsMasterKeyId"] = r.ko.Spec.KMSMasterKeyID
	}
	if r.ko.Spec.Policy != nil {
		attrMap["Policy"] = r.ko.Spec.Policy
	}
	if len(attrMap) > 0 {
		res.SetAttributes(attrMap)
	}
	if r.ko.Spec.Name != nil {
		res.SetName(*r.ko.Spec.Name)
	}
	if r.ko.Spec.Tags != nil {
		f2 := []*svcsdk.Tag{}
		for _, f2iter := range r.ko.Spec.Tags {
			f2elem := &svcsdk.Tag{}
			if f2iter.Key != nil {
				f2elem.SetKey(*f2iter.Key)
			}
			if f2iter.Value != nil {
				f2elem.SetValue(*f2iter.Value)
			}
			f2 = append(f2, f2elem)
		}
		res.SetTags(f2)
	}
`
	assert.Equal(expCreateInput, crd.GoCodeSetInput(model.OpTypeCreate, "r.ko", "res", 1))

	// None of the fields in the Topic resource's CreateTopicInput shape are
	// returned in the CreateTopicOutput shape, so none of them return any Go
	// code for setting a Status struct field to a corresponding Create Output
	// Shape member. However, the returned output shape DOES include the
	// Topic's ARN field (TopicArn), which we should be storing in the
	// ACKResourceMetadata.ARN standardized field
	expCreateOutput := `
	if ko.Status.ACKResourceMetadata == nil {
		ko.Status.ACKResourceMetadata = &ackv1alpha1.ResourceMetadata{}
	}
	if resp.TopicArn != nil {
		arn := ackv1alpha1.AWSResourceName(*resp.TopicArn)
		ko.Status.ACKResourceMetadata.ARN = &arn
	}
`
	assert.Equal(expCreateOutput, crd.GoCodeSetOutput(model.OpTypeCreate, "resp", "ko", 1, false))

	// The input shape for the GetAttributes operation has a single TopicArn
	// field. This field represents the ARN of the primary resource (the Topic
	// itself) and should be set specially from the ACKResourceMetadata.ARN
	// field in the TopicStatus struct
	expGetAttrsInput := `
	if r.ko.Status.ACKResourceMetadata != nil && r.ko.Status.ACKResourceMetadata.ARN != nil {
		res.SetTopicArn(string(*r.ko.Status.ACKResourceMetadata.ARN))
	} else {
		res.SetTopicArn(rm.ARNFromName(*r.ko.Spec.Name))
	}
`
	assert.Equal(expGetAttrsInput, crd.GoCodeGetAttributesSetInput("r.ko", "res", 1))

	// The output shape for the GetAttributes operation contains a single field
	// "Attributes" that must be unpacked into the Topic CRD's Status fields.
	// There are only three attribute keys that are *not* in the Input shape
	// (and thus in the Spec fields). Two of them are the tesource's ARN and
	// AWS Owner account ID, both of which are handled specially.
	expGetAttrsOutput := `
	ko.Spec.DeliveryPolicy = resp.Attributes["DeliveryPolicy"]
	ko.Spec.DisplayName = resp.Attributes["DisplayName"]
	ko.Status.EffectiveDeliveryPolicy = resp.Attributes["EffectiveDeliveryPolicy"]
	ko.Spec.KMSMasterKeyID = resp.Attributes["KmsMasterKeyId"]
	if ko.Status.ACKResourceMetadata == nil {
		ko.Status.ACKResourceMetadata = &ackv1alpha1.ResourceMetadata{}
	}
	tmpOwnerID := ackv1alpha1.AWSAccountID(*resp.Attributes["Owner"])
	ko.Status.ACKResourceMetadata.OwnerAccountID = &tmpOwnerID
	ko.Spec.Policy = resp.Attributes["Policy"]
	tmpARN := ackv1alpha1.AWSResourceName(*resp.Attributes["TopicArn"])
	ko.Status.ACKResourceMetadata.ARN = &tmpARN
`
	assert.Equal(expGetAttrsOutput, crd.GoCodeGetAttributesSetOutput("resp", "ko", 1))

	// The Go code for checking the GetTopicAttributes Input shape's required
	// fields needs to return false when any required field is missing in the
	// corresponding Spec or Status. The GetTopicAttributesInput shape has a
	// required TopicArn field which corresponds to the resource's ARN which is
	// stored in ACKMetadata.ARN, so the primary resource ARN field if
	// condition is a bit special.
	expReqFieldsInShape := `
	return (ko.Status.ACKResourceMetadata == nil || ko.Status.ACKResourceMetadata.ARN == nil)
`
	assert.Equal(
		strings.TrimSpace(expReqFieldsInShape),
		strings.TrimSpace(
			crd.GoCodeRequiredFieldsMissingFromShape(
				model.OpTypeGetAttributes, "ko", 1,
			),
		),
	)
}
