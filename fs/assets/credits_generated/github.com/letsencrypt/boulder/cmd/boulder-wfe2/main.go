package notmain

import (
	"bytes"
	"context"
	"crypto/x509"
	"encoding/pem"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/honeycombio/beeline-go"
	"github.com/jmhodges/clock"
	"github.com/letsencrypt/boulder/cmd"
	"github.com/letsencrypt/boulder/features"
	"github.com/letsencrypt/boulder/goodkey"
	bgrpc "github.com/letsencrypt/boulder/grpc"
	"github.com/letsencrypt/boulder/grpc/noncebalancer"
	"github.com/letsencrypt/boulder/issuance"
	blog "github.com/letsencrypt/boulder/log"
	"github.com/letsencrypt/boulder/nonce"
	rapb "github.com/letsencrypt/boulder/ra/proto"
	sapb "github.com/letsencrypt/boulder/sa/proto"
	"github.com/letsencrypt/boulder/wfe2"
	"github.com/prometheus/client_golang/prometheus"
)

type Config struct {
	WFE struct {
		cmd.ServiceConfig
		ListenAddress    string
		TLSListenAddress string

		// Timeout is the per-request overall timeout. This should be slightly
		// lower than the upstream's timeout when making request to the WFE.
		Timeout cmd.ConfigDuration

		ServerCertificatePath string
		ServerKeyPath         string

		AllowOrigins []string

		ShutdownStopTimeout cmd.ConfigDuration

		SubscriberAgreementURL string

		TLS cmd.TLSConfig

		RAService *cmd.GRPCClientConfig
		SAService *cmd.GRPCClientConfig

		// GetNonceService is a gRPC config which contains a single SRV name
		// used to lookup nonce-service instances used exclusively for nonce
		// creation. In a multi-DC deployment this should refer to local
		// nonce-service instances only.
		GetNonceService *cmd.GRPCClientConfig

		// RedeemNonceServices contains a map of nonce-service prefixes to
		// gRPC configs we want to use to redeem nonces. In a multi-DC deployment
		// this should contain all nonce-services from all DCs as we want to be
		// able to redeem nonces generated at any DC.
		//
		// DEPRECATED: See RedeemNonceService, below.
		// TODO (#6610) Remove this after all configs have migrated to
		// `RedeemNonceService`.
		RedeemNonceServices map[string]cmd.GRPCClientConfig

		// RedeemNonceService is a gRPC config which contains a list of SRV
		// names used to lookup nonce-service instances used exclusively for
		// nonce redemption. In a multi-DC deployment this should contain both
		// local and remote nonce-service instances.
		RedeemNonceService *cmd.GRPCClientConfig

		// NoncePrefixKey is a secret used for deriving the prefix of each nonce
		// instance. It should contain 256 bits of random data to be suitable as
		// an HMAC-SHA256 key (e.g. the output of `openssl rand -hex 32`). In a
		// multi-DC deployment this value should be the same across all
		// boulder-wfe and nonce-service instances.
		NoncePrefixKey cmd.PasswordConfig

		// CertificateChains maps AIA issuer URLs to certificate filenames.
		// Certificates are read into the chain in the order they are defined in the
		// slice of filenames.
		// DEPRECATED: See Chains, below.
		// TODO(5164): Remove this after all configs have migrated to `Chains`.
		CertificateChains map[string][]string

		// AlternateCertificateChains maps AIA issuer URLs to an optional alternate
		// certificate chain, represented by an ordered slice of certificate filenames.
		// DEPRECATED: See Chains, below.
		// TODO(5164): Remove this after all configs have migrated to `Chains`.
		AlternateCertificateChains map[string][]string

		// Chains is a list of lists of certificate filenames. Each inner list is
		// a chain (starting with the issuing intermediate, followed by one or
		// more additional certificates, up to and including a root) which we are
		// willing to serve. Chains that start with a given intermediate will only
		// be offered for certificates which were issued by the key pair represented
		// by that intermediate. The first chain representing any given issuing
		// key pair will be the default for that issuer, served if the client does
		// not request a specific chain.
		// NOTE: This config field deprecates the CertificateChains and
		// AlternateCertificateChains fields. If it is present, those fields are
		// ignored. They will be removed in a future release.
		Chains [][]string

		Features map[string]bool

		// DirectoryCAAIdentity is used for the /directory response's "meta"
		// element's "caaIdentities" field. It should match the VA's "issuerDomain"
		// configuration value (this value is the one used to enforce CAA)
		DirectoryCAAIdentity string
		// DirectoryWebsite is used for the /directory response's "meta" element's
		// "website" field.
		DirectoryWebsite string

		// ACMEv2 requests (outside some registration/revocation messages) use a JWS with
		// a KeyID header containing the full account URL. For new accounts this
		// will be a KeyID based on the HTTP request's Host header and the ACMEv2
		// account path. For legacy ACMEv1 accounts we need to whitelist the account
		// ID prefix that legacy accounts would have been using based on the Host
		// header of the WFE1 instance and the legacy 'reg' path component. This
		// will differ in configuration for production and staging.
		LegacyKeyIDPrefix string

		// GoodKey is an embedded config stanza for the goodkey library.
		GoodKey goodkey.Config

		// StaleTimeout determines how old should data be to be accessed via Boulder-specific GET-able APIs
		StaleTimeout cmd.ConfigDuration

		// AuthorizationLifetimeDays defines how long authorizations will be
		// considered valid for. The WFE uses this to find the creation date of
		// authorizations by subtracing this value from the expiry. It should match
		// the value configured in the RA.
		AuthorizationLifetimeDays int

		// PendingAuthorizationLifetimeDays defines how long authorizations may be in
		// the pending state before expiry. The WFE uses this to find the creation
		// date of pending authorizations by subtracting this value from the expiry.
		// It should match the value configured in the RA.
		PendingAuthorizationLifetimeDays int

		AccountCache *CacheConfig
	}

	Syslog  cmd.SyslogConfig
	Beeline cmd.BeelineConfig
}

