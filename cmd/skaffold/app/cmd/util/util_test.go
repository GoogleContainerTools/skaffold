/*
Copyright 2018 The Skaffold Authors

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

package util

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/v1alpha1"

	yaml "gopkg.in/yaml.v2"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/config"
)

func TestParseConfig(t *testing.T) {
	type args struct {
		apiVersion string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "good api version",
			args: args{
				apiVersion: config.LatestVersion,
			},
			wantErr: false,
		},
		{
			name: "old api version",
			args: args{
				apiVersion: v1alpha1.Version,
			},
			wantErr: true,
		},
		{
			name: "new api version",
			args: args{
				apiVersion: "skaffold/v9",
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg, err := config.NewConfig()
			if err != nil {
				t.Fatalf("error generating config: %s", err)
			}
			cfg.APIVersion = tt.args.apiVersion
			cfgStr, err := yaml.Marshal(cfg)
			if err != nil {
				t.Fatalf("error marshalling config: %s", err)
			}

			p := writeTestConfig(t, cfgStr)
			defer os.Remove(p)

			_, err = ParseConfig(p)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseConfig() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
		})
	}
}

func writeTestConfig(t *testing.T, cfg []byte) string {
	t.Helper()
	f, err := ioutil.TempFile("", "")
	if err != nil {
		t.Fatalf("error getting temp file: %s", err)
	}
	defer f.Close()
	if _, err := f.Write(cfg); err != nil {
		t.Fatalf("error writing config: %s", err)
	}
	return f.Name()
}
