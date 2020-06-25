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

package resource

import (
	"testing"

	"github.com/GoogleContainerTools/skaffold/proto"
	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestString(t *testing.T) {
	var tests = []struct {
		description string
		ae          proto.ActionableErr
		expected    string
	}{
		{
			description: "should return error string if error is set",
			ae:          proto.ActionableErr{Message: "some error"},
			expected:    "some error",
		},
		{
			description: "should return error if both details and error are set",
			ae:          proto.ActionableErr{Message: "error happened due to something"},
			expected:    "error happened due to something",
		},
		{
			description: "should return empty string if all empty",
			ae:          proto.ActionableErr{},
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			status := newStatus(test.ae)
			t.CheckDeepEqual(test.expected, status.String())
		})
	}
}

func TestEqual(t *testing.T) {
	var tests = []struct {
		description string
		old         Status
		new         Status
		expected    bool
	}{
		{
			description: "status should be same if error messages are same",
			old:         Status{ae: proto.ActionableErr{ErrCode: 100, Message: "Waiting for 0/1 replicas to be available..."}},
			new:         Status{ae: proto.ActionableErr{ErrCode: 100, Message: "Waiting for 0/1 replicas to be available..."}},
			expected:    true,
		},
		{
			description: "status should be new if error is different",
			old:         Status{ae: proto.ActionableErr{}},
			new:         Status{ae: proto.ActionableErr{ErrCode: 100, Message: "see this error"}},
		},
		{
			description: "status should be new if errcode are different but same message",
			old:         Status{ae: proto.ActionableErr{ErrCode: 100, Message: "see this error"}},
			new:         Status{ae: proto.ActionableErr{ErrCode: 101, Message: "see this error"}},
		},
		{
			description: "status should be new if messages change",
			old:         Status{ae: proto.ActionableErr{ErrCode: 100, Message: "Waiting for 2/2 replicas to be available..."}},
			new:         Status{ae: proto.ActionableErr{ErrCode: 100, Message: "Waiting for 1/2 replicas to be available..."}},
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			t.CheckDeepEqual(test.expected, test.old.Equal(test.new))
		})
	}
}