type CacheConfig struct {
	Size int
	TTL  cmd.ConfigDuration
}

// loadCertificateFile loads a PEM certificate from the certFile provided. It
// validates that the PEM is well-formed with no leftover bytes, and contains
// only a well-formed X509 CA certificate. If the cert file meets these
// requirements the PEM bytes from the file are returned along with the parsed
// certificate, otherwise an error is returned. If the PEM contents of
// a certFile do not have a trailing newline one is added.
// TODO(5164): Remove this after all configs have migrated to `Chains`.
func loadCertificateFile(aiaIssuerURL, certFile string) ([]byte, *issuance.Certificate, error) {
	pemBytes, err := os.ReadFile(certFile)
	if err != nil {
		return nil, nil, fmt.Errorf(
			"CertificateChain entry for AIA issuer url %q has an "+
				"invalid chain file: %q - error reading contents: %w",
			aiaIssuerURL, certFile, err)
	}
	if bytes.Contains(pemBytes, []byte("\r\n")) {
		return nil, nil, fmt.Errorf(
			"CertificateChain entry for AIA issuer url %q has an "+
				"invalid chain file: %q - contents had CRLF line endings",
			aiaIssuerURL, certFile)
	}
	// Try to decode the contents as PEM
	certBlock, rest := pem.Decode(pemBytes)
	if certBlock == nil {
		return nil, nil, fmt.Errorf(
			"CertificateChain entry for AIA issuer url %q has an "+
				"invalid chain file: %q - contents did not decode as PEM",
			aiaIssuerURL, certFile)
	}
	// The PEM contents must be a CERTIFICATE
	if certBlock.Type != "CERTIFICATE" {
		return nil, nil, fmt.Errorf(
			"CertificateChain entry for AIA issuer url %q has an "+
				"invalid chain file: %q - PEM block type incorrect, found "+
				"%q, expected \"CERTIFICATE\"",
			aiaIssuerURL, certFile, certBlock.Type)
	}
	// The PEM Certificate must successfully parse
	cert, err := x509.ParseCertificate(certBlock.Bytes)
	if err != nil {
		return nil, nil, fmt.Errorf(
			"CertificateChain entry for AIA issuer url %q has an "+
				"invalid chain file: %q - certificate bytes failed to parse: %w",
			aiaIssuerURL, certFile, err)
	}
	// If there are bytes leftover we must reject the file otherwise these
	// leftover bytes will end up in a served certificate chain.
	if len(rest) != 0 {
		return nil, nil, fmt.Errorf(
			"CertificateChain entry for AIA issuer url %q has an "+
				"invalid chain file: %q - PEM contents had unused remainder "+
				"input (%d bytes)",
			aiaIssuerURL, certFile, len(rest))
	}
	// If the PEM contents don't end in a \n, add it.
	if pemBytes[len(pemBytes)-1] != '\n' {
		pemBytes = append(pemBytes, '\n')
	}
	ic, err := issuance.NewCertificate(cert)
	if err != nil {
		return nil, nil, fmt.Errorf(
			"CertificateChain entry for AIA issuer url %q has an "+
				"invalid chain file: %q - unable to load issuer certificate: %w",
			aiaIssuerURL, certFile, err)
	}
	return pemBytes, ic, nil
}

