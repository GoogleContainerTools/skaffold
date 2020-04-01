/*
Copyright 2020 The Skaffold Authors

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

package diag

import (
	"context"
	"fmt"
	"github.com/GoogleContainerTools/skaffold/pkg/diag/validator"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)


type Diagnose struct {
	labels []string
    namespaces []string
    validators []validator.Validator
}

func New(namespaces []string) *Diagnose{
	return &Diagnose{
		namespaces: namespaces,
	}
}

func (d *Diagnose) WithLabels(labels []string) *Diagnose {
	d.labels = labels
	return d
}

func (d *Diagnose) WithValidators(v []validator.Validator) *Diagnose {
	d.validators = v
	return d
}

func (d *Diagnose)Run() ([]validator.Resource, error) {
	res := []validator.Resource{}
	errs := []error{}
	listOptions := metav1.ListOptions{}
	for _, l := range d.labels {
		listOptions = metav1.ListOptions{
			LabelSelector: l,
		}
	}
	for _, v := range d.validators {
		for _, ns := range (d.namespaces) {
		    r, err := v.Validate(context.Background(), ns, listOptions)
		    res = append(res, r...)
		    if err != nil {
			   errs = append(errs, err)
		    }
		}
	}
	if len(errs) == 0{
		return res, nil
	}
	errBuilder := ""
	for _, err := range errs {
		errBuilder = errBuilder + err.Error() +  "\n"
	}
	return res, fmt.Errorf("following errors occurred %s", errBuilder)
}