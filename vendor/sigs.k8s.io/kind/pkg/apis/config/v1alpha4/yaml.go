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

package v1alpha4

import (
	"strings"

	"sigs.k8s.io/kind/pkg/errors"
)

/*
Custom YAML (de)serialization for these types
*/

// UnmarshalYAML implements custom decoding YAML
// https://godoc.org/gopkg.in/yaml.v3
func (m *Mount) UnmarshalYAML(unmarshal func(interface{}) error) error {
	// first unmarshal in the alias type (to avoid a recursion loop on unmarshal)
	type MountAlias Mount
	var a MountAlias
	if err := unmarshal(&a); err != nil {
		return err
	}
	// now handle propagation
	switch a.Propagation {
	case "": // unset, will be defaulted
	case MountPropagationNone:
	case MountPropagationHostToContainer:
	case MountPropagationBidirectional:
	default:
		return errors.Errorf("Unknown MountPropagation: %q", a.Propagation)
	}
	// and copy over the fields
	*m = Mount(a)
	return nil
}

// UnmarshalYAML implements custom decoding YAML
// https://godoc.org/gopkg.in/yaml.v3
func (p *PortMapping) UnmarshalYAML(unmarshal func(interface{}) error) error {
	// first unmarshal in the alias type (to avoid a recursion loop on unmarshal)
	type PortMappingAlias PortMapping
	var a PortMappingAlias
	if err := unmarshal(&a); err != nil {
		return err
	}
	// now handle the protocol field
	a.Protocol = PortMappingProtocol(strings.ToUpper(string(a.Protocol)))
	switch a.Protocol {
	case "": // unset, will be defaulted
	case PortMappingProtocolTCP:
	case PortMappingProtocolUDP:
	case PortMappingProtocolSCTP:
	default:
		return errors.Errorf("Unknown PortMappingProtocol: %q", a.Protocol)
	}
	// and copy over the fields
	*p = PortMapping(a)
	return nil
}
