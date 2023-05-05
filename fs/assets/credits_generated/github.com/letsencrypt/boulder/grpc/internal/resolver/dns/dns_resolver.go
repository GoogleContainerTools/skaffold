/*
 *
 * Copyright 2018 gRPC authors.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 */

// Forked from the default internal DNS resolver in the grpc-go package. The
// original source can be found at:
// https://github.com/grpc/grpc-go/blob/v1.49.0/internal/resolver/dns/dns_resolver.go

package dns

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/letsencrypt/boulder/grpc/internal/backoff"
	"github.com/letsencrypt/boulder/grpc/noncebalancer"
	"google.golang.org/grpc/grpclog"
	"google.golang.org/grpc/resolver"
	"google.golang.org/grpc/serviceconfig"
)

var logger = grpclog.Component("srv")

// Globals to stub out in tests. TODO: Perhaps these two can be combined into a
// single variable for testing the resolver?
var (
	newTimer           = time.NewTimer
	newTimerDNSResRate = time.NewTimer
)

func init() {
	resolver.Register(NewDefaultSRVBuilder())
	resolver.Register(NewNonceSRVBuilder())
}

const defaultDNSSvrPort = "53"

var defaultResolver netResolver = net.DefaultResolver

var (
	errMissingAddr = errors.New("dns resolver: missing address")

	// Addresses ending with a colon that is supposed to be the separator
	// between host and port is not allowed.  E.g. "::" is a valid address as
	// it is an IPv6 address (host only) and "[::]:" is invalid as it ends with
	// a colon as the host and port separator
	errEndsWithColon = errors.New("dns resolver: missing port after port-separator colon")
)

var (
	// To prevent excessive re-resolution, we enforce a rate limit on DNS
	// resolution requests.
	minDNSResRate = 30 * time.Second
)

var customAuthorityDialer = func(authority string) func(ctx context.Context, network, address string) (net.Conn, error) {
	return func(ctx context.Context, network, address string) (net.Conn, error) {
		var dialer net.Dialer
		return dialer.DialContext(ctx, network, authority)
	}
}

var customAuthorityResolver = func(authority string) (*net.Resolver, error) {
	host, port, err := parseTarget(authority, defaultDNSSvrPort)
	if err != nil {
		return nil, err
	}
	return &net.Resolver{
		PreferGo: true,
		Dial:     customAuthorityDialer(net.JoinHostPort(host, port)),
	}, nil
}

// NewDefaultSRVBuilder creates a srvBuilder which is used to factory SRV DNS
// resolvers.
func NewDefaultSRVBuilder() resolver.Builder {
	return &srvBuilder{scheme: "srv"}
}

// NewSRVBuilder creates a srvBuilder which is used to factory SRV DNS resolvers
// with a custom grpc.Balancer use by nonce-service clients.
func NewNonceSRVBuilder() resolver.Builder {
	return &srvBuilder{scheme: noncebalancer.SRVResolverScheme, balancer: noncebalancer.Name}
}

type srvBuilder struct {
	scheme   string
	balancer string
}

// Build creates and starts a DNS resolver that watches the name resolution of the target.
func (b *srvBuilder) Build(target resolver.Target, cc resolver.ClientConn, opts resolver.BuildOptions) (resolver.Resolver, error) {
	var names []name
	for _, i := range strings.Split(target.Endpoint, ",") {
		service, domain, err := parseServiceDomain(i)
		if err != nil {
			return nil, err
		}
		names = append(names, name{service: service, domain: domain})
	}

	ctx, cancel := context.WithCancel(context.Background())
	d := &dnsResolver{
		names:  names,
		ctx:    ctx,
		cancel: cancel,
		cc:     cc,
		rn:     make(chan struct{}, 1),
	}

	if target.Authority == "" {
		d.resolver = defaultResolver
	} else {
		var err error
		d.resolver, err = customAuthorityResolver(target.Authority)
		if err != nil {
			return nil, err
		}
	}

	if b.balancer != "" {
		d.serviceConfig = cc.ParseServiceConfig(fmt.Sprintf(`{"loadBalancingConfig": [{"%s":{}}]}`, b.balancer))
	}

	d.wg.Add(1)
	go d.watcher()
	return d, nil
}

// Scheme returns the naming scheme of this resolver builder.
func (b *srvBuilder) Scheme() string {
	return b.scheme
}

type netResolver interface {
	LookupHost(ctx context.Context, host string) (addrs []string, err error)
	LookupSRV(ctx context.Context, service, proto, name string) (cname string, addrs []*net.SRV, err error)
}

type name struct {
	service string
	domain  string
}

// dnsResolver watches for the name resolution update for a non-IP target.
type dnsResolver struct {
	names    []name
	resolver netResolver
	ctx      context.Context
	cancel   context.CancelFunc
	cc       resolver.ClientConn
	// rn channel is used by ResolveNow() to force an immediate resolution of the target.
	rn chan struct{}
	// wg is used to enforce Close() to return after the watcher() goroutine has finished.
	// Otherwise, data race will be possible. [Race Example] in dns_resolver_test we
	// replace the real lookup functions with mocked ones to facilitate testing.
	// If Close() doesn't wait for watcher() goroutine finishes, race detector sometimes
	// will warns lookup (READ the lookup function pointers) inside watcher() goroutine
	// has data race with replaceNetFunc (WRITE the lookup function pointers).
	wg            sync.WaitGroup
	serviceConfig *serviceconfig.ParseResult
}

// ResolveNow invoke an immediate resolution of the target that this dnsResolver watches.
func (d *dnsResolver) ResolveNow(resolver.ResolveNowOptions) {
	select {
	case d.rn <- struct{}{}:
	default:
	}
}

