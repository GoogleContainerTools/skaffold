package notmain

import (
	"flag"
	"os"
	"sync"

	"github.com/beeker1121/goque"
	"github.com/honeycombio/beeline-go"
	"github.com/prometheus/client_golang/prometheus"

	"github.com/letsencrypt/boulder/ca"
	capb "github.com/letsencrypt/boulder/ca/proto"
	"github.com/letsencrypt/boulder/cmd"
	"github.com/letsencrypt/boulder/ctpolicy/loglist"
	"github.com/letsencrypt/boulder/features"
	"github.com/letsencrypt/boulder/goodkey"
	bgrpc "github.com/letsencrypt/boulder/grpc"
	"github.com/letsencrypt/boulder/issuance"
	"github.com/letsencrypt/boulder/linter"
	"github.com/letsencrypt/boulder/policy"
	sapb "github.com/letsencrypt/boulder/sa/proto"
)

type Config struct {
	CA struct {
		cmd.ServiceConfig

		DB cmd.DBConfig
		cmd.HostnamePolicyConfig

		GRPCCA            *cmd.GRPCServerConfig
		GRPCOCSPGenerator *cmd.GRPCServerConfig
		GRPCCRLGenerator  *cmd.GRPCServerConfig

		SAService *cmd.GRPCClientConfig

		// Issuance contains all information necessary to load and initialize issuers.
		Issuance struct {
			Profile      issuance.ProfileConfig
			Issuers      []issuance.IssuerConfig
			IgnoredLints []string
		}

		// How long issued certificates are valid for.
		Expiry cmd.ConfigDuration

		// How far back certificates should be backdated.
		Backdate cmd.ConfigDuration

		// What digits we should prepend to serials after randomly generating them.
		SerialPrefix int

		// The maximum number of subjectAltNames in a single certificate
		MaxNames int

		// LifespanOCSP is how long OCSP responses are valid for. It should be
		// longer than the minTimeToExpiry field for the OCSP Updater. Per the BRs,
		// Section 4.9.10, it MUST NOT be more than 10 days.
		LifespanOCSP cmd.ConfigDuration

		// LifespanCRL is how long CRLs are valid for. It should be longer than the
		// `period` field of the CRL Updater. Per the BRs, Section 4.9.7, it MUST
		// NOT be more than 10 days.
		LifespanCRL cmd.ConfigDuration

		// GoodKey is an embedded config stanza for the goodkey library.
		GoodKey goodkey.Config

		// Path to directory holding orphan queue files, if not provided an orphan queue
		// is not used.
		OrphanQueueDir string

		// Maximum length (in bytes) of a line accumulating OCSP audit log entries.
		// Recommended to be around 4000. If this is 0, do not perform OCSP audit
		// logging.
		OCSPLogMaxLength int

		// Maximum period (in Go duration format) to wait to accumulate a max-length
		// OCSP audit log line. We will emit a log line at least once per period,
		// if there is anything to be logged. Keeping this low minimizes the risk
		// of losing logs during a catastrophic failure. Making it too high
		// means logging more often than necessary, which is inefficient in terms
		// of bytes and log system resources.
		// Recommended to be around 500ms.
		OCSPLogPeriod cmd.ConfigDuration

		// Path of a YAML file containing the list of int64 RegIDs
		// allowed to request ECDSA issuance
		ECDSAAllowListFilename string

		// CTLogListFile is the path to a JSON file on disk containing the set of
		// all logs trusted by Chrome. The file must match the v3 log list schema:
		// https://www.gstatic.com/ct/log_list/v3/log_list_schema.json
		CTLogListFile string

		// CRLDPBase is the piece of the CRL Distribution Point URI which is common
		// across all issuers and shards. It must use the http:// scheme, and must
		// not end with a slash. Example: "http://prod.c.lencr.org".
		CRLDPBase string

		// DisableCertService causes the CertificateAuthority gRPC service to not
		// start, preventing any certificates or precertificates from being issued.
		DisableCertService bool
		// DisableCertService causes the OCSPGenerator gRPC service to not start,
		// preventing any OCSP responses from being issued.
		DisableOCSPService bool
		// DisableCRLService causes the CRLGenerator gRPC service to not start,
		// preventing any CRLs from being issued.
		DisableCRLService bool

		Features map[string]bool
	}

	PA cmd.PAConfig

	Syslog  cmd.SyslogConfig
	Beeline cmd.BeelineConfig
}

func loadBoulderIssuers(profileConfig issuance.ProfileConfig, issuerConfigs []issuance.IssuerConfig, ignoredLints []string) ([]*issuance.Issuer, error) {
	issuers := make([]*issuance.Issuer, 0, len(issuerConfigs))
	for _, issuerConfig := range issuerConfigs {
		profile, err := issuance.NewProfile(profileConfig, issuerConfig)
		if err != nil {
			return nil, err
		}

		cert, signer, err := issuance.LoadIssuer(issuerConfig.Location)
		if err != nil {
			return nil, err
		}

		linter, err := linter.New(cert.Certificate, signer, ignoredLints)
		if err != nil {
			return nil, err
		}

		issuer, err := issuance.NewIssuer(cert, signer, profile, linter, cmd.Clock())
		if err != nil {
			return nil, err
		}

		issuers = append(issuers, issuer)
	}
	return issuers, nil
}

