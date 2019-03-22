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

package event

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"sync"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/constants"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/event/proto"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
	empty "github.com/golang/protobuf/ptypes/empty"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"

	gw "github.com/GoogleContainerTools/skaffold/pkg/skaffold/event/proto"
)

type server struct{}

func (s *server) GetState(context.Context, *empty.Empty) (*proto.State, error) {
	state := handler.getState()
	return &state, nil
}

func (s *server) EventLog(stream proto.SkaffoldService_EventLogServer) error {
	return handler.forEachEvent(stream.Send)
}

func (s *server) Handle(ctx context.Context, event *proto.Event) (*empty.Empty, error) {
	if event != nil {
		handler.handle(event)
	}
	return &empty.Empty{}, nil
}

// newStatusServer creates the grpc server for serving the state and event log.
func newStatusServer(originalRPCPort, originalHTTPPort int) (func() error, error) {
	if originalRPCPort == -1 {
		return func() error { return nil }, nil
	}
	rpcPort := util.GetAvailablePort(originalRPCPort, &sync.Map{})
	if rpcPort != originalRPCPort && originalRPCPort != constants.DefaultRPCPort {
		logrus.Warnf("provided port %d already in use for gRPC Events API endpoint: using %d instead", originalRPCPort, rpcPort)
	}

	grpcShutdown, err := newGRPCServer(rpcPort)
	if err != nil {
		return nil, errors.Wrap(err, "starting gRPC server")
	}

	m := &sync.Map{}
	m.Store(rpcPort, true)
	httpPort := util.GetAvailablePort(originalHTTPPort, m)
	if httpPort != originalHTTPPort && originalHTTPPort != constants.DefaultRPCHTTPPort {
		logrus.Warnf("provided port %d already in use for HTTP Events API endpoint: using %d instead", originalHTTPPort, httpPort)
	}

	httpShutdown, err := newHTTPServer(httpPort, rpcPort)
	if err != nil {
		return nil, errors.Wrap(err, "starting HTTP server")
	}

	shutdown := func() error {
		grpcShutdown()
		if err := httpShutdown(); err != nil {
			return errors.Wrap(err, "http shutdown")
		}
		return nil
	}

	return shutdown, nil
}

func newGRPCServer(port int) (func(), error) {
	l, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return nil, errors.Wrap(err, "creating listener")
	}
	logrus.Infof("starting gRPC server on port %d", port)

	s := grpc.NewServer()
	proto.RegisterSkaffoldServiceServer(s, &server{})

	go func() {
		if err := s.Serve(l); err != nil {
			logrus.Errorf("failed to start grpc server: %s", err)
		}
	}()

	return func() {
		s.Stop()
	}, nil
}

func newHTTPServer(port, proxyPort int) (func() error, error) {
	mux := runtime.NewServeMux()
	opts := []grpc.DialOption{grpc.WithInsecure()}
	err := gw.RegisterSkaffoldServiceHandlerFromEndpoint(context.Background(), mux, fmt.Sprintf(":%d", proxyPort), opts)
	if err != nil {
		return func() error { return nil }, err
	}

	l, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return func() error { return nil }, errors.Wrap(err, "creating listener")
	}
	logrus.Infof("starting gRPC HTTP server on port %d", port)
	s := http.Server{Handler: mux}

	go func() {
		if err := s.Serve(l); err != nil && err != http.ErrServerClosed {
			logrus.Errorf("failed to start http server %s", err)
		}
	}()

	return func() error{
		if err := s.Shutdown(context.Background()); err != nil {
			return errors.Wrap(err, "shutting down http server")
		}
		return nil
	}, nil
}
