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

func TestSQS_Queue(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)

	g := testutil.NewGeneratorForService(t, "sqs")

	crds, err := g.GetCRDs()
	require.Nil(err)

	crd := getCRDByName("Queue", crds)
	require.NotNil(crd)

	assert.Equal("Queue", crd.Names.Camel)
	assert.Equal("queue", crd.Names.CamelLower)
	assert.Equal("queue", crd.Names.Snake)

	// The SQS Queue API has CD+L operations:
	//
	// * CreateQueue
	// * DeleteQueue
	// * ListQueues
	require.NotNil(crd.Ops)

	assert.NotNil(crd.Ops.Create)
	assert.NotNil(crd.Ops.ReadMany)
	assert.NotNil(crd.Ops.Delete)
	assert.NotNil(crd.Ops.GetAttributes)
	assert.NotNil(crd.Ops.SetAttributes)

	// But sadly, has no Update or ReadOne operation :(
	// There is, however, GetQueueUrl and GetQueueAttributes calls...
	assert.Nil(crd.Ops.ReadOne)
	assert.Nil(crd.Ops.Update)

	specFields := crd.SpecFields
	statusFields := crd.StatusFields

	expSpecFieldCamel := []string{
		"ContentBasedDeduplication",
		"DelaySeconds",
		"FifoQueue",
		"KMSDataKeyReusePeriodSeconds",
		"KMSMasterKeyID",
		"MaximumMessageSize",
		"MessageRetentionPeriod",
		"Policy",
		"QueueName",
		"ReceiveMessageWaitTimeSeconds",
		"RedrivePolicy",
		"Tags",
		"VisibilityTimeout",
	}
	assert.Equal(expSpecFieldCamel, attrCamelNames(specFields))

	expStatusFieldCamel := []string{
		// There are a set of Attribute map keys that are readonly
		// fields...
		"CreatedTimestamp",
		"LastModifiedTimestamp",
		// There is only a QueueURL field returned from CreateQueueResult shape
		"QueueURL",
	}
	assert.Equal(expStatusFieldCamel, attrCamelNames(statusFields))

	expCreateInput := `
	attrMap := map[string]*string{}
	if r.ko.Spec.ContentBasedDeduplication != nil {
		attrMap["ContentBasedDeduplication"] = r.ko.Spec.ContentBasedDeduplication
	}
	if r.ko.Spec.DelaySeconds != nil {
		attrMap["DelaySeconds"] = r.ko.Spec.DelaySeconds
	}
	if r.ko.Spec.FifoQueue != nil {
		attrMap["FifoQueue"] = r.ko.Spec.FifoQueue
	}
	if r.ko.Spec.KMSDataKeyReusePeriodSeconds != nil {
		attrMap["KmsDataKeyReusePeriodSeconds"] = r.ko.Spec.KMSDataKeyReusePeriodSeconds
	}
	if r.ko.Spec.KMSMasterKeyID != nil {
		attrMap["KmsMasterKeyId"] = r.ko.Spec.KMSMasterKeyID
	}
	if r.ko.Spec.MaximumMessageSize != nil {
		attrMap["MaximumMessageSize"] = r.ko.Spec.MaximumMessageSize
	}
	if r.ko.Spec.MessageRetentionPeriod != nil {
		attrMap["MessageRetentionPeriod"] = r.ko.Spec.MessageRetentionPeriod
	}
	if r.ko.Spec.Policy != nil {
		attrMap["Policy"] = r.ko.Spec.Policy
	}
	if r.ko.Spec.ReceiveMessageWaitTimeSeconds != nil {
		attrMap["ReceiveMessageWaitTimeSeconds"] = r.ko.Spec.ReceiveMessageWaitTimeSeconds
	}
	if r.ko.Spec.RedrivePolicy != nil {
		attrMap["RedrivePolicy"] = r.ko.Spec.RedrivePolicy
	}
	if r.ko.Spec.VisibilityTimeout != nil {
		attrMap["VisibilityTimeout"] = r.ko.Spec.VisibilityTimeout
	}
	if len(attrMap) > 0 {
		res.SetAttributes(attrMap)
	}
	if r.ko.Spec.QueueName != nil {
		res.SetQueueName(*r.ko.Spec.QueueName)
	}
	if r.ko.Spec.Tags != nil {
		f2 := map[string]*string{}
		for f2key, f2valiter := range r.ko.Spec.Tags {
			var f2val string
			f2val = *f2valiter
			f2[f2key] = &f2val
		}
		res.SetTags(f2)
	}
`
	assert.Equal(expCreateInput, crd.GoCodeSetInput(model.OpTypeCreate, "r.ko", "res", 1))

	// There are no fields other than QueueID in the returned CreateQueueResult
	// shape
	expCreateOutput := `
	if resp.QueueUrl != nil {
		ko.Status.QueueURL = resp.QueueUrl
	}
`
	assert.Equal(expCreateOutput, crd.GoCodeSetOutput(model.OpTypeCreate, "resp", "ko", 1, false))

	// The input shape for the GetAttributes operation technically has two
	// fields in it: an AttributeNames list of attribute keys to file
	// attributes for and a QueueUrl field. We only care about the QueueUrl
	// field, since we look for all attributes for a queue.
	expGetAttrsInput := `
	{
		tmpVals := []*string{}
		tmpVal0 := "All"
		tmpVals = append(tmpVals, &tmpVal0)
		res.SetAttributeNames(tmpVals)
	}
	if r.ko.Status.QueueURL != nil {
		res.SetQueueUrl(*r.ko.Status.QueueURL)
	}
`
	assert.Equal(expGetAttrsInput, crd.GoCodeGetAttributesSetInput("r.ko", "res", 1))

	// The output shape for the GetAttributes operation contains a single field
	// "Attributes" that must be unpacked into the Queue CRD's Status fields.
	// There are only three attribute keys that are *not* in the Input shape
	// (and thus in the Spec fields). One of them is the resource's ARN which
	// is handled specially.
	expGetAttrsOutput := `
	ko.Spec.ContentBasedDeduplication = resp.Attributes["ContentBasedDeduplication"]
	ko.Status.CreatedTimestamp = resp.Attributes["CreatedTimestamp"]
	ko.Spec.DelaySeconds = resp.Attributes["DelaySeconds"]
	ko.Spec.FifoQueue = resp.Attributes["FifoQueue"]
	ko.Spec.KMSDataKeyReusePeriodSeconds = resp.Attributes["KmsDataKeyReusePeriodSeconds"]
	ko.Spec.KMSMasterKeyID = resp.Attributes["KmsMasterKeyId"]
	ko.Status.LastModifiedTimestamp = resp.Attributes["LastModifiedTimestamp"]
	ko.Spec.MaximumMessageSize = resp.Attributes["MaximumMessageSize"]
	ko.Spec.MessageRetentionPeriod = resp.Attributes["MessageRetentionPeriod"]
	ko.Spec.Policy = resp.Attributes["Policy"]
	if ko.Status.ACKResourceMetadata == nil {
		ko.Status.ACKResourceMetadata = &ackv1alpha1.ResourceMetadata{}
	}
	tmpARN := ackv1alpha1.AWSResourceName(*resp.Attributes["QueueArn"])
	ko.Status.ACKResourceMetadata.ARN = &tmpARN
	ko.Spec.ReceiveMessageWaitTimeSeconds = resp.Attributes["ReceiveMessageWaitTimeSeconds"]
	ko.Spec.RedrivePolicy = resp.Attributes["RedrivePolicy"]
	ko.Spec.VisibilityTimeout = resp.Attributes["VisibilityTimeout"]
`
	assert.Equal(expGetAttrsOutput, crd.GoCodeGetAttributesSetOutput("resp", "ko", 1))

	expRequiredFieldsCode := `
	return r.ko.Status.QueueURL == nil
`
	gotCode := crd.GoCodeRequiredFieldsMissingFromShape(model.OpTypeGetAttributes, "r.ko", 1)
	assert.Equal(
		strings.TrimSpace(expRequiredFieldsCode),
		strings.TrimSpace(gotCode),
	)
}
