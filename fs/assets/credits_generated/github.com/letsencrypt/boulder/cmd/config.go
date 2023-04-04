package cmd

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"hash/fnv"
	"math"
	"net"
	"os"
	"strings"
	"time"

	"github.com/go-sql-driver/mysql"
	"github.com/honeycombio/beeline-go"
	"github.com/letsencrypt/boulder/core"
	"google.golang.org/grpc/resolver"
)

// PasswordConfig contains a path to a file containing a password.
type PasswordConfig struct {
	PasswordFile string
}

// Pass returns a password, extracted from the PasswordConfig's PasswordFile
func (pc *PasswordConfig) Pass() (string, error) {
	// Make PasswordConfigs optional, for backwards compatibility.
	if pc.PasswordFile == "" {
		return "", nil
	}
	contents, err := os.ReadFile(pc.PasswordFile)
	if err != nil {
		return "", err
	}
	return strings.TrimRight(string(contents), "\n"), nil
}

// ServiceConfig contains config items that are common to all our services, to
// be embedded in other config structs.
type ServiceConfig struct {
	// DebugAddr is the address to run the /debug handlers on.
	DebugAddr string
	GRPC      *GRPCServerConfig
	TLS       TLSConfig
}

// DBConfig defines how to connect to a database. The connect string is
// stored in a file separate from the config, because it can contain a password,
// which we want to keep out of configs.
type DBConfig struct {
	// A file containing a connect URL for the DB.
	DBConnectFile string

	// MaxOpenConns sets the maximum number of open connections to the
	// database. If MaxIdleConns is greater than 0 and MaxOpenConns is
	// less than MaxIdleConns, then MaxIdleConns will be reduced to
	// match the new MaxOpenConns limit. If n < 0, then there is no
	// limit on the number of open connections.
	MaxOpenConns int

	// MaxIdleConns sets the maximum number of connections in the idle
	// connection pool. If MaxOpenConns is greater than 0 but less than
	// MaxIdleConns, then MaxIdleConns will be reduced to match the
	// MaxOpenConns limit. If n < 0, no idle connections are retained.
	MaxIdleConns int

	// ConnMaxLifetime sets the maximum amount of time a connection may
	// be reused. Expired connections may be closed lazily before reuse.
	// If d < 0, connections are not closed due to a connection's age.
	ConnMaxLifetime ConfigDuration

	// ConnMaxIdleTime sets the maximum amount of time a connection may
	// be idle. Expired connections may be closed lazily before reuse.
	// If d < 0, connections are not closed due to a connection's idle
	// time.
	ConnMaxIdleTime ConfigDuration
}

// URL returns the DBConnect URL represented by this DBConfig object, loading it
// from the file on disk. Leading and trailing whitespace is stripped.
func (d *DBConfig) URL() (string, error) {
	url, err := os.ReadFile(d.DBConnectFile)
	return strings.TrimSpace(string(url)), err
}

// DSNAddressAndUser returns the Address and User of the DBConnect DSN from
// this object.
func (d *DBConfig) DSNAddressAndUser() (string, string, error) {
	dsnStr, err := d.URL()
	if err != nil {
		return "", "", fmt.Errorf("failed to load DBConnect URL: %s", err)
	}
	config, err := mysql.ParseDSN(dsnStr)
	if err != nil {
		return "", "", fmt.Errorf("failed to parse DSN from the DBConnect URL: %s", err)
	}
	return config.Addr, config.User, nil
}

type SMTPConfig struct {
	PasswordConfig
	Server   string
	Port     string
	Username string
}

// PAConfig specifies how a policy authority should connect to its
// database, what policies it should enforce, and what challenges
// it should offer.
type PAConfig struct {
	DBConfig
	Challenges map[core.AcmeChallenge]bool
}

// CheckChallenges checks whether the list of challenges in the PA config
// actually contains valid challenge names
func (pc PAConfig) CheckChallenges() error {
	if len(pc.Challenges) == 0 {
		return errors.New("empty challenges map in the Policy Authority config is not allowed")
	}
	for c := range pc.Challenges {
		if !c.IsValid() {
			return fmt.Errorf("invalid challenge in PA config: %s", c)
		}
	}
	return nil
}

