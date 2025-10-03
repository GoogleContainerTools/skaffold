// Copyright 2025 The Tessera authors. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package tessera

import (
	"bufio"
	"bytes"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"maps"

	f_note "github.com/transparency-dev/formats/note"
	"golang.org/x/mod/sumdb/note"
)

// policyComponent describes a component that makes up a policy. This is either a
// single Witness, or a WitnessGroup.
type policyComponent interface {
	// Satisfied returns true if the checkpoint is signed by the quorum of
	// witnesses involved in this policy component.
	Satisfied(cp []byte) bool

	// Endpoints returns the details required for updating a witness and checking the
	// response. The returned result is a map from the URL that should be used to update
	// the witness with a new checkpoint, to the value which is the verifier to check
	// the response is well formed.
	Endpoints() map[string]note.Verifier
}

// NewWitnessGroupFromPolicy creates a graph of witness objects that represents the
// policy provided, and which can be passed directly to the WithWitnesses
// appender lifecycle option.
//
// The policy must be structured as per the description in
// https://git.glasklar.is/sigsum/core/sigsum-go/-/blob/main/doc/policy.md
func NewWitnessGroupFromPolicy(p []byte) (WitnessGroup, error) {
	scanner := bufio.NewScanner(bytes.NewBuffer(p))
	components := make(map[string]policyComponent)

	var quorumName string
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if i := strings.Index(line, "#"); i >= 0 {
			line = line[:i]
		}
		if line == "" {
			continue
		}

		switch fields := strings.Fields(line); fields[0] {
		case "log":
			// This keyword is important to clients who might use the policy file, but we don't need to know about it since
			// we _are_ the log, so just ignore it.
		case "witness":
			// Strictly, the URL is optional so policy files can be used client-side, where they don't care about the URL.
			// Given this function is parsing to create the graph structure which will be used by a Tessera log to witness
			// new checkpoints we'll ignore that special case here.
			if len(fields) != 4 {
				return WitnessGroup{}, fmt.Errorf("invalid witness definition: %q", line)
			}
			name, vkey, witnessURLStr := fields[1], fields[2], fields[3]
			if isBadName(name) {
				return WitnessGroup{}, fmt.Errorf("invalid witness name %q", name)
			}
			if _, ok := components[name]; ok {
				return WitnessGroup{}, fmt.Errorf("duplicate component name: %q", name)
			}
			witnessURL, err := url.Parse(witnessURLStr)
			if err != nil {
				return WitnessGroup{}, fmt.Errorf("invalid witness URL %q: %w", witnessURLStr, err)
			}
			w, err := NewWitness(vkey, witnessURL)
			if err != nil {
				return WitnessGroup{}, fmt.Errorf("invalid witness config %q: %w", line, err)
			}
			components[name] = w
		case "group":
			if len(fields) < 3 {
				return WitnessGroup{}, fmt.Errorf("invalid group definition: %q", line)
			}

			name, N, childrenNames := fields[1], fields[2], fields[3:]
			if isBadName(name) {
				return WitnessGroup{}, fmt.Errorf("invalid group name %q", name)
			}
			if _, ok := components[name]; ok {
				return WitnessGroup{}, fmt.Errorf("duplicate component name: %q", name)
			}
			var n int
			switch N {
			case "any":
				n = 1
			case "all":
				n = len(childrenNames)
			default:
				i, err := strconv.ParseUint(N, 10, 8)
				if err != nil {
					return WitnessGroup{}, fmt.Errorf("invalid threshold %q for group %q: %w", N, name, err)
				}
				n = int(i)
			}
			if c := len(childrenNames); n > c {
				return WitnessGroup{}, fmt.Errorf("group with %d children cannot have threshold %d", c, n)
			}

			children := make([]policyComponent, len(childrenNames))
			for i, cName := range childrenNames {
				if isBadName(cName) {
					return WitnessGroup{}, fmt.Errorf("invalid component name %q", cName)
				}
				child, ok := components[cName]
				if !ok {
					return WitnessGroup{}, fmt.Errorf("unknown component %q in group definition", cName)
				}
				children[i] = child
			}
			wg := NewWitnessGroup(n, children...)
			components[name] = wg
		case "quorum":
			if len(fields) != 2 {
				return WitnessGroup{}, fmt.Errorf("invalid quorum definition: %q", line)
			}
			quorumName = fields[1]
		default:
			return WitnessGroup{}, fmt.Errorf("unknown keyword: %q", fields[0])
		}
	}
	if err := scanner.Err(); err != nil {
		return WitnessGroup{}, err
	}

	switch quorumName {
	case "":
		return WitnessGroup{}, fmt.Errorf("policy file must define a quorum")
	case "none":
		return NewWitnessGroup(0), nil
	default:
		if isBadName(quorumName) {
			return WitnessGroup{}, fmt.Errorf("invalid quorum name %q", quorumName)
		}
		policy, ok := components[quorumName]
		if !ok {
			return WitnessGroup{}, fmt.Errorf("quorum component %q not found", quorumName)
		}
		wg, ok := policy.(WitnessGroup)
		if !ok {
			// A single witness can be a policy. Wrap it in a group.
			return NewWitnessGroup(1, policy), nil
		}
		return wg, nil
	}
}

