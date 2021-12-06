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

package local

import (
	"fmt"
	"testing"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build/ko"
	latestV1 "github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest/v1"
	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestNewPerArtifactBuilder(t *testing.T) {
	tests := []struct {
		description     string
		builder         *Builder
		artifact        *latestV1.Artifact
		expectedBuilder artifactBuilder
	}{
		{
			description: "ko",
			builder:     &Builder{},
			artifact: &latestV1.Artifact{
				ArtifactType: latestV1.ArtifactType{
					KoArtifact: &latestV1.KoArtifact{},
				},
			},
			expectedBuilder: &ko.Builder{},
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			builder, err := newPerArtifactBuilder(test.builder, test.artifact)
			t.CheckNoError(err)
			t.CheckDeepEqual(fmt.Sprintf("%T", test.expectedBuilder), fmt.Sprintf("%T", builder))
		})
	}
}
