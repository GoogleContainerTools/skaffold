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

package local

import (
	"testing"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/docker"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/runner/runcontext"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestRetrieveEnv(t *testing.T) {
	testutil.Run(t, "", func(t *testutil.T) {
		extraEnv := []string{"EXTRA_ENV=additional"}
		t.Override(&docker.NewAPIClient, func(*runcontext.RunContext) (docker.LocalDaemon, error) {
			return docker.NewLocalDaemon(&testutil.FakeAPIClient{}, extraEnv, false, nil), nil
		})

		builder, err := NewBuilder(stubRunContext(latest.LocalBuild{}))
		t.CheckNoError(err)

		actual := builder.retrieveExtraEnv()

		t.CheckDeepEqual(extraEnv, actual)
	})
}
