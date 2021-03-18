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

package config

import (
	"fmt"
	"strconv"
)

// These are the list of accepted port-forward modes.
const (
	// user enables user-defined port-forwards.
	user = "user"
	// services enables forwarding Kubernetes services.
	services = "services"
	// debug enables forwarding just debug-related containerPorts on pods.
	debug = "debug"
	// pods enables forwarding of all containerPorts on pods.
	pods = "pods"
	// off disables port forwarding.
	off = "off"
)

// PortForwardOptions are options set by the command line for port forwarding.
// `off` is intended to be a single standalone option.
type PortForwardOptions struct {
	Modes []string
}

// Enabled checks if the port-forwarding options indicates that forwarding should be enabled.
func (p PortForwardOptions) Enabled() bool {
	if len(p.Modes) == 0 {
		return false
	}
	// --port-forward was previously a boolean, so accept pflag's boolean values.
	// This method accepts "off" or bool variants to be mixed with others,
	// leaving complaining to be done by Validate.
	for _, o := range p.Modes {
		b, err := strconv.ParseBool(o)
		if err == nil && !b {
			return false
		}
		if o == off {
			return false
		}
	}
	return true
}

// Validate checks that the port-forward options are ok.
// For example, `off` and boolean values should not be combined with other values.
func (p PortForwardOptions) Validate() error {
	// Boolean values (true/false/1/0) and `off` must be used alone.
	for _, o := range p.Modes {
		if _, err := strconv.ParseBool(o); err == nil {
			if len(p.Modes) > 1 {
				return fmt.Errorf("port-forward %q cannot be combined with other options", o)
			}
		} else {
			switch o {
			case off:
				if len(p.Modes) > 1 {
					return fmt.Errorf("port-forward %q cannot be combined with other options", o)
				}
			case user, services, pods, debug:
				// continue
			default:
				return fmt.Errorf("unknown port-forward option %q: expected: user, services, pods, debug, off", o)
			}
		}
	}
	return nil
}

func (p PortForwardOptions) ForwardUser(runMode RunMode) bool {
	for _, o := range p.Modes {
		if b, err := strconv.ParseBool(o); err == nil {
			// When --port-forward was a boolean option, user-defined port-forwards
			// were enabled all modes.
			return b
		}
		if o == user {
			return true
		}
	}
	return false
}

func (p PortForwardOptions) ForwardServices(runMode RunMode) bool {
	for _, o := range p.Modes {
		if b, err := strconv.ParseBool(o); err == nil {
			// When --port-forward was a boolean option, service forwarding
			// was enabled all modes.
			return b
		}
		if o == services {
			return true
		}
	}
	return false
}

func (p PortForwardOptions) ForwardPods(runMode RunMode) bool {
	for _, o := range p.Modes {
		// Compatibility break: when `--port-forward` was a boolean option,
		// all pods containerPorts were forwarded for `debug`.  But now we
		// only forward debug-related containerPorts.  So we ignore boolean
		// values.
		if o == pods {
			return true
		}
	}
	return false
}

func (p PortForwardOptions) ForwardDebug(runMode RunMode) bool {
	for _, o := range p.Modes {
		if b, err := strconv.ParseBool(o); err == nil {
			// When --port-forward was a boolean option, all containerPorts were
			// forwarded in debug mode to connect debug-related ports; now we only
			// forward debug-related ports.
			return b && runMode == RunModes.Debug
		}
		if o == debug {
			return true
		}
	}
	return false
}
