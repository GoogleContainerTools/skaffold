/*
Copyright 2020 The Skaffold Authors

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
package kaniko

import (
	"fmt"
	"testing"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestArgs(t *testing.T) {
	tests := []struct {
		description  string
		artifact     *latest.KanikoArtifact
		expectedArgs []string
		wantErr      bool
	}{
		{
			description: "simple build",
			artifact: &latest.KanikoArtifact{
				DockerfilePath: "Dockerfile",
			},
			expectedArgs: []string{},
			wantErr:      false,
		},
		{
			description: "with BuildArgs",
			artifact: &latest.KanikoArtifact{
				DockerfilePath: "Dockerfile",
				BuildArgs: map[string]*string{
					"arg1": util.StringPtr("value1"),
					"arg2": nil,
				},
			},
			expectedArgs: []string{
				BuildArgsFlag, "arg1=value1",
				BuildArgsFlag, "arg2",
			},
			wantErr: false,
		},
		{
			description: "with Cache",
			artifact: &latest.KanikoArtifact{
				DockerfilePath: "Dockerfile",
				Cache:          &latest.KanikoCache{},
			},
			expectedArgs: []string{
				CacheFlag,
			},
			wantErr: false,
		},
		{
			description: "with Cache Options",
			artifact: &latest.KanikoArtifact{
				DockerfilePath: "Dockerfile",
				Cache: &latest.KanikoCache{
					Repo:     "gcr.io/ngnix",
					HostPath: "/cache",
					TTL:      "2",
				},
			},
			expectedArgs: []string{
				CacheFlag,
				CacheRepoFlag, "gcr.io/ngnix",
				CacheDirFlag, "/cache",
				CacheTTLFlag, "2",
			},
			wantErr: false,
		},
		{
			description: "with Cleanup",
			artifact: &latest.KanikoArtifact{
				DockerfilePath: "Dockerfile",
				Cleanup:        true,
			},
			expectedArgs: []string{
				CleanupFlag,
			},
			wantErr: false,
		},
		{
			description: "with DigestFile",
			artifact: &latest.KanikoArtifact{
				DockerfilePath: "Dockerfile",
				DigestFile:     "/tmp/digest",
			},
			expectedArgs: []string{
				DigestFileFlag, "/tmp/digest",
			},
			wantErr: false,
		},
		{
			description: "with Force",
			artifact: &latest.KanikoArtifact{
				DockerfilePath: "Dockerfile",
				Force:          true,
			},
			expectedArgs: []string{
				ForceFlag,
			},
			wantErr: false,
		},
		{
			description: "with ImageNameWithDigestFile",
			artifact: &latest.KanikoArtifact{
				DockerfilePath:          "Dockerfile",
				ImageNameWithDigestFile: "/tmp/imageName",
			},
			expectedArgs: []string{
				ImageNameWithDigestFileFlag, "/tmp/imageName",
			},
			wantErr: false,
		},
		{
			description: "with Insecure",
			artifact: &latest.KanikoArtifact{
				DockerfilePath: "Dockerfile",
				Insecure:       true,
			},
			expectedArgs: []string{
				InsecureFlag,
			},
			wantErr: false,
		},
		{
			description: "with InsecurePull",
			artifact: &latest.KanikoArtifact{
				DockerfilePath: "Dockerfile",
				InsecurePull:   true,
			},
			expectedArgs: []string{
				InsecurePullFlag,
			},
			wantErr: false,
		},
		{
			description: "with InsecureRegistry",
			artifact: &latest.KanikoArtifact{
				DockerfilePath: "Dockerfile",
				InsecureRegistry: []string{
					"s1.registry.url:5000",
					"s2.registry.url:5000",
				},
			},
			expectedArgs: []string{
				InsecureRegistryFlag, "s1.registry.url:5000",
				InsecureRegistryFlag, "s2.registry.url:5000",
			},
			wantErr: false,
		},
		{
			description: "with LogFormat",
			artifact: &latest.KanikoArtifact{
				DockerfilePath: "Dockerfile",
				LogFormat:      "json",
			},
			expectedArgs: []string{
				LogFormatFlag, "json",
			},
			wantErr: false,
		},
		{
			description: "with LogTimestamp",
			artifact: &latest.KanikoArtifact{
				DockerfilePath: "Dockerfile",
				LogTimestamp:   true,
			},
			expectedArgs: []string{
				LogTimestampFlag,
			},
			wantErr: false,
		},
		{
			description: "with NoPush",
			artifact: &latest.KanikoArtifact{
				DockerfilePath: "Dockerfile",
				NoPush:         true,
			},
			expectedArgs: []string{
				NoPushFlag,
			},
			wantErr: false,
		},
		{
			description: "with OCILayoutPath",
			artifact: &latest.KanikoArtifact{
				DockerfilePath: "Dockerfile",
				OCILayoutPath:  "/tmp/builtImage",
			},
			expectedArgs: []string{
				OCILayoutFlag, "/tmp/builtImage",
			},
			wantErr: false,
		},
		{
			description: "with RegistryCertificate",
			artifact: &latest.KanikoArtifact{
				DockerfilePath: "Dockerfile",
				RegistryCertificate: map[string]*string{
					"s1.registry.url": util.StringPtr("/etc/certs/certificate1.cert"),
					"s2.registry.url": util.StringPtr("/etc/certs/certificate2.cert"),
				},
			},
			expectedArgs: []string{
				RegistryCertificateFlag, "s1.registry.url=/etc/certs/certificate1.cert",
				RegistryCertificateFlag, "s2.registry.url=/etc/certs/certificate2.cert",
			},
			wantErr: false,
		},
		{
			description: "with RegistryMirror",
			artifact: &latest.KanikoArtifact{
				DockerfilePath: "Dockerfile",
				RegistryMirror: "mirror.gcr.io",
			},
			expectedArgs: []string{
				RegistryMirrorFlag, "mirror.gcr.io",
			},
			wantErr: false,
		},
		{
			description: "with Reproducible",
			artifact: &latest.KanikoArtifact{
				DockerfilePath: "Dockerfile",
				Reproducible:   true,
			},
			expectedArgs: []string{
				ReproducibleFlag,
			},
			wantErr: false,
		},
		{
			description: "with SingleSnapshot",
			artifact: &latest.KanikoArtifact{
				DockerfilePath: "Dockerfile",
				SingleSnapshot: true,
			},
			expectedArgs: []string{
				SingleSnapshotFlag,
			},
			wantErr: false,
		},
		{
			description: "with SkipTLS",
			artifact: &latest.KanikoArtifact{
				DockerfilePath: "Dockerfile",
				SkipTLS:        true,
			},
			expectedArgs: []string{
				SkipTLSFlag,
				SkipTLSVerifyRegistryFlag, "gcr.io",
			},
			wantErr: false,
		},
		{
			description: "with SkipTLSVerifyPull",
			artifact: &latest.KanikoArtifact{
				DockerfilePath:    "Dockerfile",
				SkipTLSVerifyPull: true,
			},
			expectedArgs: []string{
				SkipTLSVerifyPullFlag,
			},
			wantErr: false,
		},
		{
			description: "with SkipTLSVerifyRegistry",
			artifact: &latest.KanikoArtifact{
				DockerfilePath: "Dockerfile",
				SkipTLSVerifyRegistry: []string{
					"s1.registry.url:443",
					"s2.registry.url:443",
				},
			},
			expectedArgs: []string{
				SkipTLSVerifyRegistryFlag, "s1.registry.url:443",
				SkipTLSVerifyRegistryFlag, "s2.registry.url:443",
			},
			wantErr: false,
		},
		{
			description: "with SkipUnusedStages",
			artifact: &latest.KanikoArtifact{
				DockerfilePath:   "Dockerfile",
				SkipUnusedStages: true,
			},
			expectedArgs: []string{
				SkipUnusedStagesFlag,
			},
			wantErr: false,
		},
		{
			description: "with Target",
			artifact: &latest.KanikoArtifact{
				DockerfilePath: "Dockerfile",
				Target:         "builder",
			},
			expectedArgs: []string{
				TargetFlag, "builder",
			},
			wantErr: false,
		},
		{
			description: "with SnapshotMode",
			artifact: &latest.KanikoArtifact{
				DockerfilePath: "Dockerfile",
				SnapshotMode:   "redo",
			},
			expectedArgs: []string{
				"--snapshotMode", "redo",
			},
			wantErr: false,
		},
		{
			description: "with TarPath",
			artifact: &latest.KanikoArtifact{
				DockerfilePath: "Dockerfile",
				TarPath:        "/workspace/tars",
			},
			expectedArgs: []string{
				TarPathFlag, "/workspace/tars",
			},
			wantErr: false,
		},
		{
			description: "with UseNewRun",
			artifact: &latest.KanikoArtifact{
				DockerfilePath: "Dockerfile",
				UseNewRun:      true,
			},
			expectedArgs: []string{
				UseNewRunFlag,
			},
			wantErr: false,
		},
		{
			description: "with Verbosity",
			artifact: &latest.KanikoArtifact{
				DockerfilePath: "Dockerfile",
				Verbosity:      "trace",
			},
			expectedArgs: []string{
				VerbosityFlag, "trace",
			},
			wantErr: false,
		},
		{
			description: "with WhitelistVarRun",
			artifact: &latest.KanikoArtifact{
				DockerfilePath:  "Dockerfile",
				WhitelistVarRun: true,
			},
			expectedArgs: []string{
				WhitelistVarRunFlag,
			},
			wantErr: false,
		},
		{
			description: "with WhitelistVarRun",
			artifact: &latest.KanikoArtifact{
				DockerfilePath:  "Dockerfile",
				WhitelistVarRun: true,
			},
			expectedArgs: []string{
				WhitelistVarRunFlag,
			},
			wantErr: false,
		},
		{
			description: "with Labels",
			artifact: &latest.KanikoArtifact{
				DockerfilePath: "Dockerfile",
				Label: map[string]*string{
					"label1": util.StringPtr("value1"),
					"label2": nil,
				},
			},
			expectedArgs: []string{
				LabelFlag, "label1=value1",
				LabelFlag, "label2",
			},
			wantErr: false,
		},
	}

	defaultExpectedArgs := []string{
		"--destination", "gcr.io/nginx",
		"--dockerfile", "Dockerfile",
		"--context", fmt.Sprintf("dir://%s", DefaultEmptyDirMountPath),
	}

	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			got, err := Args(test.artifact, "gcr.io/nginx", fmt.Sprintf("dir://%s", DefaultEmptyDirMountPath))
			if (err != nil) != test.wantErr {
				t.Errorf("Args() error = %v, wantErr %v", err, test.wantErr)
				return
			}
			t.CheckDeepEqual(got, append(defaultExpectedArgs, test.expectedArgs...))
		})
	}
}

func Test_artifactRegistry(t *testing.T) {
	tests := []struct {
		name    string
		i       string
		want    string
		wantErr bool
	}{
		{
			name:    "Regular",
			i:       "gcr.io/nginx",
			want:    "gcr.io",
			wantErr: false,
		},
		{
			name:    "with Project",
			i:       "gcr.io/google_containers/nginx",
			want:    "gcr.io",
			wantErr: false,
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.name, func(t *testutil.T) {
			got, err := artifactRegistry(test.i)
			if (err != nil) != test.wantErr {
				t.Errorf("artifactRegistry() error = %v, wantErr %v", err, test.wantErr)
				return
			}
			t.CheckDeepEqual(got, test.want)
		})
	}
}
