package notmain

import (
	"context"
	"flag"
	"os"
	"time"

	"github.com/honeycombio/beeline-go"

	capb "github.com/letsencrypt/boulder/ca/proto"
	"github.com/letsencrypt/boulder/cmd"
	cspb "github.com/letsencrypt/boulder/crl/storer/proto"
	"github.com/letsencrypt/boulder/crl/updater"
	"github.com/letsencrypt/boulder/features"
	bgrpc "github.com/letsencrypt/boulder/grpc"
	"github.com/letsencrypt/boulder/issuance"
	sapb "github.com/letsencrypt/boulder/sa/proto"
)

type Config struct {
	CRLUpdater struct {
		cmd.ServiceConfig

		SAService           *cmd.GRPCClientConfig
		CRLGeneratorService *cmd.GRPCClientConfig
		CRLStorerService    *cmd.GRPCClientConfig

		// IssuerCerts is a list of paths to issuer certificates on disk. This
		// controls the set of CRLs which will be published by this updater: it will
		// publish one set of NumShards CRL shards for each issuer in this list.
		IssuerCerts []string

		// NumShards is the number of shards into which each issuer's "full and
		// complete" CRL will be split.
		// WARNING: When this number is changed, the "JSON Array of CRL URLs" field
		// in CCADB MUST be updated.
		NumShards int

		// ShardWidth is the amount of time (width on a timeline) that a single
		// shard should cover. Ideally, NumShards*ShardWidth should be an amount of
		// time noticeably larger than the current longest certificate lifetime,
		// but the updater will continue to work if this is not the case (albeit
		// with more confusing mappings of serials to shards).
		// WARNING: When this number is changed, revocation entries will move
		// between shards.
		ShardWidth cmd.ConfigDuration

		// LookbackPeriod is how far back the updater should look for revoked expired
		// certificates. We are required to include every revoked cert in at least
		// one CRL, even if it is revoked seconds before it expires, so this must
		// always be greater than the UpdatePeriod, and should be increased when
		// recovering from an outage to ensure continuity of coverage.
		LookbackPeriod cmd.ConfigDuration

		// CertificateLifetime is the validity period (usually expressed in hours,
		// like "2160h") of the longest-lived currently-unexpired certificate. For
		// Let's Encrypt, this is usually ninety days. If the validity period of
		// the issued certificates ever changes upwards, this value must be updated
		// immediately; if the validity period of the issued certificates ever
		// changes downwards, the value must not change until after all certificates with
		// the old validity period have expired.
		// DEPRECATED: This config value is no longer used.
		// TODO(#6438): Remove this value.
		CertificateLifetime cmd.ConfigDuration

		// UpdatePeriod controls how frequently the crl-updater runs and publishes
		// new versions of every CRL shard. The Baseline Requirements, Section 4.9.7
		// state that this MUST NOT be more than 7 days. We believe that future
		// updates may require that this not be more than 24 hours, and currently
		// recommend an UpdatePeriod of 6 hours.
		UpdatePeriod cmd.ConfigDuration

		// UpdateOffset controls the times at which crl-updater runs, to avoid
		// scheduling the batch job at exactly midnight. The updater runs every
		// UpdatePeriod, starting from the Unix Epoch plus UpdateOffset, and
		// continuing forward into the future forever. This value must be strictly
		// less than the UpdatePeriod.
		UpdateOffset cmd.ConfigDuration

		// MaxParallelism controls how many workers may be running in parallel.
		// A higher value reduces the total time necessary to update all CRL shards
		// that this updater is responsible for, but also increases the memory used
		// by this updater.
		MaxParallelism int

		Features map[string]bool
	}

	Syslog  cmd.SyslogConfig
	Beeline cmd.BeelineConfig
}

func main() {
	configFile := flag.String("config", "", "File path to the configuration file for this service")
	debugAddr := flag.String("debug-addr", "", "Debug server address override")
	runOnce := flag.Bool("runOnce", false, "If true, run once immediately and then exit")
	flag.Parse()
	if *configFile == "" {
		flag.Usage()
		os.Exit(1)
	}

	var c Config
	err := cmd.ReadConfigFile(*configFile, &c)
	cmd.FailOnError(err, "Reading JSON config file into config structure")

	if *debugAddr != "" {
		c.CRLUpdater.DebugAddr = *debugAddr
	}

	err = features.Set(c.CRLUpdater.Features)
	cmd.FailOnError(err, "Failed to set feature flags")

	tlsConfig, err := c.CRLUpdater.TLS.Load()
	cmd.FailOnError(err, "TLS config")

	scope, logger := cmd.StatsAndLogging(c.Syslog, c.CRLUpdater.DebugAddr)
	defer logger.AuditPanic()
	logger.Info(cmd.VersionString())
	clk := cmd.Clock()

	bc, err := c.Beeline.Load()
	cmd.FailOnError(err, "Failed to load Beeline config")
	beeline.Init(bc)
	defer beeline.Close()

	issuers := make([]*issuance.Certificate, 0, len(c.CRLUpdater.IssuerCerts))
	for _, filepath := range c.CRLUpdater.IssuerCerts {
		cert, err := issuance.LoadCertificate(filepath)
		cmd.FailOnError(err, "Failed to load issuer cert")
		issuers = append(issuers, cert)
	}

	if c.CRLUpdater.ShardWidth.Duration == 0 {
		c.CRLUpdater.ShardWidth.Duration = 16 * time.Hour
	}
	if c.CRLUpdater.LookbackPeriod.Duration == 0 {
		c.CRLUpdater.LookbackPeriod.Duration = 24 * time.Hour
	}

	saConn, err := bgrpc.ClientSetup(c.CRLUpdater.SAService, tlsConfig, scope, clk)
	cmd.FailOnError(err, "Failed to load credentials and create gRPC connection to SA")
	sac := sapb.NewStorageAuthorityReadOnlyClient(saConn)

	caConn, err := bgrpc.ClientSetup(c.CRLUpdater.CRLGeneratorService, tlsConfig, scope, clk)
	cmd.FailOnError(err, "Failed to load credentials and create gRPC connection to CRLGenerator")
	cac := capb.NewCRLGeneratorClient(caConn)

	csConn, err := bgrpc.ClientSetup(c.CRLUpdater.CRLStorerService, tlsConfig, scope, clk)
	cmd.FailOnError(err, "Failed to load credentials and create gRPC connection to CRLStorer")
	csc := cspb.NewCRLStorerClient(csConn)

	u, err := updater.NewUpdater(
		issuers,
		c.CRLUpdater.NumShards,
		c.CRLUpdater.ShardWidth.Duration,
		c.CRLUpdater.LookbackPeriod.Duration,
		c.CRLUpdater.UpdatePeriod.Duration,
		c.CRLUpdater.UpdateOffset.Duration,
		c.CRLUpdater.MaxParallelism,
		sac,
		cac,
		csc,
		scope,
		logger,
		clk,
	)
	cmd.FailOnError(err, "Failed to create crl-updater")

	ctx, cancel := context.WithCancel(context.Background())
	go cmd.CatchSignals(logger, cancel)

	if *runOnce {
		err = u.Tick(ctx, clk.Now())
		cmd.FailOnError(err, "")
	} else {
		err = u.Run(ctx)
		if err != nil {
			logger.Err(err.Error())
		}
	}
}

func init() {
	cmd.RegisterCommand("crl-updater", main)
}
