package grpc

import (
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"strings"

	grpc_prometheus "github.com/grpc-ecosystem/go-grpc-prometheus"
	"github.com/honeycombio/beeline-go/wrappers/hnygrpc"
	"github.com/jmhodges/clock"
	bcreds "github.com/letsencrypt/boulder/grpc/creds"
	"github.com/prometheus/client_golang/prometheus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/status"

	"github.com/letsencrypt/boulder/cmd"
)

// CodedError is a alias required to appease go vet
var CodedError = status.Errorf

var errNilTLS = errors.New("boulder/grpc: received nil tls.Config")

// service represents a single gRPC service that can be registered with a gRPC
// server.
type service struct {
	desc *grpc.ServiceDesc
	impl any
}

// serverBuilder implements a builder pattern for constructing new gRPC servers
// and registering gRPC services on those servers.
type serverBuilder struct {
	cfg      *cmd.GRPCServerConfig
	services map[string]service
	err      error
}

// NewServer returns an object which can be used to build gRPC servers. It
// takes the server's configuration to perform initialization.
func NewServer(c *cmd.GRPCServerConfig) *serverBuilder {
	return &serverBuilder{cfg: c, services: make(map[string]service)}
}

// Add registers a new service (consisting of its description and its
// implementation) to the set of services which will be exposed by this server.
// It returns the modified-in-place serverBuilder so that calls can be chained.
// If there is an error adding this service, it will be exposed when .Build() is
// called.
func (sb *serverBuilder) Add(desc *grpc.ServiceDesc, impl any) *serverBuilder {
	if _, found := sb.services[desc.ServiceName]; found {
		// We've already registered a service with this same name, error out.
		sb.err = fmt.Errorf("attempted double-registration of gRPC service %q", desc.ServiceName)
		return sb
	}

	sb.services[desc.ServiceName] = service{desc, impl}
	return sb
}

// Build creates a gRPC server that uses the provided *tls.Config and exposes
// all of the services added to the builder. It also exposes a health check
// service. It returns two functions, start() and stop(), which should be used
// to start and gracefully stop the server.
func (sb *serverBuilder) Build(tlsConfig *tls.Config, statsRegistry prometheus.Registerer, clk clock.Clock) (func() error, func(), error) {
	// Add the health service to all servers.
	healthSrv := health.NewServer()
	sb = sb.Add(&healthpb.Health_ServiceDesc, healthSrv)

	// Check to see if any of the calls to .Add() resulted in an error.
	if sb.err != nil {
		return nil, nil, sb.err
	}

	// Ensure that every configured service also got added.
	var registeredServices []string
	for r := range sb.services {
		registeredServices = append(registeredServices, r)
	}
	for serviceName := range sb.cfg.Services {
		_, ok := sb.services[serviceName]
		if !ok {
			return nil, nil, fmt.Errorf("gRPC service %q in config does not match any service: %s", serviceName, strings.Join(registeredServices, ", "))
		}
	}

	if tlsConfig == nil {
		return nil, nil, errNilTLS
	}

	// Collect all names which should be allowed to connect to the server at all.
	// This is the names which are allowlisted at the server level, plus the union
	// of all names which are allowlisted for any individual service.
	acceptedSANs := make(map[string]struct{})
	for _, name := range sb.cfg.ClientNames {
		acceptedSANs[name] = struct{}{}
	}
	for _, service := range sb.cfg.Services {
		for _, name := range service.ClientNames {
			acceptedSANs[name] = struct{}{}
		}
	}

	creds, err := bcreds.NewServerCredentials(tlsConfig, acceptedSANs)
	if err != nil {
		return nil, nil, err
	}

	// Set up all of our interceptors which handle metrics, traces, error
	// propagation, and more.
	metrics, err := newServerMetrics(statsRegistry)
	if err != nil {
		return nil, nil, err
	}

	var ai serverInterceptor
	if len(sb.cfg.Services) > 0 {
		ai = newServiceAuthChecker(sb.cfg)
	} else {
		ai = &noopServerInterceptor{}
	}

	mi := newServerMetadataInterceptor(metrics, clk)

	unaryInterceptors := []grpc.UnaryServerInterceptor{
		mi.metrics.grpcMetrics.UnaryServerInterceptor(),
		ai.Unary,
		mi.Unary,
		hnygrpc.UnaryServerInterceptor(),
	}

	streamInterceptors := []grpc.StreamServerInterceptor{
		mi.metrics.grpcMetrics.StreamServerInterceptor(),
		ai.Stream,
		mi.Stream,
		// TODO(#6361): Get a tracing interceptor that works for gRPC streams.
	}

	options := []grpc.ServerOption{
		grpc.Creds(creds),
		grpc.ChainUnaryInterceptor(unaryInterceptors...),
		grpc.ChainStreamInterceptor(streamInterceptors...),
	}
	if sb.cfg.MaxConnectionAge.Duration > 0 {
		options = append(options,
			grpc.KeepaliveParams(keepalive.ServerParameters{
				MaxConnectionAge: sb.cfg.MaxConnectionAge.Duration,
			}))
	}

	// Create the server itself and register all of our services on it.
	server := grpc.NewServer(options...)
	for _, service := range sb.services {
		server.RegisterService(service.desc, service.impl)
	}

	// Finally return the functions which will start and stop the server.
	listener, err := net.Listen("tcp", sb.cfg.Address)
	if err != nil {
		return nil, nil, err
	}

	start := func() error {
		return filterShutdownErrors(server.Serve(listener))
	}
	stop := func() {
		healthSrv.Shutdown()
		server.GracefulStop()
	}

	return start, stop, nil
}

