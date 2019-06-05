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
	"math/rand"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/GoogleContainerTools/skaffold/integration/skaffold"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/event"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/server/proto"
	"github.com/GoogleContainerTools/skaffold/testutil"
	"github.com/golang/protobuf/jsonpb"
	"github.com/golang/protobuf/ptypes/empty"
	"google.golang.org/grpc"
)

var (
	connectionRetries = 2
	readRetries       = 5
	numLogEntries     = 5
	waitTime          = 1 * time.Second
)

func TestEventLogRPC(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	if ShouldRunGCPOnlyTests() {
		t.Skip("skipping test that is not gcp only")
	}

	rpcAddr := randomPort()
	teardown := setupSkaffoldWithArgs(t, "--rpc-port", rpcAddr)
	defer teardown()

	// start a grpc client and make sure we can connect properly
	var (
		conn   *grpc.ClientConn
		err    error
		client proto.SkaffoldServiceClient
	)

	// connect to the skaffold grpc server
	for i := 0; i < connectionRetries; i++ {
		conn, err = grpc.Dial(fmt.Sprintf(":%s", rpcAddr), grpc.WithInsecure())
		if err != nil {
			t.Logf("unable to establish skaffold grpc connection: retrying...")
			time.Sleep(waitTime)
			continue
		}
		defer conn.Close()

		client = proto.NewSkaffoldServiceClient(conn)
		break
	}

	if client == nil {
		t.Fatalf("error establishing skaffold grpc connection")
	}

	ctx, ctxCancel := context.WithCancel(context.Background())
	defer ctxCancel()

	// read the event log stream from the skaffold grpc server
	var stream proto.SkaffoldService_EventLogClient
	for i := 0; i < readRetries; i++ {
		stream, err = client.EventLog(ctx)
		if err != nil {
			t.Logf("waiting for connection...")
			time.Sleep(waitTime)
			continue
		}
	}
	if stream == nil {
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

func TestEventLogHTTP(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	httpAddr := randomPort()
	teardown := setupSkaffoldWithArgs(t, "--rpc-http-port", httpAddr)
	defer teardown()
	time.Sleep(500 * time.Millisecond) // give skaffold time to process all events

	httpResponse, err := http.Get(fmt.Sprintf("http://localhost:%s/v1/events", httpAddr))
	if err != nil {
		t.Fatalf("error connecting to gRPC REST API: %s", err.Error())
	}
	numEntries := 0
	var logEntries []proto.LogEntry
	for {
		e := make([]byte, 1024)
		l, err := httpResponse.Body.Read(e)
		if err != nil {
			t.Errorf("error reading body from http response: %s", err.Error())
		}
		e = e[0:l] // remove empty bytes from slice

		// sometimes reads can encompass multiple log entries, since Read() doesn't count newlines as EOF.
		readEntries := strings.Split(string(e), "\n")
		for _, entryStr := range readEntries {
			if entryStr == "" {
				continue
			}
			var entry proto.LogEntry
			// the HTTP wrapper sticks the proto messages into a map of "result" -> message.
			// attempting to JSON unmarshal drops necessary proto information, so we just manually
			// strip the string off the response and unmarshal directly to the proto message
			entryStr = strings.Replace(entryStr, "{\"result\":", "", 1)
			entryStr = entryStr[:len(entryStr)-1]
			if err := jsonpb.UnmarshalString(entryStr, &entry); err != nil {
				t.Errorf("error converting http response to proto: %s", err.Error())
			}
			numEntries++
			logEntries = append(logEntries, entry)
		}
		if numEntries >= numLogEntries {
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

func TestGetStateRPC(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	rpcAddr := randomPort()
	// start a skaffold dev loop on an example
	teardown := setupSkaffoldWithArgs(t, "--rpc-port", rpcAddr)
	defer teardown()

	// start a grpc client and make sure we can connect properly
	var (
		conn   *grpc.ClientConn
		err    error
		client proto.SkaffoldServiceClient
	)

	for i := 0; i < connectionRetries; i++ {
		conn, err = grpc.Dial(fmt.Sprintf(":%s", rpcAddr), grpc.WithInsecure())
		if err != nil {
			t.Logf("unable to establish skaffold grpc connection: retrying...")
			time.Sleep(waitTime)
			continue
		}
		defer conn.Close()

		client = proto.NewSkaffoldServiceClient(conn)
		break
	}

	if client == nil {
		t.Fatalf("error establishing skaffold grpc connection")
	}

	ctx, ctxCancel := context.WithCancel(context.Background())
	defer ctxCancel()

	// try a few times and wait around until we see the build is complete, or fail.
	success := false
	var grpcState *proto.State
	for i := 0; i < readRetries; i++ {
		grpcState = retrieveRPCState(ctx, t, client)
		if checkBuildAndDeployComplete(*grpcState) {
			success = true
			break
		}
		time.Sleep(waitTime)
	}
	if !success {
		t.Errorf("skaffold build or deploy not complete. state: %+v\n", grpcState)
	}
}

func TestGetStateHTTP(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	httpAddr := randomPort()
	teardown := setupSkaffoldWithArgs(t, "--rpc-http-port", httpAddr)
	defer teardown()
	time.Sleep(3 * time.Second) // give skaffold time to process all events

	success := false
	var httpState proto.State
	for i := 0; i < readRetries; i++ {
		httpState = retrieveHTTPState(t, httpAddr)
		if checkBuildAndDeployComplete(httpState) {
			success = true
			break
		}
		time.Sleep(waitTime)
	}
	if !success {
		t.Errorf("skaffold build or deploy not complete. state: %+v\n", httpState)
	}
}

func retrieveRPCState(ctx context.Context, t *testing.T, client proto.SkaffoldServiceClient) *proto.State {
	var grpcState *proto.State
	var err error
	attempts := 0
	for {
		grpcState, err = client.GetState(ctx, &empty.Empty{})
		if err == nil {
			break
		}
		if attempts < connectionRetries {
			attempts++
			t.Logf("waiting for connection...")
			time.Sleep(waitTime)
			continue
		}
		t.Fatalf("error retrieving state: %v\n", err)
	}
	return grpcState
}

func retrieveHTTPState(t *testing.T, httpAddr string) proto.State {
	var httpState proto.State

	// retrieve the state via HTTP as well, and verify the result is the same
	httpResponse, err := http.Get(fmt.Sprintf("http://localhost:%s/v1/state", httpAddr))
	if err != nil {
		t.Fatalf("error connecting to gRPC REST API: %s", err.Error())
	}
	b, err := ioutil.ReadAll(httpResponse.Body)
	if err != nil {
		t.Errorf("error reading body from http response: %s", err.Error())
	}
	if err := json.Unmarshal(b, &httpState); err != nil {
		t.Errorf("error converting http response to proto: %s", err.Error())
	}
	return httpState
}

func setupSkaffoldWithArgs(t *testing.T, args ...string) func() {
	Run(t, "testdata/dev", "sh", "-c", "echo foo > foo")

	// Run skaffold build first to fail quickly on a build failure
	skaffold.Build().InDir("testdata/dev").RunOrFail(t)

	// start a skaffold dev loop on an example
	ns, _, deleteNs := SetupNamespace(t)

	stop := skaffold.Dev(args...).InDir("testdata/dev").InNs(ns.Name).RunBackground(t)

	return func() {
		stop()
		deleteNs()
		Run(t, "testdata/dev", "rm", "foo")
	}
}

func randomPort() string {
	return fmt.Sprintf("%d", rand.Intn(65535))
}

func checkBuildAndDeployComplete(state proto.State) bool {
	for _, a := range state.BuildState.Artifacts {
		if a != event.Complete {
			return false
		}
	}
	return state.DeployState.Status == event.Complete
}
