package integration

import (
	"testing"

	"github.com/GoogleContainerTools/skaffold/integration/skaffold"
	"github.com/sirupsen/logrus"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubernetes/portforward"
)

func TestPortForward(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	ns, _, deleteNs := SetupNamespace(t)
	defer deleteNs()
	dir := "examples/microservices"
	skaffold.Run().InDir(dir).InNs(ns.Name).RunOrFailOutput(t)
	defer skaffold.Delete().InDir(dir).InNs(ns.Name).RunOrFailOutput(t)
	logrus.SetLevel(logrus.DebugLevel)
	portforward.WhiteBox_PortForwardCycle(ns.Name, t)
}