func main() {
	caAddr := flag.String("ca-addr", "", "CA gRPC listen address override")
	ocspAddr := flag.String("ocsp-addr", "", "OCSP gRPC listen address override")
	crlAddr := flag.String("crl-addr", "", "CRL gRPC listen address override")
	debugAddr := flag.String("debug-addr", "", "Debug server address override")
	configFile := flag.String("config", "", "File path to the configuration file for this service")
	flag.Parse()
	if *configFile == "" {
		flag.Usage()
		os.Exit(1)
	}

	var c Config
	err := cmd.ReadConfigFile(*configFile, &c)
	cmd.FailOnError(err, "Reading JSON config file into config structure")

	err = features.Set(c.CA.Features)
	cmd.FailOnError(err, "Failed to set feature flags")

	if *caAddr != "" {
		c.CA.GRPCCA.Address = *caAddr
	}
	if *ocspAddr != "" {
		c.CA.GRPCOCSPGenerator.Address = *ocspAddr
	}
	// TODO(#6161): Remove second conditional when we know it always exists.
	if *crlAddr != "" && c.CA.GRPCCRLGenerator != nil {
		c.CA.GRPCCRLGenerator.Address = *crlAddr
	}
	if *debugAddr != "" {
		c.CA.DebugAddr = *debugAddr
	}

	if c.CA.MaxNames == 0 {
		cmd.Fail("Error in CA config: MaxNames must not be 0")
	}

	bc, err := c.Beeline.Load()
	cmd.FailOnError(err, "Failed to load Beeline config")
	beeline.Init(bc)
	defer beeline.Close()

	scope, logger := cmd.StatsAndLogging(c.Syslog, c.CA.DebugAddr)
	defer logger.AuditPanic()
	logger.Info(cmd.VersionString())

	// These two metrics are created and registered here so they can be shared
	// between NewCertificateAuthorityImpl and NewOCSPImpl.
	signatureCount := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "signatures",
			Help: "Number of signatures",
		},
		[]string{"purpose", "issuer"})
	scope.MustRegister(signatureCount)

	signErrorCount := prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "signature_errors",
		Help: "A counter of signature errors labelled by error type",
	}, []string{"type"})
	scope.MustRegister(signErrorCount)

	cmd.FailOnError(c.PA.CheckChallenges(), "Invalid PA configuration")

	pa, err := policy.New(c.PA.Challenges, logger)
	cmd.FailOnError(err, "Couldn't create PA")

	if c.CA.HostnamePolicyFile == "" {
		cmd.Fail("HostnamePolicyFile was empty")
	}
	err = pa.SetHostnamePolicyFile(c.CA.HostnamePolicyFile)
	cmd.FailOnError(err, "Couldn't load hostname policy file")

	// Do this before creating the issuers to ensure the log list is loaded before
	// the linters are initialized.
	if c.CA.CTLogListFile != "" {
		err = loglist.InitLintList(c.CA.CTLogListFile)
		cmd.FailOnError(err, "Failed to load CT Log List")
	}

	var boulderIssuers []*issuance.Issuer
	boulderIssuers, err = loadBoulderIssuers(c.CA.Issuance.Profile, c.CA.Issuance.Issuers, c.CA.Issuance.IgnoredLints)
	cmd.FailOnError(err, "Couldn't load issuers")

	tlsConfig, err := c.CA.TLS.Load()
	cmd.FailOnError(err, "TLS config")

	clk := cmd.Clock()

	conn, err := bgrpc.ClientSetup(c.CA.SAService, tlsConfig, scope, clk)
	cmd.FailOnError(err, "Failed to load credentials and create gRPC connection to SA")
	sa := sapb.NewStorageAuthorityClient(conn)

	kp, err := goodkey.NewKeyPolicy(&c.CA.GoodKey, sa.KeyBlocked)
	cmd.FailOnError(err, "Unable to create key policy")

	var orphanQueue *goque.Queue
	if c.CA.OrphanQueueDir != "" {
		orphanQueue, err = goque.OpenQueue(c.CA.OrphanQueueDir)
		cmd.FailOnError(err, "Failed to open orphaned certificate queue")
		defer func() { _ = orphanQueue.Close() }()
	}

	var ecdsaAllowList *ca.ECDSAAllowList
	if c.CA.ECDSAAllowListFilename != "" {
		// Create a gauge vector to track allow list reloads.
		allowListGauge := prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "ecdsa_allow_list_status",
			Help: "Number of ECDSA allow list entries and status of most recent update attempt",
		}, []string{"result"})
		scope.MustRegister(allowListGauge)

		// Create a reloadable allow list object.
		var entries int
		ecdsaAllowList, entries, err = ca.NewECDSAAllowListFromFile(c.CA.ECDSAAllowListFilename, logger, allowListGauge)
		cmd.FailOnError(err, "Unable to load ECDSA allow list from YAML file")
		logger.Infof("Created a reloadable allow list, it was initialized with %d entries", entries)

	}

	var wg sync.WaitGroup
	stopFns := make([]func(), 0)

	// TODO(#6448): Remove this predeclaration when NewCertificateAuthorityImpl
	// no longer needs ocspi as an argument.
	var ocspi ca.OCSPGenerator
	if !c.CA.DisableOCSPService {
		ocspiReal, err := ca.NewOCSPImpl(
			boulderIssuers,
			c.CA.LifespanOCSP.Duration,
			c.CA.OCSPLogMaxLength,
			c.CA.OCSPLogPeriod.Duration,
			logger,
			scope,
			signatureCount,
			signErrorCount,
			clk,
		)
		cmd.FailOnError(err, "Failed to create OCSP impl")
		go ocspiReal.LogOCSPLoop()

		ocspStart, ocspStop, err := bgrpc.NewServer(c.CA.GRPCOCSPGenerator).Add(
			&capb.OCSPGenerator_ServiceDesc, ocspiReal).Build(tlsConfig, scope, clk)
		cmd.FailOnError(err, "Unable to setup CA OCSP gRPC server")

		wg.Add(1)
		go func() {
			cmd.FailOnError(ocspStart(), "OCSPGenerator gRPC service failed")
			wg.Done()
		}()
		stopFns = append(stopFns, ocspStop)
		ocspi = ca.OCSPGenerator(ocspiReal)
	} else {
		ocspi = ca.NewDisabledOCSPImpl()
	}

	// TODO(#6448): Remove this predeclaration when NewCertificateAuthorityImpl
	// no longer needs crli as an argument.
	var crli capb.CRLGeneratorServer
	if !c.CA.DisableCRLService {
		crli, err = ca.NewCRLImpl(
			boulderIssuers,
			c.CA.LifespanCRL.Duration,
			c.CA.CRLDPBase,
			c.CA.OCSPLogMaxLength,
			logger,
		)
		cmd.FailOnError(err, "Failed to create CRL impl")

		crlStart, crlStop, err := bgrpc.NewServer(c.CA.GRPCCRLGenerator).Add(
			&capb.CRLGenerator_ServiceDesc, crli).Build(tlsConfig, scope, clk)
		cmd.FailOnError(err, "Unable to setup CA CRL gRPC server")

		wg.Add(1)
		go func() {
			cmd.FailOnError(crlStart(), "CRLGenerator gRPC service failed")
			wg.Done()
		}()
		stopFns = append(stopFns, crlStop)
	} else {
		crli = ca.NewDisabledCRLImpl()
	}

	if !c.CA.DisableCertService {
		cai, err := ca.NewCertificateAuthorityImpl(
			sa,
			pa,
			ocspi,
			crli,
			boulderIssuers,
			ecdsaAllowList,
			c.CA.Expiry.Duration,
			c.CA.Backdate.Duration,
			c.CA.SerialPrefix,
			c.CA.MaxNames,
			kp,
			orphanQueue,
			logger,
			scope,
			signatureCount,
			signErrorCount,
			clk)
		cmd.FailOnError(err, "Failed to create CA impl")

		if orphanQueue != nil {
			go cai.OrphanIntegrationLoop()
		}

		srv := bgrpc.NewServer(c.CA.GRPCCA)

		// TODO(#6448): Move all of the impl construction inside these conditionals
		// as well, once the separate CRL and OCSP servers above have been removed.
		if !c.CA.DisableCertService {
			srv = srv.Add(&capb.CertificateAuthority_ServiceDesc, cai)
		}
		if !c.CA.DisableOCSPService {
			srv = srv.Add(&capb.OCSPGenerator_ServiceDesc, ocspi)
		}
		if !c.CA.DisableCRLService {
			srv = srv.Add(&capb.CRLGenerator_ServiceDesc, crli)
		}
		caStart, caStop, err := srv.Build(tlsConfig, scope, clk)
		cmd.FailOnError(err, "Unable to setup CA gRPC server")

		wg.Add(1)
		go func() {
			cmd.FailOnError(caStart(), "CA gRPC service failed")
			wg.Done()
		}()
		stopFns = append(stopFns, caStop)
	}

	go cmd.CatchSignals(logger, func() {
		ecdsaAllowList.Stop()
		for _, stopFn := range stopFns {
			stopFn()
		}
		wg.Wait()
	})

	select {}
}

func init() {
	cmd.RegisterCommand("boulder-ca", main)
}
