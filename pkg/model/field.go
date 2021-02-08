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

package model

import (
	awssdkmodel "github.com/aws/aws-sdk-go/private/model/api"
	"strings"

	ackgenconfig "github.com/aws/aws-controllers-k8s/pkg/generate/config"
	"github.com/aws/aws-controllers-k8s/pkg/names"
	"github.com/aws/aws-controllers-k8s/pkg/util"
)

// Field represents a single field in the CRD's Spec or Status objects
type Field struct {
	CRD               *CRD
	Names             names.Names
	GoType            string
	GoTypeElem        string
	GoTypeWithPkgName string
	ShapeRef          *awssdkmodel.ShapeRef
	FieldConfig       *ackgenconfig.FieldConfig
}

// IsRequired checks the FieldConfig for Field and returns if the field is
// marked as required or not.A
//
// If there is no required override present for this field in FieldConfig,
// IsRequired will return if the shape is marked as required in AWS SDK Private
// model We use this to append kubebuilder:validation:Required markers to
// validate using the CRD validation schema
func (f *Field) IsRequired() bool {
	if f.FieldConfig != nil && f.FieldConfig.IsRequired != nil {
		return *f.FieldConfig.IsRequired
	}
	return util.InStrings(f.Names.ModelOrginal, f.CRD.Ops.Create.InputRef.Shape.Required)
}

// newField returns a pointer to a new Field object
func newField(
	crd *CRD,
	fieldNames names.Names,
	shapeRef *awssdkmodel.ShapeRef,
	cfg *ackgenconfig.FieldConfig,
) *Field {
	var gte, gt, gtwp string
	var shape *awssdkmodel.Shape
	if shapeRef != nil {
		shape = shapeRef.Shape
	}
	if cfg != nil && len(cfg.Type) > 0 {
		gt = cfg.Type
		if strings.HasPrefix(gt, "[]") {
			gte = strings.TrimPrefix(gt, "[]")
			gtwp = gte
		}
	} else if shape != nil {
		gte, gt, gtwp = cleanGoType(crd.sdkAPI, crd.cfg, shape)
		if cfg != nil && cfg.Type == "list" &&
			shape.Type != "list" {
			gt = "[]" + gt
		}
	} else {
		gte = "string"
		gt = "*string"
		gtwp = "*string"
	}
	return &Field{
		CRD:               crd,
		Names:             fieldNames,
		ShapeRef:          shapeRef,
		GoType:            gt,
		GoTypeElem:        gte,
		GoTypeWithPkgName: gtwp,
		FieldConfig:       cfg,
	}
}