// loadCertificateChains processes the provided chainConfig of AIA Issuer URLs
// and cert filenames. For each AIA issuer URL all of its cert filenames are
// read, validated as PEM certificates, and concatenated together separated by
// newlines. The combined PEM certificate chain contents for each are returned
// in the results map, keyed by the IssuerNameID. Additionally the first
// certificate in each chain is parsed and returned in a slice of issuer
// certificates.
// TODO(5164): Remove this after all configs have migrated to `Chains`.
func loadCertificateChains(chainConfig map[string][]string, requireAtLeastOneChain bool) (map[issuance.IssuerNameID][]byte, map[issuance.IssuerNameID]*issuance.Certificate, error) {
	results := make(map[issuance.IssuerNameID][]byte, len(chainConfig))
	issuerCerts := make(map[issuance.IssuerNameID]*issuance.Certificate, len(chainConfig))

	// For each AIA Issuer URL we need to read the chain cert files
	for aiaIssuerURL, certFiles := range chainConfig {
		var buffer bytes.Buffer

		// There must be at least one chain file specified
		if requireAtLeastOneChain && len(certFiles) == 0 {
			return nil, nil, fmt.Errorf(
				"CertificateChain entry for AIA issuer url %q has no chain "+
					"file names configured",
				aiaIssuerURL)
		}

		// certFiles are read and appended in the order they appear in the
		// configuration
		var id issuance.IssuerNameID
		for i, c := range certFiles {
			// Prepend a newline before each chain entry
			buffer.Write([]byte("\n"))

			// Read and validate the chain file contents
			pemBytes, cert, err := loadCertificateFile(aiaIssuerURL, c)
			if err != nil {
				return nil, nil, err
			}

			// Save the first certificate as a direct issuer certificate
			if i == 0 {
				id = cert.NameID()
				issuerCerts[id] = cert
			}

			// Write the PEM bytes to the result buffer for this AIAIssuer
			buffer.Write(pemBytes)
		}

		// Save the full PEM chain contents, if any
		if buffer.Len() > 0 {
			results[id] = buffer.Bytes()
		}
	}

	return results, issuerCerts, nil
}

// loadChain takes a list of filenames containing pem-formatted certificates,
// and returns a chain representing all of those certificates in order. It
// ensures that the resulting chain is valid. The final file is expected to be
// a root certificate, which the chain will be verified against, but which will
// not be included in the resulting chain.
func loadChain(certFiles []string) (*issuance.Certificate, []byte, error) {
	certs, err := issuance.LoadChain(certFiles)
	if err != nil {
		return nil, nil, err
	}

	// Iterate over all certs appending their pem to the buf.
	var buf bytes.Buffer
	for _, cert := range certs {
		buf.Write([]byte("\n"))
		buf.Write(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: cert.Raw}))
	}

	return certs[0], buf.Bytes(), nil
}

