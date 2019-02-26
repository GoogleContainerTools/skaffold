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

// import (
// 	"context"
// 	"fmt"
// 	"os"
// 	"testing"
// 	"time"

// 	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/event/proto"
// 	"github.com/golang/protobuf/ptypes/empty"

// 	"google.golang.org/grpc"
// )

// var (
// 	retries  = 10
// 	waitTime = 1 * time.Second
// )

// func TestEventLog(t *testing.T) {
// 	addr := ":12345"
// 	// 1) start a skaffold dev loop on an example
// 	ns, deleteNs := SetupNamespace(t)
// 	defer deleteNs()

// 	Run(t, "examples/test-dev-job", "touch", "foo")
// 	defer Run(t, "examples/test-dev-job", "rm", "foo")

// 	cancel := make(chan bool)
// 	go RunSkaffoldNoFail(cancel, "dev", "examples/test-dev-job", ns.Name, "", nil, "--rpc-port", addr)
// 	defer func() { cancel <- true }()

// 	time.Sleep(5 * time.Second) // give skaffold time to start up

// 	// 2) start a grpc client and make sure we can connect properly
// 	var conn *grpc.ClientConn
// 	var err error
// 	var client proto.SkaffoldServiceClient
// 	attempts := 0
// 	for {
// 		conn, err = grpc.Dial(addr, grpc.WithInsecure())
// 		if err != nil {
// 			t.Logf("unable to establish skaffold grpc connection: retrying...")
// 			time.Sleep(waitTime)
// 			attempts = attempts + 1
// 		} else {
// 			defer conn.Close()
// 			client = proto.NewSkaffoldServiceClient(conn)
// 			break
// 		}
// 		if attempts == retries {
// 			t.Fatalf("error establishing skaffold grpc connection")
// 		}
// 	}

// 	ctx, ctxCancel := context.WithCancel(context.Background())
// 	defer ctxCancel()
// 	var stream proto.SkaffoldService_EventLogClient

// 	for {
// 		stream, err = client.EventLog(ctx)
// 		if err == nil {
// 			break
// 		} else if retries < MAX_RETRIES {
// 			retries = retries + 1
// 			fmt.Println("waiting for connection...")
// 			time.Sleep(3 * time.Second)
// 			continue
// 		}
// 		fmt.Printf("error retrieving event log: %v\n", err)
// 		os.Exit(1)
// 	}

// 	errors := 0
// 	for {
// 		entry, err := stream.Recv()
// 		if err != nil {
// 			errors = errors + 1
// 			fmt.Printf("[%d] error receiving message from stream: %v\n", errors, err)
// 			if errors == MAX_ERRORS {
// 				fmt.Printf("%d errors encountered: quitting", MAX_ERRORS)
// 				os.Exit(1)
// 			}
// 			time.Sleep(1 * time.Second)
// 		} else {
// 			fmt.Printf("%+v\n", entry)
// 		}
// 	}
// }

// func TestGetState(t *testing.T) {
// 	addr := ":12345"
// 	// 1) start a skaffold dev loop on an example
// 	ns, deleteNs := SetupNamespace(t)
// 	defer deleteNs()

// 	Run(t, "examples/test-dev-job", "touch", "foo")
// 	defer Run(t, "examples/test-dev-job", "rm", "foo")

// 	cancel := make(chan bool)
// 	go RunSkaffoldNoFail(cancel, "dev", "examples/test-dev-job", ns.Name, "", nil, "--rpc-port", addr)
// 	defer func() { cancel <- true }()

// 	time.Sleep(5 * time.Second) // give skaffold time to start up

// 	// 2) start a grpc client and make sure we can connect properly
// 	var conn *grpc.ClientConn
// 	var err error
// 	var client proto.SkaffoldServiceClient
// 	attempts := 0
// 	for {
// 		conn, err = grpc.Dial(addr, grpc.WithInsecure())
// 		if err != nil {
// 			t.Logf("unable to establish skaffold grpc connection: retrying...")
// 			time.Sleep(waitTime)
// 			attempts = attempts + 1
// 		} else {
// 			defer conn.Close()
// 			client = proto.NewSkaffoldServiceClient(conn)
// 			break
// 		}
// 		if attempts == retries {
// 			t.Fatalf("error establishing skaffold grpc connection")
// 		}
// 	}

// 	ctx, ctxCancel := context.WithCancel(context.Background())
// 	defer ctxCancel()

// 	// 3) change something in the example
// 	Run(t, "examples/test-dev-job", "sh", "-c", "echo bar > foo")

// 	// 4) retrieve the state and make sure we see something changed
// 	r, err := client.GetState(ctx, &empty.Empty{})
// 	if err != nil {
// 		t.Fatalf("error retrieving state: %v", err)
// 	}
// 	t.Logf("retrieved state: %v", r)
// }
