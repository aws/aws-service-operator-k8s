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

package topic

import (
	"context"

	ackv1alpha1 "github.com/aws/aws-controllers-k8s/apis/core/v1alpha1"
	ackcompare "github.com/aws/aws-controllers-k8s/pkg/compare"
	"github.com/aws/aws-sdk-go/aws"
	svcsdk "github.com/aws/aws-sdk-go/service/sns"
)

// customUpdateTopic implements specialized logic for handling Topic
// resource updates. The SNS SetTopicAttributes API must be called multiple
// times in order to set more than one attribute. This is different from the
// behaviour of the SQS SetQueueAttributes API and even the SNS
// SetPlatformApplicationAttributes API call, both of which allow setting
// multiple attributes in a single API call.
func (rm *resourceManager) customUpdateTopic(
	ctx context.Context,
	desired *resource,
	latest *resource,
	diffReporter *ackcompare.Reporter,
) (*resource, error) {
	var err error
	topicARN := *desired.ko.Status.ACKResourceMetadata.ARN
	if displayNameChanged(desired, latest) {
		attrVal := ""
		if desired.ko.Spec.DisplayName != nil {
			attrVal = *desired.ko.Spec.DisplayName
		}
		rm.log.WithValues(
			"topic_arn", topicARN,
			"attr_name", "DisplayName",
			"attr_value", attrVal,
		).V(1).Info("setting topic attribute")
		err = rm.setTopicAttribute(ctx, topicARN, "DisplayName", attrVal)
		if err != nil {
			return nil, err
		}
	}
	if policyChanged(desired, latest) {
		attrVal := ""
		if desired.ko.Spec.DisplayName != nil {
			attrVal = *desired.ko.Spec.Policy
		}
		rm.log.WithValues(
			"topic_arn", topicARN,
			"attr_name", "Policy",
			"attr_value", attrVal,
		).V(1).Info("setting topic attribute")
		err = rm.setTopicAttribute(ctx, topicARN, "Policy", attrVal)
		if err != nil {
			return nil, err
		}
	}
	return desired, nil
}

// policyChanged returns true if the policy attribute of the supplied desired
// and latest Topic resources is different
func policyChanged(
	desired *resource,
	latest *resource,
) bool {
	dspec := desired.ko.Spec
	lspec := latest.ko.Spec
	if dspec.Policy == nil {
		return lspec.Policy != nil
	}
	if lspec.Policy == nil {
		return true
	}
	dval := *dspec.Policy
	lval := *lspec.Policy
	return dval != lval
}

// displayNameChanged returns true if the display name attribute of the
// supplied desired and latest Topic resources is different
func displayNameChanged(
	desired *resource,
	latest *resource,
) bool {
	dspec := desired.ko.Spec
	lspec := latest.ko.Spec
	if dspec.DisplayName == nil {
		return lspec.DisplayName != nil
	}
	if lspec.DisplayName == nil {
		return true
	}
	dval := *dspec.DisplayName
	lval := *lspec.DisplayName
	return dval != lval
}

// setTopicAttribute calls the SetTopicAttributes SNS API call to set a SINGLE
// attribute on the topic...
func (rm *resourceManager) setTopicAttribute(
	ctx context.Context,
    topicARN ackv1alpha1.AWSResourceName,
    attrName string,
	attrValue string,
) error {
	input := &svcsdk.SetTopicAttributesInput{
		TopicArn: aws.String(string(topicARN)),
        AttributeName: aws.String(attrName),
        AttributeValue: aws.String(attrValue),
	}
	_, err := rm.sdkapi.SetTopicAttributesWithContext(ctx, input)
	return err
}