var keywords = map[string]struct{}{
	"witness": {},
	"group":   {},
	"any":     {},
	"all":     {},
	"none":    {},
	"quorum":  {},
	"log":     {},
}

func isBadName(n string) bool {
	_, isKeyword := keywords[n]
	return isKeyword
}

// NewWitness returns a Witness given a verifier key and the root URL for where this
// witness can be reached.
func NewWitness(vkey string, witnessRoot *url.URL) (Witness, error) {
	v, err := f_note.NewVerifierForCosignatureV1(vkey)
	if err != nil {
		return Witness{}, err
	}

	u := witnessRoot.JoinPath("/add-checkpoint")

	return Witness{
		Key: v,
		URL: u.String(),
	}, err
}

// Witness represents a single witness that can be reached in order to perform a witnessing operation.
// The URLs() method returns the URL where it can be reached for witnessing, and the Satisfied method
// provides a predicate to check whether this witness has signed a checkpoint.
type Witness struct {
	Key note.Verifier
	URL string
}

// Satisfied returns true if the checkpoint provided is signed by this witness.
// This will return false if there is no signature, and also if the
// checkpoint cannot be read as a valid note. It is up to the caller to ensure
// that the input value represents a valid note.
func (w Witness) Satisfied(cp []byte) bool {
	n, err := note.Open(cp, note.VerifierList(w.Key))
	if err != nil {
		return false
	}
	return len(n.Sigs) == 1
}

// Endpoints returns the details required for updating a witness and checking the
// response. The returned result is a map from the URL that should be used to update
// the witness with a new checkpoint, to the value which is the verifier to check
// the response is well formed.
func (w Witness) Endpoints() map[string]note.Verifier {
	return map[string]note.Verifier{w.URL: w.Key}
}

// NewWitnessGroup creates a grouping of Witness or WitnessGroup with a configurable threshold
// of these sub-components that need to be satisfied in order for this group to be satisfied.
//
// The threshold should only be set to less than the number of sub-components if these are
// considered fungible.
func NewWitnessGroup(n int, children ...policyComponent) WitnessGroup {
	if n < 0 || n > len(children) {
		panic(fmt.Errorf("threshold of %d outside bounds for children %s", n, children))
	}
	return WitnessGroup{
		Components: children,
		N:          n,
	}
}

// WitnessGroup defines a group of witnesses, and a threshold of
// signatures that must be met for this group to be satisfied.
// Witnesses within a group should be fungible, e.g. all of the Armored
// Witness devices form a logical group, and N should be picked to
// represent a threshold of the quorum. For some users this will be a
// simple majority, but other strategies are available.
// N must be <= len(WitnessKeys).
type WitnessGroup struct {
	Components []policyComponent
	N          int
}

// Satisfied returns true if the checkpoint provided has sufficient signatures
// from the witnesses in this group to satisfy the threshold.
// This will return false if there are insufficient signatures, and also if the
// checkpoint cannot be read as a valid note. It is up to the caller to ensure
// that the input value represents a valid note.
//
// The implementation of this requires every witness in the group to verify the
// checkpoint, which is O(N). If this is called every time a witness returns a
// checkpoint then this algorithm is O(N^2). To support large N, this may require
// some rewriting in order to maintain performance.
func (wg WitnessGroup) Satisfied(cp []byte) bool {
	if wg.N <= 0 {
		return true
	}
	satisfaction := 0
	for _, c := range wg.Components {
		if c.Satisfied(cp) {
			satisfaction++
		}
		if satisfaction >= wg.N {
			return true
		}
	}
	return false
}

// Endpoints returns the details required for updating a witness and checking the
// response. The returned result is a map from the URL that should be used to update
// the witness with a new checkpoint, to the value which is the verifier to check
// the response is well formed.
func (wg WitnessGroup) Endpoints() map[string]note.Verifier {
	endpoints := make(map[string]note.Verifier)
	for _, c := range wg.Components {
		maps.Copy(endpoints, c.Endpoints())
	}
	return endpoints
}