func setupWFE(c Config, scope prometheus.Registerer, clk clock.Clock) (rapb.RegistrationAuthorityClient, sapb.StorageAuthorityReadOnlyClient, nonce.Getter, map[string]nonce.Redeemer, nonce.Redeemer, string) {
	tlsConfig, err := c.WFE.TLS.Load()
	cmd.FailOnError(err, "TLS config")

	raConn, err := bgrpc.ClientSetup(c.WFE.RAService, tlsConfig, scope, clk)
	cmd.FailOnError(err, "Failed to load credentials and create gRPC connection to RA")
	rac := rapb.NewRegistrationAuthorityClient(raConn)

	saConn, err := bgrpc.ClientSetup(c.WFE.SAService, tlsConfig, scope, clk)
	cmd.FailOnError(err, "Failed to load credentials and create gRPC connection to SA")
	sac := sapb.NewStorageAuthorityReadOnlyClient(saConn)

	// TODO(#6610) Refactor these checks.
	if c.WFE.RedeemNonceService != nil && c.WFE.RedeemNonceServices != nil {
		cmd.Fail("Only one of 'redeemNonceService' or 'redeemNonceServices' should be configured.")
	}
	if c.WFE.RedeemNonceService == nil && c.WFE.RedeemNonceServices == nil {
		cmd.Fail("One of 'redeemNonceService' or 'redeemNonceServices' must be configured.")
	}
	if c.WFE.RedeemNonceService != nil && c.WFE.NoncePrefixKey.PasswordFile == "" {
		cmd.Fail("'noncePrefixKey' must be configured if 'redeemNonceService' is configured.")
	}
	if c.WFE.GetNonceService == nil {
		cmd.Fail("'getNonceService' must be configured")
	}

	var rncKey string
	if c.WFE.NoncePrefixKey.PasswordFile != "" {
		rncKey, err = c.WFE.NoncePrefixKey.Pass()
		cmd.FailOnError(err, "Failed to load noncePrefixKey")
	}

	getNonceConn, err := bgrpc.ClientSetup(c.WFE.GetNonceService, tlsConfig, scope, clk)
	cmd.FailOnError(err, "Failed to load credentials and create gRPC connection to get nonce service")
	gnc := nonce.NewGetter(getNonceConn)

	var rnc nonce.Redeemer
	var npm map[string]nonce.Redeemer
	if c.WFE.RedeemNonceService != nil {
		// Dispatch nonce redemption RPCs dynamically.
		if c.WFE.RedeemNonceService.SRVResolver != noncebalancer.SRVResolverScheme {
			cmd.Fail(fmt.Sprintf(
				"'redeemNonceService.SRVResolver' must be set to %q", noncebalancer.SRVResolverScheme),
			)
		}
		redeemNonceConn, err := bgrpc.ClientSetup(c.WFE.RedeemNonceService, tlsConfig, scope, clk)
		cmd.FailOnError(err, "Failed to load credentials and create gRPC connection to redeem nonce service")
		rnc = nonce.NewRedeemer(redeemNonceConn)
	} else {
		// Dispatch nonce redpemption RPCs using a static mapping.
		//
		// TODO(#6610) Remove code below and the `npm` mapping.
		npm = make(map[string]nonce.Redeemer)
		for prefix, serviceConfig := range c.WFE.RedeemNonceServices {
			serviceConfig := serviceConfig
			conn, err := bgrpc.ClientSetup(&serviceConfig, tlsConfig, scope, clk)
			cmd.FailOnError(err, "Failed to load credentials and create gRPC connection to redeem nonce service")
			npm[prefix] = nonce.NewRedeemer(conn)
		}
	}

	return rac, sac, gnc, npm, rnc, rncKey
}

type errorWriter struct {
	blog.Logger
}

