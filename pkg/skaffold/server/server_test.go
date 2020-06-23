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

package server

import (
	"fmt"
	"net"
	"testing"

	"google.golang.org/grpc"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/config"
	"github.com/GoogleContainerTools/skaffold/proto"
	"github.com/GoogleContainerTools/skaffold/testutil"
)

var (
	rpcAddr  = 12345
	httpAddr = 23456
)

func TestServerStartup(t *testing.T) {
	// start up servers
	shutdown, err := Initialize(config.SkaffoldOptions{
		EnableRPC:   true,
		RPCPort:     rpcAddr,
		RPCHTTPPort: httpAddr,
	})
	defer shutdown()
	testutil.CheckError(t, false, err)

	// create gRPC client and ensure we can connect
	conn, err := grpc.Dial(fmt.Sprintf(":%d", rpcAddr), grpc.WithInsecure())
	if err != nil {
		t.Errorf("unable to establish skaffold grpc connection")
	}
	defer conn.Close()

	client := proto.NewSkaffoldServiceClient(conn)
	if client == nil {
		t.Errorf("unable to connect to gRPC server")
	}

	// dial httpAddr and make sure port is being served on
	httpConn, err := net.Dial("tcp", fmt.Sprintf(":%d", httpAddr))
	if err != nil {
		t.Errorf("unable to connect to gRPC HTTP server")
	}
	if httpConn == nil {
		t.Errorf("unable to connect to gRPC HTTP server")
	} else {
		httpConn.Close()
	}
}
