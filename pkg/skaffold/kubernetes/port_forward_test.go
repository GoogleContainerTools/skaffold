package kubernetes

import (
	"testing"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
)

func TestPortForwardPod(t *testing.T) {
	var tests = []struct {
		description string
		command     util.Command
		shouldErr   bool
	}{
		{},
	}

	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			if test.command != nil {
				defer func(c util.Command) { util.DefaultExecCommand = c }(util.DefaultExecCommand)
				util.DefaultExecCommand = test.command
			}
		})
	}

	return nil
}
