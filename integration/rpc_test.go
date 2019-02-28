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

package integration

import (
	"context"
	"testing"
	"time"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/event"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/event/proto"
	"github.com/GoogleContainerTools/skaffold/testutil"
	"github.com/golang/protobuf/ptypes/empty"

	"google.golang.org/grpc"
)

var (
	retries       = 10
	maxErrors     = 5
	numLogEntries = 5
	waitTime      = 1 * time.Second
)

func TestEventLog(t *testing.T) {
	addr := ":12345"
	// start a skaffold dev loop on an example
	ns, deleteNs := SetupNamespace(t)
	defer deleteNs()

	Run(t, "examples/test-dev-job", "touch", "foo")
	defer Run(t, "examples/test-dev-job", "rm", "foo")

	cancel := make(chan bool)
	go RunSkaffoldNoFail(cancel, "dev", "examples/test-dev-job", ns.Name, "", nil, "--rpc-port", addr)
	defer func() { cancel <- true }()

	time.Sleep(5 * time.Second) // give skaffold time to start up

	// start a grpc client and make sure we can connect properly
	var conn *grpc.ClientConn
	var err error
	var client proto.SkaffoldServiceClient
	attempts := 0
	for {
		conn, err = grpc.Dial(addr, grpc.WithInsecure())
		if err != nil {
			t.Logf("unable to establish skaffold grpc connection: retrying...")
			time.Sleep(waitTime)
			attempts = attempts + 1
		} else {
			defer conn.Close()
			client = proto.NewSkaffoldServiceClient(conn)
			break
		}
		if attempts == retries {
			t.Fatalf("error establishing skaffold grpc connection")
		}
	}

	ctx, ctxCancel := context.WithCancel(context.Background())
	defer ctxCancel()
	var stream proto.SkaffoldService_EventLogClient

	for {
		stream, err = client.EventLog(ctx)
		if err == nil {
			break
		} else if retries < retries {
			retries = retries + 1
			t.Logf("waiting for connection...")
			time.Sleep(3 * time.Second)
			continue
		}
		t.Fatalf("error retrieving event log: %v\n", err)
	}

	// read a preset number of entries from the event log
	logEntries := make([]*proto.LogEntry, 0)
	entriesReceived := 0
	for {
		entry, err := stream.Recv()
		if err != nil {
			t.Errorf("error receiving entry from stream: %s", err)
		}
		if entry != nil {
			logEntries = append(logEntries, entry)
			entriesReceived = entriesReceived + 1
		}
		if entriesReceived == numLogEntries {
			break
		}
	}
	metaEntries, buildEntries, deployEntries := 0, 0, 0
	for _, entry := range logEntries {
		switch entry.Event.GetEventType().(type) {
		case *proto.Event_MetaEvent:
			metaEntries = metaEntries + 1
		case *proto.Event_BuildEvent:
			buildEntries = buildEntries + 1
		case *proto.Event_DeployEvent:
			deployEntries = deployEntries + 1
		default:
		}
	}
	// make sure we have exactly 1 meta entry, 2 deploy entries and 2 build entries
	testutil.CheckDeepEqual(t, 1, metaEntries)
	testutil.CheckDeepEqual(t, 2, deployEntries)
	testutil.CheckDeepEqual(t, 2, buildEntries)
}

func TestGetState(t *testing.T) {
	addr := ":12345"
	// start a skaffold dev loop on an example
	ns, deleteNs := SetupNamespace(t)
	defer deleteNs()

	Run(t, "examples/test-dev-job", "touch", "foo")
	defer Run(t, "examples/test-dev-job", "rm", "foo")

	cancel := make(chan bool)
	go RunSkaffoldNoFail(cancel, "dev", "examples/test-dev-job", ns.Name, "", nil, "--rpc-port", addr)
	defer func() { cancel <- true }()

	time.Sleep(5 * time.Second) // give skaffold time to start up

	// start a grpc client and make sure we can connect properly
	var conn *grpc.ClientConn
	var err error
	var client proto.SkaffoldServiceClient
	attempts := 0
	for {
		conn, err = grpc.Dial(addr, grpc.WithInsecure())
		if err != nil {
			t.Logf("unable to establish skaffold grpc connection: retrying...")
			time.Sleep(waitTime)
			attempts = attempts + 1
		} else {
			defer conn.Close()
			client = proto.NewSkaffoldServiceClient(conn)
			break
		}
		if attempts == retries {
			t.Fatalf("error establishing skaffold grpc connection")
		}
	}

	ctx, ctxCancel := context.WithCancel(context.Background())
	defer ctxCancel()

	// retrieve the state and make sure everything looks correct
	r, err := client.GetState(ctx, &empty.Empty{})
	if err != nil {
		t.Fatalf("error retrieving state: %v", err)
	}
	for _, v := range r.BuildState.Artifacts {
		testutil.CheckDeepEqual(t, event.Complete, v)
	}
	testutil.CheckDeepEqual(t, event.Complete, r.DeployState.Status)
}
