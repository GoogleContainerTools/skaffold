/*
Copyright 2021 The Skaffold Authors

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
	"strings"

	"github.com/sirupsen/logrus"
	"github.com/spf13/pflag"
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
	modes           []string
	forwardUser     bool
	forwardServices bool
	forwardPods     bool
	forwardDebug    bool
	// compat is true if we're in backwards-compatible mode when --port-forward was boolean
	compat bool
}

var _ pflag.Value = (*PortForwardOptions)(nil)
var _ pflag.SliceValue = (*PortForwardOptions)(nil)

func (p *PortForwardOptions) reset() {
	p.modes = nil
	p.forwardUser = false
	p.forwardServices = false
	p.forwardPods = false
	p.forwardDebug = false
	p.compat = false
}

func (p *PortForwardOptions) Set(csv string) error {
	split := strings.Split(csv, ",")
	return p.Replace(split)
}

func (p *PortForwardOptions) Type() string {
	return "stringSlice"
}

func (p *PortForwardOptions) String() string {
	if len(p.modes) == 0 {
		return off
	}
	return strings.Join(p.modes, ",")
}

func (p *PortForwardOptions) Append(o string) error {
	// check if already present under the assumption that the
	// current set is valid
	for _, v := range p.modes {
		if v == o {
			return nil
		}
	}

	if err := validateModes(append(p.modes, o)); err != nil {
		return err
	}

	// `off` and boolean values must be standalone
	switch o {
	case user:
		p.forwardUser = true
	case services:
		p.forwardServices = true
	case pods:
		p.forwardPods = true
	case debug:
		p.forwardDebug = true
	}
	if b, err := strconv.ParseBool(o); err == nil {
		p.compat = b
	}
	p.modes = append(p.modes, o)
	return nil
}

func (p *PortForwardOptions) Replace(options []string) error {
	if err := validateModes(options); err != nil {
		return err
	}
	p.reset()
	for _, o := range options {
		if err := p.Append(o); err != nil {
			logrus.Fatal(err) // should never happen since we validated the options
		}
	}
	return nil
}

func (p *PortForwardOptions) GetSlice() []string {
	return p.modes
}

// Enabled checks if the port-forwarding options indicates that forwarding should be enabled.
func (p PortForwardOptions) Enabled() bool {
	if len(p.modes) == 0 {
		return false
	}
	// --port-forward was previously a boolean, so accept pflag's boolean values.
	// This method accepts "off" or bool variants to be mixed with others,
	// leaving complaining to be done by Validate.
	for _, o := range p.modes {
		if o == off {
			return false
		}
		b, err := strconv.ParseBool(o)
		if err == nil && !b {
			return false
		}
	}
	return true
}

// validateModes checks that the given set of port-forward modes are ok.
// For example, `off` and boolean values should not be combined with other values.
func validateModes(modes []string) error {
	for _, mode := range modes {
		if err := validateMode(mode); err != nil {
			return err
		}
		// Boolean values (true/false/1/0) and `off` must be used alone.
		if _, err := strconv.ParseBool(mode); len(modes) > 1 && (err == nil || mode == off) {
			return fmt.Errorf("port-forward %q cannot be combined with other options", mode)
		}
	}
	return nil
}

func validateMode(mode string) error {
	if _, err := strconv.ParseBool(mode); err == nil {
		return nil
	}
	switch mode {
	case off, user, services, pods, debug:
		return nil
	default:
		return fmt.Errorf("unknown port-forward option %q: expected: user, services, pods, debug, off", mode)
	}
}

func (p PortForwardOptions) ForwardUser(runMode RunMode) bool {
	// When --port-forward was a boolean option, user-defined port-forwards
	// were enabled all modes.
	return p.forwardUser || p.compat
}

func (p PortForwardOptions) ForwardServices(runMode RunMode) bool {
	// When --port-forward was a boolean option, service forwarding
	// was enabled all modes.
	return p.forwardServices || p.compat
}

func (p PortForwardOptions) ForwardPods(runMode RunMode) bool {
	// Compatibility break: when `--port-forward` was a boolean option,
	// all pods containerPorts were forwarded for `debug`.  But now we
	// only forward debug-related containerPorts.  So we ignore boolean
	// values.
	return p.forwardPods
}

func (p PortForwardOptions) ForwardDebug(runMode RunMode) bool {
	// When --port-forward was a boolean option, all containerPorts were
	// forwarded in debug mode to connect debug-related ports; now we only
	// forward debug-related ports.
	return p.forwardDebug || (p.compat && runMode == RunModes.Debug)
}
