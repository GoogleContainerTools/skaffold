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
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"

	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/status"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/config"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
	"github.com/GoogleContainerTools/skaffold/proto"
)

var srv *server

type server struct {
	buildIntentCallback  func()
	syncIntentCallback   func()
	deployIntentCallback func()
	autoBuildCallback    func(bool)
	autoSyncCallback     func(bool)
	autoDeployCallback   func(bool)
}

func SetBuildCallback(callback func()) {
	if srv != nil {
		srv.buildIntentCallback = callback
	}
}

func SetDeployCallback(callback func()) {
	if srv != nil {
		srv.deployIntentCallback = callback
	}
}

func SetSyncCallback(callback func()) {
	if srv != nil {
		srv.syncIntentCallback = callback
	}
}

func SetAutoBuildCallback(callback func(bool)) {
	if srv != nil {
		srv.autoBuildCallback = callback
	}
}

func SetAutoDeployCallback(callback func(bool)) {
	if srv != nil {
		srv.autoDeployCallback = callback
	}
}

func SetAutoSyncCallback(callback func(bool)) {
	if srv != nil {
		srv.autoSyncCallback = callback
	}
}

// Initialize creates the gRPC and HTTP servers for serving the state and event log.
// It returns a shutdown callback for tearing down the grpc server,
// which the runner is responsible for calling.
func Initialize(opts config.SkaffoldOptions) (func() error, error) {
	if !opts.EnableRPC {
		return func() error { return nil }, nil
	}

	var usedPorts util.PortSet

	originalRPCPort := opts.RPCPort
	if originalRPCPort == -1 {
		return func() error { return nil }, nil
	}
	rpcPort := util.GetAvailablePort(util.Loopback, originalRPCPort, &usedPorts)
	if rpcPort != originalRPCPort {
		logrus.Warnf("port %d for gRPC server already in use: using %d instead", originalRPCPort, rpcPort)
	}
	usedPorts.Set(rpcPort)
	grpcCallback, err := newGRPCServer(rpcPort)
	if err != nil {
		return grpcCallback, fmt.Errorf("starting gRPC server: %w", err)
	}

	originalHTTPPort := opts.RPCHTTPPort
	httpPort := util.GetAvailablePort(util.Loopback, originalHTTPPort, &usedPorts)
	if httpPort != originalHTTPPort {
		logrus.Warnf("port %d for gRPC HTTP server already in use: using %d instead", originalHTTPPort, httpPort)
	}

	httpCallback, err := newHTTPServer(httpPort, rpcPort)
	callback := func() error {
		httpErr := httpCallback()
		grpcErr := grpcCallback()
		errStr := ""
		if grpcErr != nil {
			errStr += fmt.Sprintf("grpc callback error: %s\n", grpcErr.Error())
		}
		if httpErr != nil {
			errStr += fmt.Sprintf("http callback error: %s\n", httpErr.Error())
		}
		return errors.New(errStr)
	}
	if err != nil {
		return callback, fmt.Errorf("starting HTTP server: %w", err)
	}

	return callback, nil
}

func newGRPCServer(port int) (func() error, error) {
	l, err := net.Listen("tcp", fmt.Sprintf("%s:%d", util.Loopback, port))
	if err != nil {
		return func() error { return nil }, fmt.Errorf("creating listener: %w", err)
	}
	logrus.Infof("starting gRPC server on port %d", port)

	s := grpc.NewServer()
	srv = &server{
		buildIntentCallback:  func() {},
		deployIntentCallback: func() {},
		syncIntentCallback:   func() {},
		autoBuildCallback:    func(bool) {},
		autoSyncCallback:     func(bool) {},
		autoDeployCallback:   func(bool) {},
	}
	proto.RegisterSkaffoldServiceServer(s, srv)

	go func() {
		if err := s.Serve(l); err != nil {
			logrus.Errorf("failed to start grpc server: %s", err)
		}
	}()
	return func() error {
		s.Stop()
		return l.Close()
	}, nil
}

func newHTTPServer(port, proxyPort int) (func() error, error) {
	mux := runtime.NewServeMux(runtime.WithProtoErrorHandler(errorHandler))
	opts := []grpc.DialOption{grpc.WithInsecure()}
	err := proto.RegisterSkaffoldServiceHandlerFromEndpoint(context.Background(), mux, fmt.Sprintf("%s:%d", util.Loopback, proxyPort), opts)
	if err != nil {
		return func() error { return nil }, err
	}

	l, err := net.Listen("tcp", fmt.Sprintf("%s:%d", util.Loopback, port))
	if err != nil {
		return func() error { return nil }, fmt.Errorf("creating listener: %w", err)
	}
	logrus.Infof("starting gRPC HTTP server on port %d", port)

	go http.Serve(l, mux)

	return l.Close, nil
}

type errResponse struct {
	Err string `json:"error,omitempty"`
}

func errorHandler(ctx context.Context, _ *runtime.ServeMux, marshaler runtime.Marshaler, writer http.ResponseWriter, _ *http.Request, err error) {
	writer.Header().Set("Content-type", marshaler.ContentType())
	s, _ := status.FromError(err)
	writer.WriteHeader(runtime.HTTPStatusFromCode(s.Code()))
	if err := json.NewEncoder(writer).Encode(errResponse{
		Err: s.Message(),
	}); err != nil {
		writer.Write([]byte(`{"error": "failed to marshal error message"}`))
	}
}
