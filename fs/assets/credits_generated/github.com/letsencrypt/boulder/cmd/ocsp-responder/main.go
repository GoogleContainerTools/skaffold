package notmain

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/honeycombio/beeline-go"
	"github.com/honeycombio/beeline-go/wrappers/hnynethttp"
	"github.com/prometheus/client_golang/prometheus"

	"github.com/letsencrypt/boulder/cmd"
	"github.com/letsencrypt/boulder/db"
	"github.com/letsencrypt/boulder/features"
	bgrpc "github.com/letsencrypt/boulder/grpc"
	"github.com/letsencrypt/boulder/issuance"
	blog "github.com/letsencrypt/boulder/log"
	"github.com/letsencrypt/boulder/metrics/measured_http"
	"github.com/letsencrypt/boulder/ocsp/responder"
	"github.com/letsencrypt/boulder/ocsp/responder/live"
	redis_responder "github.com/letsencrypt/boulder/ocsp/responder/redis"
	rapb "github.com/letsencrypt/boulder/ra/proto"
	rocsp_config "github.com/letsencrypt/boulder/rocsp/config"
	"github.com/letsencrypt/boulder/sa"
	sapb "github.com/letsencrypt/boulder/sa/proto"
)

type Config struct {
	OCSPResponder struct {
		cmd.ServiceConfig
		DB cmd.DBConfig

		// Source indicates the source of pre-signed OCSP responses to be used. It
		// can be a DBConnect string or a file URL. The file URL style is used
		// when responding from a static file for intermediates and roots.
		// If DBConfig has non-empty fields, it takes precedence over this.
		Source string

		// The list of issuer certificates, against which OCSP requests/responses
		// are checked to ensure we're not responding for anyone else's certs.
		IssuerCerts []string

		Path          string
		ListenAddress string
		// Deprecated and unused.
		MaxAge cmd.ConfigDuration

		// When to timeout a request. This should be slightly lower than the
		// upstream's timeout when making request to ocsp-responder.
		Timeout cmd.ConfigDuration

		// The worst-case freshness of a response during normal operations.
		//
		// This controls behavior when both Redis and MariaDB backends are
		// configured. If a MariaDB response is older than this, ocsp-responder
		// will try to serve a fresher response from Redis, waiting for a Redis
		// response if necessary.
		//
		// This is related to OCSPMinTimeToExpiry in ocsp-updater's config,
		// and both are related to the mandated refresh times in the BRs and
		// root programs (minus a safety margin).
		//
		// This should be configured slightly higher than ocsp-updater's
		// OCSPMinTimeToExpiry, to account for the time taken to sign
		// responses once they pass that threshold. For instance, a good value
		// would be: OCSPMinTimeToExpiry + OldOCSPWindow.
		//
		// This has a default value of 61h.
		ExpectedFreshness cmd.ConfigDuration

		// How often a response should be signed when using Redis/live-signing
		// path. This has a default value of 60h.
		LiveSigningPeriod cmd.ConfigDuration

		// A limit on how many requests to the RA (and onwards to the CA) will
		// be made to sign responses that are not fresh in the cache. This
		// should be set to somewhat less than
		// (HSM signing capacity) / (number of ocsp-responders).
		// Requests that would exceed this limit will block until capacity is
		// available and eventually serve an HTTP 500 Internal Server Error.
		MaxInflightSignings int

		// A limit on how many goroutines can be waiting for a signing slot at
		// a time. When this limit is exceeded, additional signing requests
		// will immediately serve an HTTP 500 Internal Server Error until
		// we are back below the limit. This provides load shedding for when
		// inbound requests arrive faster than our ability to sign them.
		// The default of 0 means "no limit." A good value for this is the
		// longest queue we can expect to process before a timeout. For
		// instance, if the timeout is 5 seconds, and a signing takes 20ms,
		// and we have MaxInflightSignings = 40, we can expect to process
		// 40 * 5 / 0.02 = 10,000 requests before the oldest request times out.
		MaxSigningWaiters int

		ShutdownStopTimeout cmd.ConfigDuration

		RequiredSerialPrefixes []string

		Features map[string]bool

		// Configuration for using Redis as a cache. This configuration should
		// allow for both read and write access.
		Redis rocsp_config.RedisConfig

		// TLS client certificate, private key, and trusted root bundle.
		TLS cmd.TLSConfig

		// RAService configures how to communicate with the RA when it is necessary
		// to generate a fresh OCSP response.
		RAService *cmd.GRPCClientConfig

		// SAService configures how to communicate with the SA to look up
		// certificate status metadata used to confirm/deny that the response from
		// Redis is up-to-date.
		SAService *cmd.GRPCClientConfig

		// LogSampleRate sets how frequently error logs should be emitted. This
		// avoids flooding the logs during outages. 1 out of N log lines will be emitted.
		LogSampleRate int
	}

	Syslog  cmd.SyslogConfig
	Beeline cmd.BeelineConfig
}

