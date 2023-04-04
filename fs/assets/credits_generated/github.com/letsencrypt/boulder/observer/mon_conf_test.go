package observer

import (
	"testing"
	"time"

	"github.com/letsencrypt/boulder/cmd"
	"github.com/letsencrypt/boulder/test"
)

func TestMonConf_validatePeriod(t *testing.T) {
	type fields struct {
		Period cmd.ConfigDuration
	}
	tests := []struct {
		name    string
		fields  fields
		wantErr bool
	}{
		{"valid", fields{cmd.ConfigDuration{Duration: 1 * time.Microsecond}}, false},
		{"1 nanosecond", fields{cmd.ConfigDuration{Duration: 1 * time.Nanosecond}}, true},
		{"none supplied", fields{cmd.ConfigDuration{}}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &MonConf{
				Period: tt.fields.Period,
			}
			err := c.validatePeriod()
			if tt.wantErr {
				test.AssertError(t, err, "MonConf.validatePeriod() should have errored")
			} else {
				test.AssertNotError(t, err, "MonConf.validatePeriod() shouldn't have errored")
			}
		})
	}
}
