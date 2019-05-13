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

package build

import (
	"context"
	"fmt"
	"io"
	"testing"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/testutil"
)

var (
	wordsToInt = map[string]int{
		"one":   1,
		"two":   2,
		"three": 3,
		"four":  4,
		"five":  5,
		"eight": 8,
	}
)

type operator interface {
	doBuild(ctx context.Context, out io.Writer, artifact *latest.Artifact, tag string) (string, error)
}

func newOperator(op string) operator {
	switch op {
	case "sum":
		return &summer{}
	default:
		return &identity{}
	}
}

// summer builder sums all the numbers
type summer struct {
	sum int
}

// Build Calculate the tag based in the sum value.
// For in sequence builds the sum will updated safely and the next images will
// be tagged as sum of values seen so far.
func (s *summer) doBuild(ctx context.Context, out io.Writer, artifact *latest.Artifact, tag string) (string, error) {
	num, ok := wordsToInt[artifact.ImageName]
	if !ok {
		return "", fmt.Errorf("could not build artifact %s", artifact.ImageName)
	}
	// update sum
	s.sum += num
	return fmt.Sprintf("%s@sha256:%d", tag, s.sum), nil
}

// identity build returns the tag as is
type identity struct {
}

// doBuild returns the int value corresponding to the image name
func (i *identity) doBuild(ctx context.Context, out io.Writer, artifact *latest.Artifact, tag string) (string, error) {
	num, ok := wordsToInt[artifact.ImageName]
	if !ok {
		return "", fmt.Errorf("could not build artifact %s", artifact.ImageName)
	}
	return fmt.Sprintf("%s@sha256:%d", tag, num), nil
}

func CheckBuildResults(t *testing.T, expected []Result, actual []Result) {
	// build results are going to be out of order so checking individual fields.
	for _, e := range expected {
		for _, a := range actual {
			if a.Target.ImageName != e.Target.ImageName {
				continue
			}
			checkBuildResult(t, e, a)
		}
	}
}

// CheckBuildResultOrder makes sure builds are completed in order
func CheckBuildResultsOrder(t *testing.T, expected []Result, actual []Result) {
	for i := 0; i < len(expected); i++ {
		checkBuildResult(t, expected[i], actual[i])
	}
}

func checkBuildResult(t *testing.T, e Result, a Result) {
	shouldErr := e.Error != nil
	testutil.CheckError(t, shouldErr, a.Error)
	if shouldErr {
		testutil.CheckErrorContains(t, e.Error.Error(), a.Error)
	}
	testutil.CheckDeepEqual(t, e.Target, a.Target)
	testutil.CheckDeepEqual(t, e.Result, a.Result)
}
