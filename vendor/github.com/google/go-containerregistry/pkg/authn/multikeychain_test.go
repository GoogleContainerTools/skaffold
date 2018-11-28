// Copyright 2018 Google LLC All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package authn

import (
	"testing"

	"github.com/google/go-containerregistry/pkg/name"
)

func TestMultiKeychain(t *testing.T) {
	one := &Basic{Username: "one", Password: "secret"}
	two := &Basic{Username: "two", Password: "secret"}
	three := &Basic{Username: "three", Password: "secret"}

	regOne, _ := name.NewRegistry("one.gcr.io", name.StrictValidation)
	regTwo, _ := name.NewRegistry("two.gcr.io", name.StrictValidation)
	regThree, _ := name.NewRegistry("three.gcr.io", name.StrictValidation)

	tests := []struct {
		name string
		reg  name.Registry
		kc   Keychain
		want Authenticator
	}{{
		// Make sure our test keychain WAI
		name: "simple fixed test (match)",
		reg:  regOne,
		kc:   fixedKeychain{regOne: one},
		want: one,
	}, {
		// Make sure our test keychain WAI
		name: "simple fixed test (no match)",
		reg:  regTwo,
		kc:   fixedKeychain{regOne: one},
		want: Anonymous,
	}, {
		name: "match first keychain",
		reg:  regOne,
		kc: NewMultiKeychain(
			fixedKeychain{regOne: one},
			fixedKeychain{regOne: three, regTwo: two},
		),
		want: one,
	}, {
		name: "match second keychain",
		reg:  regTwo,
		kc: NewMultiKeychain(
			fixedKeychain{regOne: one},
			fixedKeychain{regOne: three, regTwo: two},
		),
		want: two,
	}, {
		name: "match no keychain",
		reg:  regThree,
		kc: NewMultiKeychain(
			fixedKeychain{regOne: one},
			fixedKeychain{regOne: three, regTwo: two},
		),
		want: Anonymous,
	}}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := test.kc.Resolve(test.reg)
			if err != nil {
				t.Errorf("Resolve() = %v", err)
			}
			if got != test.want {
				t.Errorf("Resolve() = %v, wanted %v", got, test.want)
			}
		})
	}
}

type fixedKeychain map[name.Registry]Authenticator

var _ Keychain = (fixedKeychain)(nil)

// Resolve implements Keychain.
func (fk fixedKeychain) Resolve(reg name.Registry) (Authenticator, error) {
	if auth, ok := fk[reg]; ok {
		return auth, nil
	}
	return Anonymous, nil
}
