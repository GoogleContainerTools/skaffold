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
	"strconv"
	"time"

	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"google.golang.org/grpc"
	"google.golang.org/grpc/status"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/config"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/event"
	eventV2 "github.com/GoogleContainerTools/skaffold/pkg/skaffold/event/v2"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/output/log"
	v2 "github.com/GoogleContainerTools/skaffold/pkg/skaffold/server/v2"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
	"github.com/GoogleContainerTools/skaffold/proto/v1"
	protoV2 "github.com/GoogleContainerTools/skaffold/proto/v2"
)

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
	emptyCallback := func() error { return nil }
	if !opts.EnableRPC && opts.RPCPort.Value() == nil && opts.RPCHTTPPort.Value() == nil {
		log.Entry(context.TODO()).Debug("skaffold API not starting as it's not requested")
		return emptyCallback, nil
	}

	preferredGRPCPort := 0 // bind to an available port atomically
	if opts.RPCPort.Value() != nil {
		preferredGRPCPort = *opts.RPCPort.Value()
	}
	grpcCallback, grpcPort, err := newGRPCServer(preferredGRPCPort)
	if err != nil {
		return grpcCallback, fmt.Errorf("starting gRPC server: %w", err)
	}

	httpCallback := emptyCallback
	if opts.RPCHTTPPort.Value() != nil {
		httpCallback, err = newHTTPServer(*opts.RPCHTTPPort.Value(), grpcPort)
	}
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
		if opts.EventLogFile != "" {
			logFileErr := event.SaveEventsToFile(opts.EventLogFile)
			if logFileErr != nil {
				errStr += fmt.Sprintf("event log file error: %s\n", logFileErr.Error())
			}
		}
		return errors.New(errStr)
	}
	if err != nil {
		return callback, fmt.Errorf("starting HTTP server: %w", err)
	}

	if opts.EnableRPC && opts.RPCPort.Value() == nil && opts.RPCHTTPPort.Value() == nil {
		log.Entry(context.TODO()).Warnf("started skaffold gRPC API on random port %d", grpcPort)
	}

	// Optionally pause execution until endpoint hit
	if opts.WaitForConnection {
		eventV2.WaitForConnection()
	}

	return callback, nil
}

func newGRPCServer(preferredPort int) (func() error, int, error) {
	l, port, err := listenPort(preferredPort)
	if err != nil {
		return func() error { return nil }, 0, fmt.Errorf("creating listener: %w", err)
	}

	log.Entry(context.TODO()).Infof("starting gRPC server on port %d", port)

	s := grpc.NewServer()
	srv = &server{
		buildIntentCallback:  func() {},
		deployIntentCallback: func() {},
		syncIntentCallback:   func() {},
		autoBuildCallback:    func(bool) {},
		autoSyncCallback:     func(bool) {},
		autoDeployCallback:   func(bool) {},
	}
	v2.Srv = &v2.Server{
		BuildIntentCallback:  func() {},
		DeployIntentCallback: func() {},
		SyncIntentCallback:   func() {},
		AutoBuildCallback:    func(bool) {},
		AutoSyncCallback:     func(bool) {},
		AutoDeployCallback:   func(bool) {},
	}
	proto.RegisterSkaffoldServiceServer(s, srv)
	protoV2.RegisterSkaffoldV2ServiceServer(s, v2.Srv)

	go func() {
		if err := s.Serve(l); err != nil {
			log.Entry(context.TODO()).Errorf("failed to start grpc server: %s", err)
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

func newHTTPServer(preferredPort, proxyPort int) (func() error, error) {
	mux := runtime.NewServeMux(runtime.WithProtoErrorHandler(errorHandler), runtime.WithMarshalerOption(runtime.MIMEWildcard, &runtime.JSONPb{OrigName: true, EmitDefaults: true}))
	opts := []grpc.DialOption{grpc.WithInsecure()}
	err := proto.RegisterSkaffoldServiceHandlerFromEndpoint(context.Background(), mux, net.JoinHostPort(util.Loopback, strconv.Itoa(proxyPort)), opts)
	if err != nil {
		return func() error { return nil }, err
	}
	err = protoV2.RegisterSkaffoldV2ServiceHandlerFromEndpoint(context.Background(), mux, net.JoinHostPort(util.Loopback, strconv.Itoa(proxyPort)), opts)
	if err != nil {
		return func() error { return nil }, err
	}

	l, port, err := listenPort(preferredPort)
	if err != nil {
		return func() error { return nil }, fmt.Errorf("creating listener: %w", err)
	}

	log.Entry(context.TODO()).Infof("starting gRPC HTTP server on port %d (proxying to %d)", port, proxyPort)
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

func listenPort(port int) (net.Listener, int, error) {
	l, err := net.Listen("tcp", net.JoinHostPort(util.Loopback, strconv.Itoa(port)))
	if err != nil {
		return nil, 0, err
	}
	return l, l.Addr().(*net.TCPAddr).Port, nil
}