// HostnamePolicyConfig specifies a file from which to load a policy regarding
// what hostnames to issue for.
type HostnamePolicyConfig struct {
	HostnamePolicyFile string
}

// TLSConfig represents certificates and a key for authenticated TLS.
type TLSConfig struct {
	CertFile   *string
	KeyFile    *string
	CACertFile *string
}

// Load reads and parses the certificates and key listed in the TLSConfig, and
// returns a *tls.Config suitable for either client or server use.
func (t *TLSConfig) Load() (*tls.Config, error) {
	if t == nil {
		return nil, fmt.Errorf("nil TLS section in config")
	}
	if t.CertFile == nil {
		return nil, fmt.Errorf("nil CertFile in TLSConfig")
	}
	if t.KeyFile == nil {
		return nil, fmt.Errorf("nil KeyFile in TLSConfig")
	}
	if t.CACertFile == nil {
		return nil, fmt.Errorf("nil CACertFile in TLSConfig")
	}
	caCertBytes, err := os.ReadFile(*t.CACertFile)
	if err != nil {
		return nil, fmt.Errorf("reading CA cert from %q: %s", *t.CACertFile, err)
	}
	rootCAs := x509.NewCertPool()
	if ok := rootCAs.AppendCertsFromPEM(caCertBytes); !ok {
		return nil, fmt.Errorf("parsing CA certs from %s failed", *t.CACertFile)
	}
	cert, err := tls.LoadX509KeyPair(*t.CertFile, *t.KeyFile)
	if err != nil {
		return nil, fmt.Errorf("loading key pair from %q and %q: %s",
			*t.CertFile, *t.KeyFile, err)
	}
	return &tls.Config{
		RootCAs:      rootCAs,
		ClientCAs:    rootCAs,
		ClientAuth:   tls.RequireAndVerifyClientCert,
		Certificates: []tls.Certificate{cert},
		// Set the only acceptable TLS to v1.2 and v1.3.
		MinVersion: tls.VersionTLS12,
		MaxVersion: tls.VersionTLS13,
		// CipherSuites will be ignored for TLS v1.3.
		CipherSuites: []uint16{tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305},
	}, nil
}

// SyslogConfig defines the config for syslogging.
// 3 means "error", 4 means "warning", 6 is "info" and 7 is "debug".
// Configuring a given level causes all messages at that level and below to
// be logged.
type SyslogConfig struct {
	// When absent or zero, this causes no logs to be emitted on stdout/stderr.
	// Errors and warnings will be emitted on stderr if the configured level
	// allows.
	StdoutLevel int
	// When absent or zero, this defaults to logging all messages of level 6
	// or below. To disable syslog logging entirely, set this to -1.
	SyslogLevel int
}

// ConfigDuration is just an alias for time.Duration that allows
// serialization to YAML as well as JSON.
type ConfigDuration struct {
	time.Duration
}

// ErrDurationMustBeString is returned when a non-string value is
// presented to be deserialized as a ConfigDuration
var ErrDurationMustBeString = errors.New("cannot JSON unmarshal something other than a string into a ConfigDuration")

// UnmarshalJSON parses a string into a ConfigDuration using
// time.ParseDuration.  If the input does not unmarshal as a
// string, then UnmarshalJSON returns ErrDurationMustBeString.
func (d *ConfigDuration) UnmarshalJSON(b []byte) error {
	s := ""
	err := json.Unmarshal(b, &s)
	if err != nil {
		var jsonUnmarshalTypeErr *json.UnmarshalTypeError
		if errors.As(err, &jsonUnmarshalTypeErr) {
			return ErrDurationMustBeString
		}
		return err
	}
	dd, err := time.ParseDuration(s)
	d.Duration = dd
	return err
}

