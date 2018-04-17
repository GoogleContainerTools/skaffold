/*
Copyright 2018 Google LLC

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

package cmd

import (
	"testing"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestReadConfiguration(t *testing.T) {
	localFile, teardown := testutil.TempFile(t, "skaffold.yaml", []byte("local file"))
	defer teardown()

	remoteFile, teardown := testutil.ServeFile(t, []byte("remote file"))
	defer teardown()

	var tests = []struct {
		filename    string
		expectedCfg []byte
		shouldErr   bool
	}{
		{
			filename:  "",
			shouldErr: true,
		},
		{
			filename:    localFile,
			expectedCfg: []byte("local file"),
			shouldErr:   false,
		},
		{
			filename:    remoteFile,
			expectedCfg: []byte("remote file"),
			shouldErr:   false,
		},
	}

	for _, test := range tests {
		cfg, err := util.ReadConfiguration(test.filename)

		testutil.CheckErrorAndDeepEqual(t, test.shouldErr, err, test.expectedCfg, cfg)
	}
}
