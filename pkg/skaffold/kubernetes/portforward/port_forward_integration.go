/*
Copyright 2019 The Skaffold Authors

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

package portforward

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/util/wait"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubectl"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
)

// SimulateDevCycle is used for testing a port forward + stop + restart in a simulated dev cycle
func SimulateDevCycle(t *testing.T, kubectlCLI *kubectl.CLI, namespace string) {
	em := NewEntryManager(os.Stdout, NewKubectlForwarder(os.Stdout, kubectlCLI))
	portForwardEventHandler := portForwardEvent
	defer func() { portForwardEvent = portForwardEventHandler }()
	portForwardEvent = func(entry *portForwardEntry) {}
	ctx := context.Background()
	localPort := retrieveAvailablePort("127.0.0.1", 9000, &em.forwardedPorts)
	pfe := newPortForwardEntry(0, latest.PortForwardResource{
		Type:      "deployment",
		Name:      "leeroy-web",
		Namespace: namespace,
		Port:      8080,
	}, "", "dummy container", "", "", localPort, false)
	defer em.Stop()
	em.forwardPortForwardEntry(ctx, pfe)
	em.Stop()

	logrus.Info("waiting for the same port to become available...")
	if err := wait.Poll(100*time.Millisecond, 5*time.Second, func() (done bool, err error) {
		nextPort := retrieveAvailablePort("127.0.0.1", localPort, &em.forwardedPorts)

		logrus.Infof("next port %d", nextPort)

		// theoretically we should be able to bind to the very same port
		// this might get flaky when multiple tests are ran. However
		// we shouldn't collide with our own process because of poor cleanup
		return nextPort == localPort, nil
	}); err != nil {
		t.Fatalf("port is not released after portforwarding stopped: %d", localPort)
	}

	em.forwardPortForwardEntry(ctx, pfe)
}
