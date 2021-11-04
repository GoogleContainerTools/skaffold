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

package manifest

import "regexp"

type GroupKindSelector interface {
	Matches(group, kind string) bool
}

type wildcardGroupKind struct {
	Group *regexp.Regexp
	Kind  *regexp.Regexp
}

func (w *wildcardGroupKind) Matches(group, kind string) bool {
	return (w.Group == nil || w.Group.Match([]byte(group))) && (w.Kind == nil || w.Kind.Match([]byte(kind)))
}

// ConfigConnectorResourceSelector provides a resource selector for Google Cloud Config Connector resources
// See https://cloud.google.com/config-connector/docs/overview
var ConfigConnectorResourceSelector = []GroupKindSelector{
	// add preliminary support for config connector services; group name is currently in flux
	&wildcardGroupKind{Group: regexp.MustCompile(`([[:alpha:]]+\.)+cnrm\.cloud\.google\.com`)},
}