// Close closes the dnsResolver.
func (d *dnsResolver) Close() {
	d.cancel()
	d.wg.Wait()
}

func (d *dnsResolver) watcher() {
	defer d.wg.Done()
	backoffIndex := 1
	for {
		state, err := d.lookup()
		if err != nil {
			// Report error to the underlying grpc.ClientConn.
			d.cc.ReportError(err)
		} else {
			if d.serviceConfig != nil {
				state.ServiceConfig = d.serviceConfig
			}
			err = d.cc.UpdateState(*state)
		}

		var timer *time.Timer
		if err == nil {
			// Success resolving, wait for the next ResolveNow. However, also wait 30 seconds at the very least
			// to prevent constantly re-resolving.
			backoffIndex = 1
			timer = newTimerDNSResRate(minDNSResRate)
			select {
			case <-d.ctx.Done():
				timer.Stop()
				return
			case <-d.rn:
			}
		} else {
			// Poll on an error found in DNS Resolver or an error received from ClientConn.
			timer = newTimer(backoff.DefaultExponential.Backoff(backoffIndex))
			backoffIndex++
		}
		select {
		case <-d.ctx.Done():
			timer.Stop()
			return
		case <-timer.C:
		}
	}
}

func (d *dnsResolver) lookupSRV() ([]resolver.Address, error) {
	var newAddrs []resolver.Address
	for _, n := range d.names {
		_, srvs, err := d.resolver.LookupSRV(d.ctx, n.service, "tcp", n.domain)
		if err != nil {
			err = handleDNSError(err, "SRV") // may become nil
			return nil, err
		}
		for _, s := range srvs {
			backendAddrs, err := d.resolver.LookupHost(d.ctx, s.Target)
			if err != nil {
				err = handleDNSError(err, "A") // may become nil
				if err == nil {
					// If there are other SRV records, look them up and ignore this
					// one that does not exist.
					continue
				}
				return nil, err
			}
			for _, a := range backendAddrs {
				ip, ok := formatIP(a)
				if !ok {
					return nil, fmt.Errorf("srv: error parsing A record IP address %v", a)
				}
				addr := ip + ":" + strconv.Itoa(int(s.Port))
				newAddrs = append(newAddrs, resolver.Address{Addr: addr, ServerName: s.Target})
			}
		}
	}
	return newAddrs, nil
}

func handleDNSError(err error, lookupType string) error {
	if dnsErr, ok := err.(*net.DNSError); ok && !dnsErr.IsTimeout && !dnsErr.IsTemporary {
		// Timeouts and temporary errors should be communicated to gRPC to
		// attempt another DNS query (with backoff).  Other errors should be
		// suppressed (they may represent the absence of a TXT record).
		return nil
	}
	if err != nil {
		err = fmt.Errorf("srv: %v record lookup error: %v", lookupType, err)
		logger.Info(err)
	}
	return err
}

func (d *dnsResolver) lookup() (*resolver.State, error) {
	addrs, err := d.lookupSRV()
	if err != nil {
		return nil, err
	}
	return &resolver.State{Addresses: addrs}, nil
}

// formatIP returns ok = false if addr is not a valid textual representation of an IP address.
// If addr is an IPv4 address, return the addr and ok = true.
// If addr is an IPv6 address, return the addr enclosed in square brackets and ok = true.
func formatIP(addr string) (addrIP string, ok bool) {
	ip := net.ParseIP(addr)
	if ip == nil {
		return "", false
	}
	if ip.To4() != nil {
		return addr, true
	}
	return "[" + addr + "]", true
}

// parseTarget takes the user input target string and default port, returns formatted host and port info.
// If target doesn't specify a port, set the port to be the defaultPort.
// If target is in IPv6 format and host-name is enclosed in square brackets, brackets
// are stripped when setting the host.
// examples:
// target: "www.google.com" defaultPort: "443" returns host: "www.google.com", port: "443"
// target: "ipv4-host:80" defaultPort: "443" returns host: "ipv4-host", port: "80"
// target: "[ipv6-host]" defaultPort: "443" returns host: "ipv6-host", port: "443"
// target: ":80" defaultPort: "443" returns host: "localhost", port: "80"
func parseTarget(target, defaultPort string) (host, port string, err error) {
	if target == "" {
		return "", "", errMissingAddr
	}
	if ip := net.ParseIP(target); ip != nil {
		// target is an IPv4 or IPv6(without brackets) address
		return target, defaultPort, nil
	}
	if host, port, err = net.SplitHostPort(target); err == nil {
		if port == "" {
			// If the port field is empty (target ends with colon), e.g. "[::1]:", this is an error.
			return "", "", errEndsWithColon
		}
		// target has port, i.e ipv4-host:port, [ipv6-host]:port, host-name:port
		if host == "" {
			// Keep consistent with net.Dial(): If the host is empty, as in ":80", the local system is assumed.
			host = "localhost"
		}
		return host, port, nil
	}
	if host, port, err = net.SplitHostPort(target + ":" + defaultPort); err == nil {
		// target doesn't have port
		return host, port, nil
	}
	return "", "", fmt.Errorf("invalid target address %v, error info: %v", target, err)
}

// parseTarget takes the user input target string and parses the service domain
// names for SRV lookup. Input is expected to be a hostname containing at least
// two labels (e.g. "foo.bar", "foo.bar.baz"). The first label is the service
// name and the rest is the domain name. If the target is not in the expected
// format, an error is returned.
func parseServiceDomain(target string) (string, string, error) {
	sd := strings.SplitN(target, ".", 2)
	if len(sd) < 2 || sd[0] == "" || sd[1] == "" {
		return "", "", fmt.Errorf("srv: hostname %q contains < 2 labels", target)
	}
	return sd[0], sd[1], nil
}