// serverMetrics is a struct type used to return a few registered metrics from
// `newServerMetrics`
type serverMetrics struct {
	grpcMetrics *grpc_prometheus.ServerMetrics
	rpcLag      prometheus.Histogram
}

// newServerMetrics registers metrics with a registry. It constructs and
// registers a *grpc_prometheus.ServerMetrics with timing histogram enabled as
// well as a prometheus Histogram for RPC latency. If called more than once on a
// single registry, it will gracefully avoid registering duplicate metrics.
func newServerMetrics(stats prometheus.Registerer) (serverMetrics, error) {
	// Create the grpc prometheus server metrics instance and register it
	grpcMetrics := grpc_prometheus.NewServerMetrics()
	grpcMetrics.EnableHandlingTimeHistogram()
	err := stats.Register(grpcMetrics)
	if err != nil {
		are := prometheus.AlreadyRegisteredError{}
		if errors.As(err, &are) {
			grpcMetrics = are.ExistingCollector.(*grpc_prometheus.ServerMetrics)
		} else {
			return serverMetrics{}, err
		}
	}

	// rpcLag is a prometheus histogram tracking the difference between the time
	// the client sent an RPC and the time the server received it. Create and
	// register it.
	rpcLag := prometheus.NewHistogram(
		prometheus.HistogramOpts{
			Name: "grpc_lag",
			Help: "Delta between client RPC send time and server RPC receipt time",
		})
	err = stats.Register(rpcLag)
	if err != nil {
		are := prometheus.AlreadyRegisteredError{}
		if errors.As(err, &are) {
			rpcLag = are.ExistingCollector.(prometheus.Histogram)
		} else {
			return serverMetrics{}, err
		}
	}

	return serverMetrics{
		grpcMetrics: grpcMetrics,
		rpcLag:      rpcLag,
	}, nil
}

// filterShutdownErrors returns the input error, with the exception of "use of
// closed network connection," on which it returns nil
// Per https://github.com/grpc/grpc-go/issues/1017, a gRPC server's `Serve()`
// will always return an error, even when GracefulStop() is called. We don't
// want to log graceful stops as errors, so we filter out the meaningless
// error we get in that situation.
func filterShutdownErrors(err error) error {
	if err == nil {
		return nil
	}
	if strings.Contains(err.Error(), "use of closed network connection") {
		return nil
	}
	return err
}
