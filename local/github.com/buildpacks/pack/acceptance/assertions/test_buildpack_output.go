//go:build acceptance

package assertions

import (
	"testing"

	h "github.com/buildpacks/pack/testhelpers"
)

type TestBuildpackOutputAssertionManager struct {
	testObject *testing.T
	assert     h.AssertionManager
	output     string
}

func NewTestBuildpackOutputAssertionManager(t *testing.T, output string) TestBuildpackOutputAssertionManager {
	return TestBuildpackOutputAssertionManager{
		testObject: t,
		assert:     h.NewAssertionManager(t),
		output:     output,
	}
}

func (t TestBuildpackOutputAssertionManager) ReportsReadingFileContents(phase, path, content string) {
	t.testObject.Helper()

	t.assert.ContainsF(t.output, "%s: Reading file '%s': %s", phase, path, content)
}

func (t TestBuildpackOutputAssertionManager) ReportsWritingFileContents(phase, path string) {
	t.testObject.Helper()

	t.assert.ContainsF(t.output, "%s: Writing file '%s': written", phase, path)
}

func (t TestBuildpackOutputAssertionManager) ReportsFailingToWriteFileContents(phase, path string) {
	t.testObject.Helper()

	t.assert.ContainsF(t.output, "%s: Writing file '%s': failed", phase, path)
}

func (t TestBuildpackOutputAssertionManager) ReportsConnectedToInternet() {
	t.testObject.Helper()

	t.assert.Contains(t.output, "RESULT: Connected to the internet")
}

func (t TestBuildpackOutputAssertionManager) ReportsDisconnectedFromInternet() {
	t.testObject.Helper()

	t.assert.Contains(t.output, "RESULT: Disconnected from the internet")
}

func (t TestBuildpackOutputAssertionManager) ReportsBuildStep(message string) {
	t.testObject.Helper()

	t.assert.ContainsF(t.output, "Build: %s", message)
}
