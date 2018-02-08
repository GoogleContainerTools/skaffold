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

package tag

import (
	"errors"
	"testing"

	"github.com/GoogleCloudPlatform/skaffold/pkg/skaffold/util"
	"github.com/GoogleCloudPlatform/skaffold/testutil"
)

func TestGitCommit_GenerateFullyQualifiedImageName(t *testing.T) {

	tests := []struct {
		name    string
		want    string
		wantErr bool
		command util.Command
	}{
		{
			name:    "success",
			command: testutil.NewFakeRunCommand("somecommit", "", nil),
			want:    "somecommit",
			wantErr: false,
		},
		{
			name:    "failure",
			command: testutil.NewFakeRunCommand("", "", errors.New("error")),
			want:    "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			util.DefaultExecCommand = tt.command
			defer util.ResetDefaultExecCommand()

			c := &GitCommit{}
			got, err := c.GenerateFullyQualifiedImageName(nil)
			if (err != nil) != tt.wantErr {
				t.Errorf("GitCommit.GenerateFullyQualifiedImageName() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("GitCommit.GenerateFullyQualifiedImageName() = %v, want %v", got, tt.want)
			}
		})
	}
}
