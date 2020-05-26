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

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"

	"github.com/GoogleContainerTools/skaffold/pkg/diag/validator"
)

type Diagnose interface {
	Run(ctx context.Context) ([]validator.Resource, error)
	WithLabel(key, value string) Diagnose
	WithValidators(v []validator.Validator) Diagnose
}

type diag struct {
	namespaces []string
	labels     map[string]string
	validators []validator.Validator
}

func New(namespaces []string) Diagnose {
	var ns []string
	for _, n := range namespaces {
		if n != "" {
			ns = append(ns, n)
		}
	}
	return &diag{
		namespaces: ns,
		labels:     map[string]string{},
	}
}

func (d *diag) WithLabel(key, value string) Diagnose {
	d.labels[key] = value
	return d
}

func (d *diag) WithValidators(v []validator.Validator) Diagnose {
	d.validators = v
	return d
}

func (d *diag) Run(ctx context.Context) ([]validator.Resource, error) {
	var (
		res  []validator.Resource
		errs []error
	)
	// get selector from labels
	selector := labels.SelectorFromSet(d.labels)
	listOptions := metav1.ListOptions{
		LabelSelector: selector.String(),
	}

	for _, v := range d.validators {
		for _, ns := range d.namespaces {
			r, err := v.Validate(ctx, ns, listOptions)
			res = append(res, r...)
			if err != nil {
				errs = append(errs, err)
			}
		}
	}
	if len(errs) == 0 {
		return res, nil
	}

	errBuilder := ""
	for _, err := range errs {
		errBuilder = errBuilder + err.Error() + "\n"
	}

	return res, fmt.Errorf("following errors occurred %s", errBuilder)
}
