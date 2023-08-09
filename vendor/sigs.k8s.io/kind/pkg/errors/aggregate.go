/*
Copyright 2019 The Kubernetes Authors.

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

package errors

// NewAggregate is a k8s.io/apimachinery/pkg/util/errors.NewAggregate compatible wrapper
// note that while it returns a StackTrace wrapped Aggregate
// That has been Flattened and Reduced
func NewAggregate(errlist []error) error {
	return WithStack(
		reduce(
			flatten(
				newAggregate(errlist),
			),
		),
	)
}

// Errors returns the deepest Aggregate in a Cause chain
func Errors(err error) []error {
	var errors Aggregate
	for {
		if v, ok := err.(Aggregate); ok {
			errors = v
		}
		if causerErr, ok := err.(Causer); ok {
			err = causerErr.Cause()
		} else {
			break
		}
	}
	if errors != nil {
		return errors.Errors()
	}
	return nil
}
