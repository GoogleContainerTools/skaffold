/*
Copyright 2019 The Skaffold Authors

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

package kubectl

// Ignorer is used by Replacer to ignore object blob to replace based
// on a manifest key in the Manifest
// Note: If the manifest key is not present, the blob will not be ignored.
type Ignorer interface {
	MatchesKey(key string) bool
	Ignore(v interface{}) bool
}

// ReplaceAny is a Replacer which returns nil Ignorer.
type ReplaceAny struct{}

func (r ReplaceAny) ObjIgnorer() Ignorer {
	return nil
}
