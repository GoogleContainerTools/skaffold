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
	"errors"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/encoding/protojson"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/config"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/event"
	eventV2 "github.com/GoogleContainerTools/skaffold/pkg/skaffold/event/v2"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/output/log"
	v2 "github.com/GoogleContainerTools/skaffold/pkg/skaffold/server/v2"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
	protoV1 "github.com/GoogleContainerTools/skaffold/proto/v1"
	protoV2 "github.com/GoogleContainerTools/skaffold/proto/v2"
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

	// Optionally pause execution until endpoint hit
	if opts.WaitForConnection {
		eventV2.WaitForConnection()
	}

	return callback, nil
}

func newGRPCServer(preferredPort int, usedPorts *util.PortSet) (func() error, int, error) {
	l, port, err := listenOnAvailablePort(preferredPort, usedPorts)
	if err != nil {
		return func() error { return nil }, 0, fmt.Errorf("creating listener: %w", err)
	}

	if port != preferredPort {
		log.Entry(context.TODO()).Warnf("starting gRPC server on port %d. (%d is already in use)", port, preferredPort)
	} else {
		log.Entry(context.TODO()).Infof("starting gRPC server on port %d", port)
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
	v2.Srv = &v2.Server{
		BuildIntentCallback:  func() {},
		DeployIntentCallback: func() {},
		SyncIntentCallback:   func() {},
		AutoBuildCallback:    func(bool) {},
		AutoSyncCallback:     func(bool) {},
		AutoDeployCallback:   func(bool) {},
	}
	protoV1.RegisterSkaffoldServiceServer(s, srv)
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

func newHTTPServer(preferredPort, proxyPort int, usedPorts *util.PortSet) (func() error, error) {
	mux := runtime.NewServeMux(
		runtime.WithMarshalerOption(runtime.MIMEWildcard, &runtime.HTTPBodyMarshaler{
			Marshaler: &runtime.JSONPb{
				MarshalOptions: protojson.MarshalOptions{
					UseProtoNames:   true,
					EmitUnpopulated: true,
				},
				UnmarshalOptions: protojson.UnmarshalOptions{
					DiscardUnknown: true,
				},
			},
		}),
	)
	opts := []grpc.DialOption{grpc.WithInsecure()}
	err := protoV1.RegisterSkaffoldServiceHandlerFromEndpoint(context.Background(), mux, fmt.Sprintf("%s:%d", util.Loopback, proxyPort), opts)
	if err != nil {
		return func() error { return nil }, err
	}
	err = protoV2.RegisterSkaffoldV2ServiceHandlerFromEndpoint(context.Background(), mux, fmt.Sprintf("%s:%d", util.Loopback, proxyPort), opts)
	if err != nil {
		return func() error { return nil }, err
	}

	l, port, err := listenOnAvailablePort(preferredPort, usedPorts)
	if err != nil {
		return func() error { return nil }, fmt.Errorf("creating listener: %w", err)
	}

	if port != preferredPort {
		log.Entry(context.TODO()).Warnf("starting gRPC HTTP server on port %d. (%d is already in use)", port, preferredPort)
	} else {
		log.Entry(context.TODO()).Infof("starting gRPC HTTP server on port %d", port)
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
