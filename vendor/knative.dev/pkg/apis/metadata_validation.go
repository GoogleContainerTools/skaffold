/*
Copyright 2019 The Knative Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package apis

import (
	"fmt"

	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/validation"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	// CreatorAnnotationSuffix is the suffix of the annotation key to describe
	// the user that created the resource.
	CreatorAnnotationSuffix = "/creator"

	// UpdaterAnnotationSuffix is the suffix of the annotation key to describe
	// the user who last modified the resource.
	UpdaterAnnotationSuffix = "/lastModifier"
)

// ValidateObjectMetadata validates that `metadata` stanza of the
// resources is correct.
func ValidateObjectMetadata(meta metav1.Object) *FieldError {
	name := meta.GetName()
	generateName := meta.GetGenerateName()

	if generateName != "" {
		msgs := validation.NameIsDNS1035Label(generateName, true)

		if len(msgs) > 0 {
			return &FieldError{
				Message: fmt.Sprintf("not a DNS 1035 label prefix: %v", msgs),
				Paths:   []string{"generateName"},
			}
		}
	}

	if name != "" {
		msgs := validation.NameIsDNS1035Label(name, false)

		if len(msgs) > 0 {
			return &FieldError{
				Message: fmt.Sprintf("not a DNS 1035 label: %v", msgs),
				Paths:   []string{"name"},
			}
		}
	}

	if generateName == "" && name == "" {
		return &FieldError{
			Message: "name or generateName is required",
			Paths:   []string{"name"},
		}
	}

	return nil
}

// ValidateCreatorAndModifier validates `metadata.annotation`
func ValidateCreatorAndModifier(oldSpec, newSpec interface{}, oldAnnotation, newAnnotation map[string]string, groupName string) *FieldError {
	var errs *FieldError
	if oldAnnotation[groupName+CreatorAnnotationSuffix] != newAnnotation[groupName+CreatorAnnotationSuffix] {
		errs = errs.Also(&FieldError{
			Message: "annotation value is immutable",
			Paths:   []string{groupName + CreatorAnnotationSuffix},
		})
	}

	if equality.Semantic.DeepEqual(oldSpec, newSpec) && oldAnnotation[groupName+UpdaterAnnotationSuffix] != newAnnotation[groupName+UpdaterAnnotationSuffix] {
		errs = errs.Also(ErrInvalidValue(newAnnotation[groupName+UpdaterAnnotationSuffix], groupName+UpdaterAnnotationSuffix))
	}
	return errs
}
