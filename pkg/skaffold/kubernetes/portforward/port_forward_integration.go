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

	"k8s.io/apimachinery/pkg/util/wait"

	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/kubectl"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/output/log"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/schema/latest"
	schemautil "github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/schema/util"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/util"
)

// SimulateDevCycle is used for testing a port forward + stop + restart in a simulated dev cycle
func SimulateDevCycle(t *testing.T, kubectlCLI *kubectl.CLI, namespace string, level log.Level) {
	log.SetLevel(level)
	em := NewEntryManager(NewKubectlForwarder(kubectlCLI))
	portForwardEventHandler := portForwardEvent
	defer func() { portForwardEvent = portForwardEventHandler }()
	portForwardEvent = func(entry *portForwardEntry) {}
	ctx := context.Background()
	localPort := retrieveAvailablePort(util.Loopback, 9000, &em.forwardedPorts)
	pfe := newPortForwardEntry(0, latest.PortForwardResource{
		Type:      "deployment",
		Name:      "leeroy-web",
		Namespace: namespace,
		Port:      schemautil.FromInt(8080),
	}, "", "dummy container", "", "", localPort, false)
	defer em.Stop()
	em.forwardPortForwardEntry(ctx, os.Stdout, pfe)
	em.Stop()

	log.Entry(ctx).Info("waiting for the same port to become available...")
	if err := wait.Poll(100*time.Millisecond, 5*time.Second, func() (done bool, err error) {
		nextPort := retrieveAvailablePort(util.Loopback, localPort, &em.forwardedPorts)

		log.Entry(ctx).Infof("next port %d", nextPort)

		// theoretically we should be able to bind to the very same port
		// this might get flaky when multiple tests are ran. However
		// we shouldn't collide with our own process because of poor cleanup
		return nextPort == localPort, nil
	}); err != nil {
		t.Fatalf("port is not released after portforwarding stopped: %d", localPort)
	}

	em.forwardPortForwardEntry(ctx, os.Stdout, pfe)
}