// MarshalJSON returns the string form of the duration, as a byte array.
func (d ConfigDuration) MarshalJSON() ([]byte, error) {
	return []byte(d.Duration.String()), nil
}

// UnmarshalYAML uses the same format as JSON, but is called by the YAML
// parser (vs. the JSON parser).
func (d *ConfigDuration) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var s string
	err := unmarshal(&s)
	if err != nil {
		return err
	}
	dur, err := time.ParseDuration(s)
	if err != nil {
		return err
	}

	d.Duration = dur
	return nil
}

// ServiceDomain contains the service and domain name the gRPC client will use
// to construct a SRV DNS query to lookup backends.
type ServiceDomain struct {
	Service string
	Domain  string
}

// GRPCClientConfig contains the information necessary to setup a gRPC client
// connection. The following field combinations are allowed:
//
// ServerIPAddresses, [Timeout]
// ServerAddress, [Timeout], [DNSAuthority], [HostOverride]
// SRVLookup, [Timeout], [DNSAuthority], [HostOverride], [SRVResolver]
// SRVLookups, [Timeout], [DNSAuthority], [HostOverride], [SRVResolver]
type GRPCClientConfig struct {
	// DNSAuthority is a single <hostname|IPv4|[IPv6]>:<port> of the DNS server
	// to be used for resolution of gRPC backends. If the address contains a
	// hostname the gRPC client will resolve it via the system DNS. If the
	// address contains a port, the client will use it directly, otherwise port
	// 53 is used.
	DNSAuthority string

	// SRVLookup contains the service and domain name the gRPC client will use
	// to construct a SRV DNS query to lookup backends. For example: if the
	// resource record is 'foo.service.consul', then the 'Service' is 'foo' and
	// the 'Domain' is 'service.consul'. The expected dNSName to be
	// authenticated in the server certificate would be 'foo.service.consul'.
	//
	// Note: The 'proto' field of the SRV record MUST be 'tcp' and the 'port'
	// field MUST be contain valid port. In a Consul configuration file you
	// would specify 'foo.service.consul' as:
	//
	// services {
	//   id      = "some-unique-id-1"
	//   name    = "foo"
	//   address = "10.77.77.77"
	//   port    = 8080
	//   tags    = ["tcp"]
	// }
	// services {
	//   id      = "some-unique-id-2"
	//   name    = "foo"
	//   address = "10.88.88.88"
	//   port    = 8080
	//   tags    = ["tcp"]
	// }
	//
	// If you've added the above to your Consul configuration file (and reloaded
	// Consul) then you should be able to resolve the following dig query:
	//
	// $ dig @10.55.55.10 -t SRV _foo._tcp.service.consul +short
	// 1 1 8080 0a585858.addr.dc1.consul.
	// 1 1 8080 0a4d4d4d.addr.dc1.consul.
	SRVLookup *ServiceDomain

	// SRVLookups allows you to pass multiple SRV records to the gRPC client.
	// The gRPC client will resolves each SRV record and use the results to
	// construct a list of backends to connect to. For more details, see the
	// documentation for the SRVLookup field. Note: while you can pass multiple
	// targets to the gRPC client using this field, all of the targets will use
	// the same HostOverride and TLS configuration.
	SRVLookups []*ServiceDomain

	// SRVResolver is an optional override to indicate that a specific
	// implementation of the SRV resolver should be used. The default is 'srv'
	// For more details, see the documentation in:
	// grpc/internal/resolver/dns/dns_resolver.go.
	SRVResolver string

	// ServerAddress is a single <hostname|IPv4|[IPv6]>:<port> or `:<port>` that
	// the gRPC client will, if necessary, resolve via DNS and then connect to.
	// If the address provided is 'foo.service.consul:8080' then the dNSName to
	// be authenticated in the server certificate would be 'foo.service.consul'.
	//
	// In a Consul configuration file you would specify 'foo.service.consul' as:
	//
	// services {
	//   id      = "some-unique-id-1"
	//   name    = "foo"
	//   address = "10.77.77.77"
	// }
	// services {
	//   id      = "some-unique-id-2"
	//   name    = "foo"
	//   address = "10.88.88.88"
	// }
	//
	// If you've added the above to your Consul configuration file (and reloaded
	// Consul) then you should be able to resolve the following dig query:
	//
	// $ dig A @10.55.55.10 foo.service.consul +short
	// 10.77.77.77
	// 10.88.88.88
	ServerAddress string

	// ServerIPAddresses is a comma separated list of IP addresses, in the
	// format `<IPv4|[IPv6]>:<port>` or `:<port>`, that the gRPC client will
	// connect to. If the addresses provided are ["10.77.77.77", "10.88.88.88"]
	// then the iPAddress' to be authenticated in the server certificate would
	// be '10.77.77.77' and '10.88.88.88'.
	ServerIPAddresses []string

	// HostOverride is an optional override for the dNSName the client will
	// verify in the certificate presented by the server.
	HostOverride string
	Timeout      ConfigDuration
}

