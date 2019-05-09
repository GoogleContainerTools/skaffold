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
	"time"

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

// StaggerBuild function stalls the build by adding a sleep for the number of
// milliseconds passed in as `artifact.ImageName`
func StaggerBuild(ctx context.Context, out io.Writer, artifact *latest.Artifact, tag string) (string, error) {
	num, ok := wordsToInt[artifact.ImageName]
	if !ok {
		return "", fmt.Errorf("could not build artifact %s", artifact.ImageName)
	}
	time.Sleep(time.Duration(num) * time.Millisecond)
	return fmt.Sprintf("%s@sha256:abac", tag), nil
}

func CheckBuildResults(t *testing.T, expected []Result, actual []Result) {
	// build results are going to be out of order so
	// exported field.
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
