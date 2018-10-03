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
package apiversion

import (
	"reflect"
	"testing"

	"github.com/blang/semver"
)

func TestMustParse(t *testing.T) {
	_ = MustParse("skaffold/v1alpha4")
}

func TestMustParse_panic(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Errorf("Should have panicked")
		}
	}()
	_ = MustParse("invalid version")
}

func TestParseVersion(t *testing.T) {
	type args struct {
		v string
	}
	tests := []struct {
		name    string
		args    args
		want    semver.Version
		wantErr bool
	}{
		{
			name: "full",
			args: args{
				v: "skaffold/v7alpha3",
			},
			want: semver.Version{
				Major: 7,
				Pre: []semver.PRVersion{
					{
						VersionStr: "alpha",
					},
					{
						VersionNum: 3,
						IsNum:      true,
					},
				},
			},
		},
		{
			name: "ga",
			args: args{
				v: "skaffold/v4",
			},
			want: semver.Version{
				Major: 4,
			},
		},
		{
			name: "incorrect",
			args: args{
				v: "apps/v1",
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Parse(tt.args.v)
			if (err != nil) != tt.wantErr {
				t.Errorf("Parse() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Parse() = %v, want %v", got, tt.want)
			}
		})
	}
}
