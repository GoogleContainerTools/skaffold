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
	"time"

	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/status"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/config"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
	"github.com/GoogleContainerTools/skaffold/proto"
)

const maxTryListen = 10

var (
	srv *server

	// waits for 1 second before forcing a server shutdown
	forceShutdownTimeout = 1 * time.Second
)

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
	if !opts.EnableRPC || opts.RPCPort == -1 {
		return func() error { return nil }, nil
	}

	var usedPorts util.PortSet

	grpcCallback, rpcPort, err := newGRPCServer(opts.RPCPort, &usedPorts)
	if err != nil {
		return grpcCallback, fmt.Errorf("starting gRPC server: %w", err)
	}

	httpCallback, err := newHTTPServer(opts.RPCHTTPPort, rpcPort, &usedPorts)
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

func newGRPCServer(preferredPort int, usedPorts *util.PortSet) (func() error, int, error) {
	l, port, err := listenOnAvailablePort(preferredPort, usedPorts)
	if err != nil {
		return func() error { return nil }, 0, fmt.Errorf("creating listener: %w", err)
	}

	if port != preferredPort {
		logrus.Warnf("starting gRPC server on port %d. (%d is already in use)", port, preferredPort)
	} else {
		logrus.Infof("starting gRPC server on port %d", port)
	}

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
		ctx, cancel := context.WithTimeout(context.Background(), forceShutdownTimeout)
		defer cancel()
		ch := make(chan bool, 1)
		go func() {
			s.GracefulStop()
			ch <- true
		}()
		for {
			select {
			case <-ctx.Done():
				return l.Close()
			case <-ch:
				return l.Close()
			}
		}
	}, port, nil
}

func newHTTPServer(preferredPort, proxyPort int, usedPorts *util.PortSet) (func() error, error) {
	mux := runtime.NewServeMux(runtime.WithProtoErrorHandler(errorHandler))
	opts := []grpc.DialOption{grpc.WithInsecure()}
	err := proto.RegisterSkaffoldServiceHandlerFromEndpoint(context.Background(), mux, fmt.Sprintf("%s:%d", util.Loopback, proxyPort), opts)
	if err != nil {
		return func() error { return nil }, err
	}

	l, port, err := listenOnAvailablePort(preferredPort, usedPorts)
	if err != nil {
		return func() error { return nil }, fmt.Errorf("creating listener: %w", err)
	}

	if port != preferredPort {
		logrus.Warnf("starting gRPC HTTP server on port %d. (%d is already in use)", port, preferredPort)
	} else {
		logrus.Infof("starting gRPC HTTP server on port %d", port)
	}

	server := &http.Server{
		Handler: mux,
	}

	go server.Serve(l)

	return func() error {
		ctx, cancel := context.WithTimeout(context.Background(), forceShutdownTimeout)
		defer cancel()
		return server.Shutdown(ctx)
	}, nil
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

func listenOnAvailablePort(preferredPort int, usedPorts *util.PortSet) (net.Listener, int, error) {
	for try := 1; ; try++ {
		port := util.GetAvailablePort(util.Loopback, preferredPort, usedPorts)

		l, err := net.Listen("tcp", fmt.Sprintf("%s:%d", util.Loopback, port))
		if err != nil {
			if try >= maxTryListen {
				return nil, 0, err
			}

			time.Sleep(1 * time.Second)
			continue
		}

		return l, port, nil
	}
}
