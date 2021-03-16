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
	"testing"

	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestPortForwardOptions_Validate(t *testing.T) {
	tests := []struct {
		name      string
		modes     []string
		shouldErr bool
	}{
		{name: "off", modes: nil, shouldErr: false},
		{name: "compat", modes: []string{"compat"}, shouldErr: false},
		{name: "off", modes: []string{"off"}, shouldErr: false},
		{name: "true", modes: []string{"true"}, shouldErr: false},
		{name: "false", modes: []string{"false"}, shouldErr: false},
		{name: "user,debug,pods,services", modes: []string{"user", "debug", "pods", "services"}, shouldErr: false},
		{name: "user,compat,debug", modes: []string{"user", "compat", "debug"}, shouldErr: true},
		{name: "off,debug", modes: []string{"off", "debug"}, shouldErr: true},
		{name: "pods,false", modes: []string{"pods", "false"}, shouldErr: true},
	}
	for _, test := range tests {
		testutil.Run(t, test.name, func(t *testutil.T) {
			err := PortForwardOptions{Modes: test.modes}.Validate()
			t.CheckError(test.shouldErr, err)
		})
	}
}

func TestPortForwardOptions_Forwards(t *testing.T) {
	tests := []struct {
		name            string
		runModes        []RunMode // if empty, then all of Deploy, Dev, Debug, Run
		modes           []string
		forwardUser     bool
		forwardServices bool
		forwardPods     bool
		forwardDebug    bool
	}{
		{name: "nil", modes: nil},                 // all disabled
		{name: "off", modes: []string{"off"}},     // all disabled
		{name: "false", modes: []string{"false"}}, // all disabled
		{name: "compat - deploy, dev, run", modes: []string{"compat"}, runModes: []RunMode{RunModes.Deploy, RunModes.Run, RunModes.Dev}, forwardUser: true, forwardServices: true},
		{name: "compat - debug", modes: []string{"compat"}, runModes: []RunMode{RunModes.Debug}, forwardUser: true, forwardServices: true, forwardDebug: true},
		{name: "true - deploy, dev, run", modes: []string{"true"}, runModes: []RunMode{RunModes.Deploy, RunModes.Run, RunModes.Dev}, forwardUser: true, forwardServices: true},
		{name: "true - debug", modes: []string{"true"}, runModes: []RunMode{RunModes.Debug}, forwardUser: true, forwardServices: true, forwardDebug: true},

		{name: "user,debug,pods,services", modes: []string{"user", "debug", "pods", "services"}, forwardUser: true, forwardServices: true, forwardPods: true, forwardDebug: true},
		{name: "user", modes: []string{"user"}, forwardUser: true},
		{name: "services", modes: []string{"services"}, forwardServices: true},
		{name: "pods", modes: []string{"pods"}, forwardPods: true},
		{name: "debug", modes: []string{"debug"}, forwardDebug: true},
	}
	for _, test := range tests {
		testutil.Run(t, test.name, func(t *testutil.T) {
			opts := PortForwardOptions{Modes: test.modes}
			t.CheckError(false, opts.Validate())
			runModes := test.runModes
			if len(runModes) == 0 {
				runModes = []RunMode{RunModes.Deploy, RunModes.Run, RunModes.Dev, RunModes.Debug}
			}
			for _, rm := range runModes {
				t.CheckDeepEqual(test.forwardUser, opts.ForwardUser(rm))
				t.CheckDeepEqual(test.forwardServices, opts.ForwardServices(rm))
				t.CheckDeepEqual(test.forwardPods, opts.ForwardPods(rm))
				t.CheckDeepEqual(test.forwardDebug, opts.ForwardDebug(rm))
			}
		})
	}
}
