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

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
)

//For WhiteBox testing only
//This is testing a port forward + stop + restart in a simulated dev cycle
func WhiteBoxPortForwardCycle(namespace string, t *testing.T) {
	em := NewEntryManager(os.Stdout)
	portForwardEvent = func(entry *portForwardEntry) {}
	ctx := context.Background()
	localPort := retrieveAvailablePort(9000, em.forwardedPorts)
	pfe := &portForwardEntry{
		resource: latest.PortForwardResource{
			Type:      "deployment",
			Name:      "leeroy-web",
			Namespace: namespace,
			Port:      8080,
		},
		containerName: "dummy container",
		localPort:     localPort,
	}

	defer em.Stop()
	if err := em.forwardPortForwardEntry(ctx, pfe); err != nil {
		t.Fatalf("failed to forward port: %s", err)
	}
	em.Stop()

	time.Sleep(2 * time.Second)

	logrus.Info("getting next port...")
	nextPort := retrieveAvailablePort(localPort, em.forwardedPorts)

	// theoretically we should be able to bind to the very same port
	// this might get flaky when multiple tests are ran. However
	// we shouldn't collide with our own process because of poor cleanup
	if nextPort != localPort {
		t.Fatalf("the same port should be still open, instead first port: %d != second port: %d", localPort, nextPort)
	}

	if err := em.forwardPortForwardEntry(ctx, pfe); err != nil {
		t.Fatalf("failed to forward port: %s", err)
	}
}
