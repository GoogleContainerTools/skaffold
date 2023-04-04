package ca

import (
	"testing"

	"github.com/letsencrypt/boulder/log"
	"github.com/letsencrypt/boulder/reloader"
	"github.com/prometheus/client_golang/prometheus"
)

func TestNewECDSAAllowListFromFile(t *testing.T) {
	type args struct {
		filename string
		reloader *reloader.Reloader
		logger   log.Logger
		metric   *prometheus.GaugeVec
	}
	tests := []struct {
		name              string
		args              args
		want1337Permitted bool
		wantEntries       int
		wantErrBool       bool
	}{
		{
			name:              "one entry",
			args:              args{"testdata/ecdsa_allow_list.yml", nil, nil, nil},
			want1337Permitted: true,
			wantEntries:       1,
			wantErrBool:       false,
		},
		{
			name:              "one entry but it's not 1337",
			args:              args{"testdata/ecdsa_allow_list2.yml", nil, nil, nil},
			want1337Permitted: false,
			wantEntries:       1,
			wantErrBool:       false,
		},
		{
			name:              "should error due to no file",
			args:              args{"testdata/ecdsa_allow_list_no_exist.yml", nil, nil, nil},
			want1337Permitted: false,
			wantEntries:       0,
			wantErrBool:       true,
		},
		{
			name:              "should error due to malformed YAML",
			args:              args{"testdata/ecdsa_allow_list_malformed.yml", nil, nil, nil},
			want1337Permitted: false,
			wantEntries:       0,
			wantErrBool:       true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1, err := NewECDSAAllowListFromFile(tt.args.filename, tt.args.logger, tt.args.metric)

			if (err != nil) != tt.wantErrBool {
				t.Errorf("NewECDSAAllowListFromFile() error = %v, wantErr %v", err, tt.wantErrBool)
				t.Error(got, got1, err)
				return
			}
			if got != nil && got.permitted(1337) != tt.want1337Permitted {
				t.Errorf("NewECDSAAllowListFromFile() got = %v, want %v", got, tt.want1337Permitted)
			}
			if got1 != tt.wantEntries {
				t.Errorf("NewECDSAAllowListFromFile() got1 = %v, want %v", got1, tt.wantEntries)
			}
		})
	}
}
