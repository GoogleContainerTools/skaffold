package bdns

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"net"
	"strconv"
	"sync"
	"time"

	"github.com/miekg/dns"
	"github.com/prometheus/client_golang/prometheus"
)

// serverProvider represents a type which can provide a list of addresses for
// the bdns to use as DNS resolvers. Different implementations may provide
// different strategies for providing addresses, and may provide different kinds
// of addresses (e.g. host:port combos vs IP addresses).
type ServerProvider interface {
	Addrs() ([]string, error)
	Stop()
}

// staticProvider stores a list of host:port combos, and provides that whole
// list in randomized order when asked for addresses. This replicates the old
// behavior of the bdns.impl's servers field.
type staticProvider struct {
	servers []string
}

var _ ServerProvider = &staticProvider{}

// validateServerAddress ensures that a given server address is formatted in
// such a way that it can be dialed. The provided server address must include a
// host/IP and port separated by colon. Additionally, if the host is a literal
// IPv6 address, it must be enclosed in square brackets.
// (https://golang.org/src/net/dial.go?s=9833:9881#L281)
func validateServerAddress(address string) error {
	// Ensure the host and port portions of `address` can be split.
	host, port, err := net.SplitHostPort(address)
	if err != nil {
		return err
	}

	// Ensure `address` contains both a `host` and `port` portion.
	if host == "" || port == "" {
		return errors.New("port cannot be missing")
	}

	// Ensure the `port` portion of `address` is a valid port.
	portNum, err := strconv.Atoi(port)
	if err != nil {
		return errors.New("port must be an integer: %s")
	}
	if portNum <= 0 || portNum > 65535 {
		return errors.New("port must be an integer between 0 - 65535")
	}

	// Ensure the `host` portion of `address` is a valid FQDN or IP address.
	IPv6 := net.ParseIP(host).To16()
	IPv4 := net.ParseIP(host).To4()
	FQDN := dns.IsFqdn(dns.Fqdn(host))
	if IPv6 == nil && IPv4 == nil && !FQDN {
		return errors.New("host is not an FQDN or IP address")
	}
	return nil
}

func NewStaticProvider(servers []string) (*staticProvider, error) {
	var serverAddrs []string
	for _, server := range servers {
		err := validateServerAddress(server)
		if err != nil {
			return nil, fmt.Errorf("server address %q invalid: %s", server, err)
		}
		serverAddrs = append(serverAddrs, server)
	}
	return &staticProvider{servers: serverAddrs}, nil
}

func (sp *staticProvider) Addrs() ([]string, error) {
	if len(sp.servers) == 0 {
		return nil, fmt.Errorf("no servers configured")
	}
	r := make([]string, len(sp.servers))
	perm := rand.Perm(len(sp.servers))
	for i, v := range perm {
		r[i] = sp.servers[v]
	}
	return r, nil
}

func (sp *staticProvider) Stop() {}

// dynamicProvider uses DNS to look up the set of IP addresses which correspond
// to its single host. It returns this list in random order when asked for
// addresses, and refreshes it regularly using a goroutine started by its
// constructor.
type dynamicProvider struct {
	// The domain name which should be used for DNS. Will be used as the basis of
	// a SRV query to locate DNS services on this domain, which will in turn be
	// used as the basis for A queries to cache IP addrs for those services.
	name string
	// A map of IP addresses (results of A record lookups for SRV Targets) to
	// ports (Port fields in SRV records) associated with those addresses.
	addrs map[string][]uint16
	// Other internal bookkeeping state.
	cancel        chan interface{}
	mu            sync.RWMutex
	refresh       time.Duration
	updateCounter *prometheus.CounterVec
}

var _ ServerProvider = &dynamicProvider{}

// StartDynamicProvider constructs a new dynamicProvider and starts its
// auto-update goroutine. The auto-update process queries DNS for SRV records
// at refresh intervals and uses the resulting IP/port combos to populate the
// list returned by Addrs. The update process ignores the Priority and Weight
// attributes of the SRV records. The given server name should be a full domain
// name like `example.com`, which will result in SRV queries for `_dns._udp.example.com`.
func StartDynamicProvider(server string, refresh time.Duration) (*dynamicProvider, error) {
	if server == "" {
		return nil, fmt.Errorf("no DNS domain name provided")
	}
	dp := dynamicProvider{
		name:    server,
		addrs:   make(map[string][]uint16),
		cancel:  make(chan interface{}),
		refresh: refresh,
		updateCounter: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "dns_update",
				Help: "Counter of attempts to update a dynamic provider",
			},
			[]string{"success"},
		),
	}

	// Update once immediately, so we can know whether that was successful, then
	// kick off the long-running update goroutine.
	err := dp.update()
	if err != nil {
		return nil, fmt.Errorf("failed to start dynamic provider: %w", err)
	}
	go dp.run()

	return &dp, nil
}

// run loops forever, calling dp.update() every dp.refresh interval. Does not
// halt until the dp.cancel channel is closed, so should be run in a goroutine.
func (dp *dynamicProvider) run() {
	t := time.NewTicker(dp.refresh)
	for {
		select {
		case <-t.C:
			err := dp.update()
			if err != nil {
				dp.updateCounter.With(prometheus.Labels{
					"success": "false",
				}).Inc()
				continue
			}
			dp.updateCounter.With(prometheus.Labels{
				"success": "true",
			}).Inc()
		case <-dp.cancel:
			return
		}
	}
}

// update performs the SRV and A record queries necessary to map the given DNS
// domain name to a set of cacheable IP addresses and ports, and stores the
// results in dp.addrs.
func (dp *dynamicProvider) update() error {
	ctx, cancel := context.WithTimeout(context.Background(), dp.refresh/2)
	defer cancel()

	_, srvs, err := net.DefaultResolver.LookupSRV(ctx, "dns", "udp", dp.name)
	if err != nil {
		return fmt.Errorf("failed to lookup SRV records for %q: %w", dp.name, err)
	}
	if len(srvs) == 0 {
		return fmt.Errorf("no SRV records found for %q", dp.name)
	}

	addrPorts := make(map[string][]uint16)
	for _, srv := range srvs {
		addrs, err := net.DefaultResolver.LookupHost(ctx, srv.Target)
		if err != nil {
			return fmt.Errorf("failed to resolve SRV Target %q: %w", srv.Target, err)
		}
		for _, addr := range addrs {
			joinedHostPort := net.JoinHostPort(addr, fmt.Sprint(srv.Port))
			err := validateServerAddress(joinedHostPort)
			if err != nil {
				return fmt.Errorf("invalid SRV addr %q: %w", joinedHostPort, err)
			}
			addrPorts[addr] = append(addrPorts[addr], srv.Port)
		}
	}

	dp.mu.Lock()
	dp.addrs = addrPorts
	dp.mu.Unlock()
	return nil
}

// Addrs returns a shuffled list of IP/port pairs, with the guarantee that no
// two IP/port pairs will share the same IP.
func (dp *dynamicProvider) Addrs() ([]string, error) {
	var r []string
	dp.mu.RLock()
	for ip, ports := range dp.addrs {
		port := fmt.Sprint(ports[rand.Intn(len(ports))])
		addr := net.JoinHostPort(ip, port)
		r = append(r, addr)
	}
	dp.mu.RUnlock()
	rand.Shuffle(len(r), func(i, j int) {
		r[i], r[j] = r[j], r[i]
	})
	return r, nil
}

// Stop tells the background update goroutine to cease. It does not wait for
// confirmation that it has done so.
func (dp *dynamicProvider) Stop() {
	close(dp.cancel)
}