func (ew errorWriter) Write(p []byte) (n int, err error) {
	// log.Logger will append a newline to all messages before calling
	// Write. Our log checksum checker doesn't like newlines, because
	// syslog will strip them out so the calculated checksums will
	// differ. So that we don't hit this corner case for every line
	// logged from inside net/http.Server we strip the newline before
	// we get to the checksum generator.
	p = bytes.TrimRight(p, "\n")
	ew.Logger.Err(fmt.Sprintf("net/http.Server: %s", string(p)))
	return
}

func main() {
	configFile := flag.String("config", "", "File path to the configuration file for this service")
	flag.Parse()
	if *configFile == "" {
		flag.Usage()
		os.Exit(1)
	}

	var c Config
	err := cmd.ReadConfigFile(*configFile, &c)
	cmd.FailOnError(err, "Reading JSON config file into config structure")

	err = features.Set(c.WFE.Features)
	cmd.FailOnError(err, "Failed to set feature flags")

	allCertChains := map[issuance.IssuerNameID][][]byte{}
	issuerCerts := map[issuance.IssuerNameID]*issuance.Certificate{}
	if c.WFE.Chains != nil {
		for _, files := range c.WFE.Chains {
			issuer, chain, err := loadChain(files)
			cmd.FailOnError(err, "Failed to load chain")

			id := issuer.NameID()
			allCertChains[id] = append(allCertChains[id], chain)
			// This may overwrite a previously-set issuerCert (e.g. if there are two
			// chains for the same issuer, but with different versions of the same
			// same intermediate issued by different roots). This is okay, as the
			// only truly important content here is the public key to verify other
			// certs.
			issuerCerts[id] = issuer
		}
	} else {
		// TODO(5164): Remove this after all configs have migrated to `Chains`.
		var certChains map[issuance.IssuerNameID][]byte
		certChains, issuerCerts, err = loadCertificateChains(c.WFE.CertificateChains, true)
		cmd.FailOnError(err, "Couldn't read configured CertificateChains")

		for nameID, chainPEM := range certChains {
			allCertChains[nameID] = [][]byte{chainPEM}
		}

		if c.WFE.AlternateCertificateChains != nil {
			altCertChains, _, err := loadCertificateChains(c.WFE.AlternateCertificateChains, false)
			cmd.FailOnError(err, "Couldn't read configured AlternateCertificateChains")

			for nameID, chainPEM := range altCertChains {
				if _, ok := allCertChains[nameID]; !ok {
					cmd.Fail(fmt.Sprintf("IssuerNameId %q appeared in AlternateCertificateChains, "+
						"but does not exist in CertificateChains", nameID))
				}
				allCertChains[nameID] = append(allCertChains[nameID], chainPEM)
			}
		}
	}

	bc, err := c.Beeline.Load()
	cmd.FailOnError(err, "Failed to load Beeline config")
	beeline.Init(bc)
	defer beeline.Close()

	stats, logger := cmd.StatsAndLogging(c.Syslog, c.WFE.DebugAddr)
	defer logger.AuditPanic()
	logger.Info(cmd.VersionString())

	clk := cmd.Clock()

	rac, sac, gnc, npm, rnc, npKey := setupWFE(c, stats, clk)

	kp, err := goodkey.NewKeyPolicy(&c.WFE.GoodKey, sac.KeyBlocked)
	cmd.FailOnError(err, "Unable to create key policy")

	if c.WFE.StaleTimeout.Duration == 0 {
		c.WFE.StaleTimeout.Duration = time.Minute * 10
	}

	// Baseline Requirements v1.8.1 section 4.2.1: "any reused data, document,
	// or completed validation MUST be obtained no more than 398 days prior
	// to issuing the Certificate". If unconfigured or the configured value is
	// greater than 397 days, bail out.
	if c.WFE.AuthorizationLifetimeDays <= 0 || c.WFE.AuthorizationLifetimeDays > 397 {
		cmd.Fail("authorizationLifetimeDays value must be greater than 0 and less than 398")
	}
	authorizationLifetime := time.Duration(c.WFE.AuthorizationLifetimeDays) * 24 * time.Hour

	// The Baseline Requirements v1.8.1 state that validation tokens "MUST
	// NOT be used for more than 30 days from its creation". If unconfigured
	// or the configured value pendingAuthorizationLifetimeDays is greater
	// than 29 days, bail out.
	if c.WFE.PendingAuthorizationLifetimeDays <= 0 || c.WFE.PendingAuthorizationLifetimeDays > 29 {
		cmd.Fail("pendingAuthorizationLifetimeDays value must be greater than 0 and less than 30")
	}
	pendingAuthorizationLifetime := time.Duration(c.WFE.PendingAuthorizationLifetimeDays) * 24 * time.Hour

	var accountGetter wfe2.AccountGetter
	if c.WFE.AccountCache != nil {
		accountGetter = wfe2.NewAccountCache(sac,
			c.WFE.AccountCache.Size,
			c.WFE.AccountCache.TTL.Duration,
			clk,
			stats)
	} else {
		accountGetter = sac
	}
	wfe, err := wfe2.NewWebFrontEndImpl(
		stats,
		clk,
		kp,
		allCertChains,
		issuerCerts,
		logger,
		c.WFE.Timeout.Duration,
		c.WFE.StaleTimeout.Duration,
		authorizationLifetime,
		pendingAuthorizationLifetime,
		rac,
		sac,
		gnc,
		npm,
		rnc,
		npKey,
		accountGetter,
	)
	cmd.FailOnError(err, "Unable to create WFE")

	wfe.SubscriberAgreementURL = c.WFE.SubscriberAgreementURL
	wfe.AllowOrigins = c.WFE.AllowOrigins
	wfe.DirectoryCAAIdentity = c.WFE.DirectoryCAAIdentity
	wfe.DirectoryWebsite = c.WFE.DirectoryWebsite
	wfe.LegacyKeyIDPrefix = c.WFE.LegacyKeyIDPrefix

	logger.Infof("WFE using key policy: %#v", kp)

	logger.Infof("Server running, listening on %s....", c.WFE.ListenAddress)
	handler := wfe.Handler(stats)

	srv := http.Server{
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 120 * time.Second,
		IdleTimeout:  120 * time.Second,
		Addr:         c.WFE.ListenAddress,
		ErrorLog:     log.New(errorWriter{logger}, "", 0),
		Handler:      handler,
	}

	go func() {
		err := srv.ListenAndServe()
		if err != nil && err != http.ErrServerClosed {
			cmd.FailOnError(err, "Running HTTP server")
		}
	}()

	// The gosec linter complains that ReadHeaderTimeout is not set. That's fine,
	// because that field inherits its value from ReadTimeout.
	////nolint:gosec
	tlsSrv := http.Server{
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 120 * time.Second,
		IdleTimeout:  120 * time.Second,
		Addr:         c.WFE.TLSListenAddress,
		ErrorLog:     log.New(errorWriter{logger}, "", 0),
		Handler:      handler,
	}
	if tlsSrv.Addr != "" {
		go func() {
			err := tlsSrv.ListenAndServeTLS(c.WFE.ServerCertificatePath, c.WFE.ServerKeyPath)
			if err != nil && err != http.ErrServerClosed {
				cmd.FailOnError(err, "Running TLS server")
			}
		}()
	}

	done := make(chan bool)
	go cmd.CatchSignals(logger, func() {
		ctx, cancel := context.WithTimeout(context.Background(), c.WFE.ShutdownStopTimeout.Duration)
		defer cancel()
		_ = srv.Shutdown(ctx)
		_ = tlsSrv.Shutdown(ctx)
		done <- true
	})

	// https://godoc.org/net/http#Server.Shutdown:
	// When Shutdown is called, Serve, ListenAndServe, and ListenAndServeTLS
	// immediately return ErrServerClosed. Make sure the program doesn't exit and
	// waits instead for Shutdown to return.
	<-done
}

func init() {
	cmd.RegisterCommand("boulder-wfe2", main)
}
