// +build integration

package integration

import (
	"os/exec"
	"testing"

	"github.com/GoogleCloudPlatform/skaffold/pkg/skaffold/kubernetes"
	"github.com/GoogleCloudPlatform/skaffold/pkg/skaffold/util"
)

func TestRunNoArgs(t *testing.T) {
	client, err := kubernetes.GetClientset()
	if err != nil {
		t.Fatalf("Test setup error: getting kubernetes client: %s", err)
	}
	defer func() {
		if err := client.CoreV1().Pods("default").Delete("skaffold", nil); err != nil {
			t.Fatalf("Error deleting pod %s", err)
		}
	}()
	cmd := exec.Command("skaffold", "run")
	cmd.Dir = "../"
	out, outerr, err := util.RunCommand(cmd, nil)
	if err != nil {
		t.Fatalf("skaffold run: \nstdout: %s\nstderr: %s\nerror: %s", out, outerr, err)
	}

	if err := kubernetes.WaitForPodReady(client.CoreV1().Pods("default"), "skaffold"); err != nil {
		t.Fatalf("Timed out waiting for pod ready")
	}
}