func main() {
	configFile := flag.String("config", "", "File path to the configuration file for this service")
	flag.Parse()
	if *configFile == "" {
		fmt.Fprintf(os.Stderr, `Usage of %s:
Config JSON should contain either a DBConnectFile or a Source value containing a file: URL.
If Source is a file: URL, the file should contain a list of OCSP responses in base64-encoded DER,
as generated by Boulder's ceremony command.
`, os.Args[0])
		flag.PrintDefaults()
		os.Exit(1)
	}

	var c Config
	err := cmd.ReadConfigFile(*configFile, &c)
	cmd.FailOnError(err, "Reading JSON config file into config structure")
	err = features.Set(c.OCSPResponder.Features)
	cmd.FailOnError(err, "Failed to set feature flags")

	bc, err := c.Beeline.Load()
	cmd.FailOnError(err, "Failed to load Beeline config")
	beeline.Init(bc)
	defer beeline.Close()

	scope, logger := cmd.StatsAndLogging(c.Syslog, c.OCSPResponder.DebugAddr)
	defer logger.AuditPanic()
	logger.Info(cmd.VersionString())

	clk := cmd.Clock()

	var source responder.Source

	if strings.HasPrefix(c.OCSPResponder.Source, "file:") {
		url, err := url.Parse(c.OCSPResponder.Source)
		cmd.FailOnError(err, "Source was not a URL")
		filename := url.Path
		// Go interprets cwd-relative file urls (file:test/foo.txt) as having the
		// relative part of the path in the 'Opaque' field.
		if filename == "" {
			filename = url.Opaque
		}
		source, err = responder.NewMemorySourceFromFile(filename, logger)
		cmd.FailOnError(err, fmt.Sprintf("Couldn't read file: %s", url.Path))
	} else {
		// Set up the redis source and the combined multiplex source.
		rocspRWClient, err := rocsp_config.MakeClient(&c.OCSPResponder.Redis, clk, scope)
		cmd.FailOnError(err, "Could not make redis client")

		err = rocspRWClient.Ping(context.Background())
		cmd.FailOnError(err, "pinging Redis")

		liveSigningPeriod := c.OCSPResponder.LiveSigningPeriod.Duration
		if liveSigningPeriod == 0 {
			liveSigningPeriod = 60 * time.Hour
		}

		tlsConfig, err := c.OCSPResponder.TLS.Load()
		cmd.FailOnError(err, "TLS config")

		raConn, err := bgrpc.ClientSetup(c.OCSPResponder.RAService, tlsConfig, scope, clk)
		cmd.FailOnError(err, "Failed to load credentials and create gRPC connection to RA")
		rac := rapb.NewRegistrationAuthorityClient(raConn)

		maxInflight := c.OCSPResponder.MaxInflightSignings
		if maxInflight == 0 {
			maxInflight = 1000
		}
		liveSource := live.New(rac, int64(maxInflight), c.OCSPResponder.MaxSigningWaiters)

		rocspSource, err := redis_responder.NewRedisSource(rocspRWClient, liveSource, liveSigningPeriod, clk, scope, logger)
		cmd.FailOnError(err, "Could not create redis source")

		var dbMap *db.WrappedMap
		if c.OCSPResponder.DB != (cmd.DBConfig{}) {
			dbMap, err = sa.InitWrappedDb(c.OCSPResponder.DB, scope, logger)
			cmd.FailOnError(err, "While initializing dbMap")
		}

		var sac sapb.StorageAuthorityReadOnlyClient
		if c.OCSPResponder.SAService != nil {
			saConn, err := bgrpc.ClientSetup(c.OCSPResponder.SAService, tlsConfig, scope, clk)
			cmd.FailOnError(err, "Failed to load credentials and create gRPC connection to SA")
			sac = sapb.NewStorageAuthorityReadOnlyClient(saConn)
		}

		source, err = redis_responder.NewCheckedRedisSource(rocspSource, dbMap, sac, scope, logger)
		cmd.FailOnError(err, "Could not create checkedRedis source")

		// Load the certificate from the file path.
		issuerCerts := make([]*issuance.Certificate, len(c.OCSPResponder.IssuerCerts))
		for i, issuerFile := range c.OCSPResponder.IssuerCerts {
			issuerCert, err := issuance.LoadCertificate(issuerFile)
			cmd.FailOnError(err, "Could not load issuer cert")
			issuerCerts[i] = issuerCert
		}

		source, err = responder.NewFilterSource(
			issuerCerts,
			c.OCSPResponder.RequiredSerialPrefixes,
			source,
			scope,
			logger,
			clk,
		)
		cmd.FailOnError(err, "Could not create filtered source")
	}

	m := mux(c.OCSPResponder.Path, source, c.OCSPResponder.Timeout.Duration, scope, logger, c.OCSPResponder.LogSampleRate)

	srv := &http.Server{
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 120 * time.Second,
		IdleTimeout:  120 * time.Second,
		Addr:         c.OCSPResponder.ListenAddress,
		Handler:      m,
	}

	done := make(chan bool)
	go cmd.CatchSignals(logger, func() {
		ctx, cancel := context.WithTimeout(context.Background(),
			c.OCSPResponder.ShutdownStopTimeout.Duration)
		defer cancel()
		_ = srv.Shutdown(ctx)
		done <- true
	})

	err = srv.ListenAndServe()
	if err != nil && err != http.ErrServerClosed {
		cmd.FailOnError(err, "Running HTTP server")
	}

	// https://godoc.org/net/http#Server.Shutdown:
	// When Shutdown is called, Serve, ListenAndServe, and ListenAndServeTLS
	// immediately return ErrServerClosed. Make sure the program doesn't exit and
	// waits instead for Shutdown to return.
	<-done
}

// ocspMux partially implements the interface defined for http.ServeMux but doesn't implement
// the path cleaning its Handler method does. Notably http.ServeMux will collapse repeated
// slashes into a single slash which breaks the base64 encoding that is used in OCSP GET
// requests. ocsp.Responder explicitly recommends against using http.ServeMux
// for this reason.
type ocspMux struct {
	handler http.Handler
}

func (om *ocspMux) Handler(_ *http.Request) (http.Handler, string) {
	return om.handler, "/"
}

func mux(responderPath string, source responder.Source, timeout time.Duration, stats prometheus.Registerer, logger blog.Logger, sampleRate int) http.Handler {
	stripPrefix := http.StripPrefix(responderPath, responder.NewResponder(source, timeout, stats, logger, sampleRate))
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" && r.URL.Path == "/" {
			w.Header().Set("Cache-Control", "max-age=43200") // Cache for 12 hours
			w.WriteHeader(200)
			return
		}
		stripPrefix.ServeHTTP(w, r)
	})
	return hnynethttp.WrapHandler(measured_http.New(&ocspMux{h}, cmd.Clock(), stats))
}

func init() {
	cmd.RegisterCommand("ocsp-responder", main)
}