// MakeTargetAndHostOverride constructs the target URI that the gRPC client will
// connect to and the hostname (only for 'ServerAddress' and 'SRVLookup') that
// will be validated during the mTLS handshake. An error is returned if the
// provided configuration is invalid.
func (c *GRPCClientConfig) MakeTargetAndHostOverride() (string, string, error) {
	var hostOverride string
	if c.ServerAddress != "" {
		if c.ServerIPAddresses != nil || c.SRVLookup != nil {
			return "", "", errors.New(
				"both 'serverAddress' and 'serverIPAddresses' or 'SRVLookup' in gRPC client config. Only one should be provided",
			)
		}
		// Lookup backends using DNS A records.
		targetHost, _, err := net.SplitHostPort(c.ServerAddress)
		if err != nil {
			return "", "", err
		}

		hostOverride = targetHost
		if c.HostOverride != "" {
			hostOverride = c.HostOverride
		}
		return fmt.Sprintf("dns://%s/%s", c.DNSAuthority, c.ServerAddress), hostOverride, nil

	} else if c.SRVLookup != nil {
		scheme, err := c.makeSRVScheme()
		if err != nil {
			return "", "", err
		}
		if c.ServerIPAddresses != nil {
			return "", "", errors.New(
				"both 'SRVLookup' and 'serverIPAddresses' in gRPC client config. Only one should be provided",
			)
		}
		// Lookup backends using DNS SRV records.
		targetHost := c.SRVLookup.Service + "." + c.SRVLookup.Domain

		hostOverride = targetHost
		if c.HostOverride != "" {
			hostOverride = c.HostOverride
		}
		return fmt.Sprintf("%s://%s/%s", scheme, c.DNSAuthority, targetHost), hostOverride, nil

	} else if c.SRVLookups != nil {
		scheme, err := c.makeSRVScheme()
		if err != nil {
			return "", "", err
		}
		if c.ServerIPAddresses != nil {
			return "", "", errors.New(
				"both 'SRVLookups' and 'serverIPAddresses' in gRPC client config. Only one should be provided",
			)
		}
		// Lookup backends using multiple DNS SRV records.
		var targetHosts []string
		for _, s := range c.SRVLookups {
			targetHosts = append(targetHosts, s.Service+"."+s.Domain)
		}
		if c.HostOverride != "" {
			hostOverride = c.HostOverride
		}
		return fmt.Sprintf("%s://%s/%s", scheme, c.DNSAuthority, strings.Join(targetHosts, ",")), hostOverride, nil

	} else {
		if c.ServerIPAddresses == nil {
			return "", "", errors.New(
				"neither 'serverAddress', 'SRVLookup', 'SRVLookups' nor 'serverIPAddresses' in gRPC client config. One should be provided",
			)
		}
		// Specify backends as a list of IP addresses.
		return "static:///" + strings.Join(c.ServerIPAddresses, ","), "", nil
	}
}

