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

package registry

import (
	"regexp"
)

const (
	generic = "generic"
	GCR     = "gcr"
)

type Registry interface {
	// Name returns the string representation of the registry
	Name() string

	// Replace replaces the current registry in a given registry name to input registry
	Update(reg Registry) Registry

	// Type returns registry type
	Type() string
}

var (
	prefixRegex = regexp.MustCompile(`(.*\.)?gcr.io/[a-zA-Z0-9-_]+/?`)
)

// GetRegistry takes an input string repo and parses it to return the appropriate registry type
func GetRegistry(repo string) Registry {
	if prefixRegex.MatchString(repo) {
		if reg, err := NewGCRRegistry(repo); err == nil {
			return reg
		}
	}
	// Default: return generic registry type
	return NewGenericRegistry(repo)
}
