/*
Copyright 2019 The Tekton Authors

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

package v1alpha1

import "context"

// hdcnKey is used as the key for associating information
// with a context.Context.
type hdcnKey struct{}

// WithDefaultConfigurationName notes on the context for nested validation
// that there is a default configuration name, which affects how an empty
// configurationName is validated.
func WithDefaultConfigurationName(ctx context.Context) context.Context {
	return context.WithValue(ctx, hdcnKey{}, struct{}{})
}
