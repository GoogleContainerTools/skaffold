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

// PortForwardOptions are options set by the command line for port forwarding
// with additional configuration information as well
type PortForwardOptions struct {
	Modes []string
}

func (p PortForwardOptions) Enabled() bool {
	if len(p.Modes) == 0 {
		return false
	}
	// treat "none" or bool variant of "false" as disabled
	for _, v := range p.Modes {
		// --port-forward was previously a boolean, so accept pflag's boolean values.
		//  But treat "user,false" -> "false"
		b, err := strconv.ParseBool(string(v))
		if err == nil && !b {
			return false
		}
		switch v {
		case "none":
			return false
		}
	}
	return true
}

// Validate checks that the port-forward options are ok.
// For example, `none` is not mixed with other values.
func (p PortForwardOptions) Validate() error {
	// boolean values (true/false/1/0), `compat`, and `none` must be used alone
	for _, v := range p.Modes {
		if _, err := strconv.ParseBool(v); err == nil {
			if len(p.Modes) > 1 {
				return fmt.Errorf("port-forward %q cannot be combined with other options", v)
			}
		} else {
			switch v {
			case "none", "compat":
				if len(p.Modes) > 1 {
					return fmt.Errorf("port-forward %q cannot be combined with other options", v)
				}
			case "user", "services", "pods", "debug":
				// continue
			default:
				return fmt.Errorf("unknown port-forward option %q: expected: user, services, pods, debug, none", v)
			}
		}
	}
	return nil
}

func (p PortForwardOptions) ForwardUser(runMode RunMode) bool {
	for _, o := range p.Modes {
		if b, err := strconv.ParseBool(string(o)); err == nil && b {
			o = "compat"
		}
		switch o {
		// when --port-forward as a bool option, all modes forwarded user-defines
		case "user", "compat":
			return true
		}
	}
	return false
}

func (p PortForwardOptions) ForwardServices(runMode RunMode) bool {
	for _, o := range p.Modes {
		if b, err := strconv.ParseBool(string(o)); err == nil && b {
			o = "compat"
		}
		switch o {
		// when --port-forward as a bool option, all modes forward services port-forwards
		case "services", "compat":
			return true
		}
	}
	return false
}

func (p PortForwardOptions) ForwardPods(runMode RunMode) bool {
	for _, o := range p.Modes {
		if b, err := strconv.ParseBool(string(o)); err == nil && b {
			o = "compat"
		}
		// compatibility break: when `--port-forward` was a boolean,
		// pods were forwarded for `debug`.
		switch o {
		case "pods":
			return true
		}
	}
	return false
}

func (p PortForwardOptions) ForwardDebug(runMode RunMode) bool {
	for _, o := range p.Modes {
		if b, err := strconv.ParseBool(string(o)); err == nil && b {
			o = "compat"
		}
		switch o {
		case "debug":
			return true
		// when --port-forward was a bool option, debug container ports were forwarded
		case "compat":
			return runMode == RunModes.Debug
		}
	}
	return false
}
