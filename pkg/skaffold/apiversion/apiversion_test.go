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
)

func TestParseVersion(t *testing.T) {
	type args struct {
		v string
	}
	tests := []struct {
		name    string
		args    args
		want    *Version
		wantErr bool
	}{
		{
			name: "full",
			args: args{
				v: "skaffold/v7alpha3",
			},
			want: &Version{
				Major:   7,
				Release: alpha,
				Minor:   3,
			},
		},
		{
			name: "ga",
			args: args{
				v: "skaffold/v3",
			},
			want: &Version{
				Major:   3,
				Release: ga,
			},
		},
		{
			name: "beta",
			args: args{
				v: "skaffold/v2beta1",
			},
			want: &Version{
				Major:   2,
				Release: beta,
				Minor:   1,
			},
		},
		{
			name: "bad track",
			args: args{
				v: "skaffold/v7notalpha3",
			},
			wantErr: true,
		},
		{
			name: "no minor",
			args: args{
				v: "skaffold/v7alpha",
			},
			wantErr: true,
		},
		{
			name: "bad track",
			args: args{
				v: "skaffold/v7notalpha3",
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseVersion(tt.args.v)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseVersion() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ParseVersion() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestVersion_Compare(t *testing.T) {
	type fields struct {
		Major   int
		Minor   int
		Release ReleaseTrack
	}
	type args struct {
		ov *Version
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   int
	}{
		{
			name: "equal alpha",
			fields: fields{
				Major:   3,
				Minor:   4,
				Release: 0,
			},
			args: args{
				ov: &Version{
					Major:   3,
					Minor:   4,
					Release: 0,
				},
			},
			want: 0,
		},
		{
			name: "equal ga",
			fields: fields{
				Major:   3,
				Release: 2,
			},
			args: args{
				ov: &Version{
					Major:   3,
					Release: 2,
				},
			},
			want: 0,
		},
		{
			name: "alpha < beta",
			fields: fields{
				Major:   3,
				Minor:   4,
				Release: 1,
			},
			args: args{
				ov: &Version{
					Major:   3,
					Minor:   5,
					Release: 0,
				},
			},
			want: 1,
		},
		{
			name: "beta < ga",
			fields: fields{
				Major:   3,
				Minor:   4,
				Release: 2,
			},
			args: args{
				ov: &Version{
					Major:   3,
					Minor:   7,
					Release: 1,
				},
			},
			want: 1,
		},
		{
			name: "ga > beta",
			fields: fields{
				Major:   3,
				Minor:   5,
				Release: 1,
			},
			args: args{
				ov: &Version{
					Major:   3,
					Minor:   4,
					Release: 2,
				},
			},
			want: -1,
		},
		{
			name: "v2 > v1",
			fields: fields{
				Major:   2,
				Minor:   1,
				Release: 1,
			},
			args: args{
				ov: &Version{
					Major:   1,
					Minor:   4,
					Release: 0,
				},
			},
			want: 1,
		},
		{
			name: "minor versions",
			fields: fields{
				Major:   3,
				Minor:   3,
				Release: 0,
			},
			args: args{
				ov: &Version{
					Major:   3,
					Minor:   4,
					Release: 0,
				},
			},
			want: -1,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := &Version{
				Major:   tt.fields.Major,
				Minor:   tt.fields.Minor,
				Release: tt.fields.Release,
			}
			if got := v.Compare(tt.args.ov); got != tt.want {
				t.Errorf("Version.Compare() = %v, want %v", got, tt.want)
			}
		})
	}
}
