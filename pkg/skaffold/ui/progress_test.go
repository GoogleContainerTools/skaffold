package ui

import (
	"fmt"
	"testing"

	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestNewProgressGroup(t *testing.T) {
	tests := []struct {
		description string
	}{
		{
			description: "create a new progress group",
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			NewProgressGroup()

			t.CheckTrue(current.BarCount() == 0)
		})
	}
}

func TestAddNewSpinner(t *testing.T) {
	type args struct {
		name   string
		prefix string
	}
	tests := []struct {
		description string
		numBars     int
	}{
		{
			description: "Add one spinner",
			numBars:     1,
		},
		{
			description: "Add multiple spinners",
			numBars:     3,
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			NewProgressGroup()
			for i := 0; i < test.numBars; i++ {
				spin := AddNewSpinner("", fmt.Sprintf("bar-%d", i), nil)
				spin.Increment()
			}

			t.CheckTrue(current.BarCount() == test.numBars)
			Wait()
		})
	}
}