// makeSRVScheme returns the scheme to use for SRV lookups. If the SRVResolver
// field is empty, it returns "srv". Otherwise it checks that the specified
// SRVResolver is registered with the gRPC runtime and returns it.
func (c *GRPCClientConfig) makeSRVScheme() (string, error) {
	if c.SRVResolver == "" {
		return "srv", nil
	}
	rb := resolver.Get(c.SRVResolver)
	if rb == nil {
		return "", fmt.Errorf("resolver %q is not registered", c.SRVResolver)
	}
	return c.SRVResolver, nil
}

// GRPCServerConfig contains the information needed to start a gRPC server.
type GRPCServerConfig struct {
	Address string `json:"address"`
	// ClientNames is a list of allowed client certificate subject alternate names
	// (SANs). The server will reject clients that do not present a certificate
	// with a SAN present on the `ClientNames` list.
	// DEPRECATED: Use the ClientNames field within each Service instead.
	ClientNames []string `json:"clientNames"`
	// Services is a map of service names to configuration specific to that service.
	// These service names must match the service names advertised by gRPC itself,
	// which are identical to the names set in our gRPC .proto files prefixed by
	// the package names set in those files (e.g. "ca.CertificateAuthority").
	Services map[string]GRPCServiceConfig `json:"services"`
	// MaxConnectionAge specifies how long a connection may live before the server sends a GoAway to the
	// client. Because gRPC connections re-resolve DNS after a connection close,
	// this controls how long it takes before a client learns about changes to its
	// backends.
	// https://pkg.go.dev/google.golang.org/grpc/keepalive#ServerParameters
	MaxConnectionAge ConfigDuration
}

// GRPCServiceConfig contains the information needed to configure a gRPC service.
type GRPCServiceConfig struct {
	// PerServiceClientNames is a map of gRPC service names to client certificate
	// SANs. The upstream listening server will reject connections from clients
	// which do not appear in this list, and the server interceptor will reject
	// RPC calls for this service from clients which are not listed here.
	ClientNames []string `json:"clientNames"`
}

// BeelineConfig provides config options for the Honeycomb beeline-go library,
// which are passed to its beeline.Init() method.
type BeelineConfig struct {
	// WriteKey is the API key needed to send data Honeycomb. This can be given
	// directly in the JSON config for local development, or as a path to a
	// separate file for production deployment.
	WriteKey PasswordConfig
	// Dataset deprecated.
	Dataset string
	// ServiceName is the event collection, e.g. Staging or Prod.
	ServiceName string
	// SampleRate is the (positive integer) denominator of the sample rate.
	// Default: 1 (meaning all traces are sent). Set higher to send fewer traces.
	SampleRate uint32
	// Mute disables honeycomb entirely; useful in test environments.
	Mute bool
	// Many other fields of beeline.Config are omitted as they are not yet used.
}

// makeSampler constructs a SamplerHook which will deterministically decide if
// any given span should be sampled based on its TraceID, which is shared by all
// spans within a trace. If a trace_id can't be found, the span will be sampled.
// A sample rate of 0 defaults to a sample rate of 1 (i.e. all events are sent).
func makeSampler(rate uint32) func(fields map[string]interface{}) (bool, int) {
	if rate == 0 {
		rate = 1
	}
	upperBound := math.MaxUint32 / rate

	return func(fields map[string]interface{}) (bool, int) {
		id, ok := fields["trace.trace_id"].(string)
		if !ok {
			return true, 1
		}
		h := fnv.New32()
		h.Write([]byte(id))
		return h.Sum32() < upperBound, int(rate)
	}
}

// Load converts a BeelineConfig to a beeline.Config, loading the api WriteKey
// and setting the ServiceName automatically.
func (bc *BeelineConfig) Load() (beeline.Config, error) {
	writekey, err := bc.WriteKey.Pass()
	if err != nil {
		return beeline.Config{}, fmt.Errorf("failed to get write key: %w", err)
	}

	return beeline.Config{
		WriteKey:    writekey,
		ServiceName: bc.ServiceName,
		SamplerHook: makeSampler(bc.SampleRate),
		Mute:        bc.Mute,
	}, nil
}
