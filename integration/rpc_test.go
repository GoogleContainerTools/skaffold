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
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"testing"
	"time"

	"github.com/GoogleContainerTools/skaffold/integration/skaffold"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/event"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/event/proto"
	"github.com/GoogleContainerTools/skaffold/testutil"
	"github.com/golang/protobuf/ptypes/empty"
	"google.golang.org/grpc"
)

var (
	retries       = 20
	numLogEntries = 5
	waitTime      = 1 * time.Second
)

func TestEventLog(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	Run(t, "testdata/dev", "sh", "-c", "echo foo > foo")
	defer Run(t, "testdata/dev", "rm", "foo")

	// Run skaffold build first to fail quickly on a build failure
	skaffold.Build().InDir("testdata/dev").RunOrFail(t)

	// start a skaffold dev loop on an example
	ns, _, deleteNs := SetupNamespace(t)
	defer deleteNs()

	addr := "12345"
	stop := skaffold.Dev("--rpc-port", addr).InDir("testdata/dev").InNs(ns.Name).RunBackground(t)
	defer stop()

	// start a grpc client and make sure we can connect properly
	var conn *grpc.ClientConn
	var err error
	var client proto.SkaffoldServiceClient
	attempts := 0
	for {
		conn, err = grpc.Dial(fmt.Sprintf(":%s", addr), grpc.WithInsecure())
		if err != nil {
			t.Logf("unable to establish skaffold grpc connection: retrying...")
			attempts++
			if attempts == retries {
				t.Fatalf("error establishing skaffold grpc connection")
			}

			time.Sleep(waitTime)
			continue
		}
		defer conn.Close()

		client = proto.NewSkaffoldServiceClient(conn)
		break
	}

	ctx, ctxCancel := context.WithCancel(context.Background())
	defer ctxCancel()

	var stream proto.SkaffoldService_EventLogClient
	attempts = 0
	for {
		stream, err = client.EventLog(ctx)
		if err == nil {
			break
		}
		if attempts < retries {
			attempts++
			t.Logf("waiting for connection...")
			time.Sleep(3 * time.Second)
			continue
		}
		t.Fatalf("error retrieving event log: %v\n", err)
	}

	// read a preset number of entries from the event log
	var logEntries []*proto.LogEntry
	entriesReceived := 0
	for {
		entry, err := stream.Recv()
		if err != nil {
			t.Errorf("error receiving entry from stream: %s", err)
		}

		if entry != nil {
			logEntries = append(logEntries, entry)
			entriesReceived++
		}
		if entriesReceived == numLogEntries {
			break
		}
	}
	metaEntries, buildEntries, deployEntries := 0, 0, 0
	for _, entry := range logEntries {
		switch entry.Event.GetEventType().(type) {
		case *proto.Event_MetaEvent:
			metaEntries++
		case *proto.Event_BuildEvent:
			buildEntries++
		case *proto.Event_DeployEvent:
			deployEntries++
		default:
		}
	}
	// make sure we have exactly 1 meta entry, 2 deploy entries and 2 build entries
	testutil.CheckDeepEqual(t, 1, metaEntries)
	testutil.CheckDeepEqual(t, 2, deployEntries)
	testutil.CheckDeepEqual(t, 2, buildEntries)
}

func TestGetState(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	Run(t, "testdata/dev", "sh", "-c", "echo foo > foo")
	defer Run(t, "testdata/dev", "rm", "foo")

	// Run skaffold build first to fail quickly on a build failure
	skaffold.Build().InDir("testdata/dev").RunOrFail(t)

	// start a skaffold dev loop on an example
	ns, _, deleteNs := SetupNamespace(t)
	defer deleteNs()

	// start a skaffold dev loop on an example
	rpcAddr := "12345"
	httpAddr := "23456"
	stop := skaffold.Dev("--rpc-port", rpcAddr, "--rpc-http-port", httpAddr).InDir("testdata/dev").InNs(ns.Name).RunBackground(t)
	defer stop()

	// start a grpc client and make sure we can connect properly
	var conn *grpc.ClientConn
	var err error
	var client proto.SkaffoldServiceClient
	attempts := 0
	for {
		conn, err = grpc.Dial(fmt.Sprintf(":%s", rpcAddr), grpc.WithInsecure())
		if err != nil {
			t.Logf("unable to establish skaffold grpc connection: retrying...")
			attempts++
			if attempts == retries {
				t.Fatalf("error establishing skaffold grpc connection")
			}

			time.Sleep(waitTime)
			continue
		}
		defer conn.Close()

		client = proto.NewSkaffoldServiceClient(conn)
		break
	}

	ctx, ctxCancel := context.WithCancel(context.Background())
	defer ctxCancel()

	// retrieve the state and make sure everything looks correct
	var grpcState *proto.State
	attempts = 0
	for {
		grpcState, err = client.GetState(ctx, &empty.Empty{})
		if err == nil {
			break
		}
		if attempts < retries {
			attempts++
			t.Logf("waiting for connection...")
			time.Sleep(3 * time.Second)
			continue
		}
		t.Fatalf("error retrieving state: %v\n", err)
	}

	// retrieve the state via HTTP as well, and verify the result is the same
	httpResponse, err := http.Get(fmt.Sprintf("http://localhost:%s/v1/state", httpAddr))
	if err != nil {
		t.Errorf("error connecting to gRPC REST API: %s", err.Error())
	}
	b, err := ioutil.ReadAll(httpResponse.Body)
	if err != nil {
		t.Errorf("error reading body from http response: %s", err.Error())
	}
	var httpState proto.State
	if err := json.Unmarshal(b, &httpState); err != nil {
		t.Errorf("error converting http response to proto: %s", err.Error())
	}
	testutil.CheckDeepEqual(t, grpcState, &httpState)

	for _, v := range grpcState.BuildState.Artifacts {
		testutil.CheckDeepEqual(t, event.Complete, v)
	}
	testutil.CheckDeepEqual(t, event.Complete, grpcState.DeployState.Status)
}
