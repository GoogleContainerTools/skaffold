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

// Matcher is used by Replacer to match object blob to replace based
// on a manifest key in the Manifest
// Note: If the manifest key is not present, the replacer will replace.
type Matcher interface {
	IsMatchKey(key interface{}) bool
	Matches(v interface{}) bool
}

// anyMatcher matches any object in the yaml manifest.
type anyMatcher struct{}

func (f anyMatcher) Matches(interface{}) bool {
	return true
}

func (f anyMatcher) IsMatchKey(key interface{}) bool {
	return false
}

// ReplaceAny is a Replacer which returns anyMatcher.
type ReplaceAny struct {
	m Matcher
}

func (r ReplaceAny) ObjMatcher() Matcher {
	if r.m == nil {
		r.m = anyMatcher{}
	}
	return r.m
}
