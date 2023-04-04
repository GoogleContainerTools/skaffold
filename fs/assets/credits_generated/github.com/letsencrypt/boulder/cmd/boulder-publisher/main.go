package notmain

import (
	"flag"
	"fmt"
	"os"
	"runtime"

	ct "github.com/google/certificate-transparency-go"
	"github.com/honeycombio/beeline-go"

	"github.com/letsencrypt/boulder/cmd"
	"github.com/letsencrypt/boulder/features"
	bgrpc "github.com/letsencrypt/boulder/grpc"
	"github.com/letsencrypt/boulder/issuance"
	"github.com/letsencrypt/boulder/publisher"
	pubpb "github.com/letsencrypt/boulder/publisher/proto"
)

type Config struct {
	Publisher struct {
		cmd.ServiceConfig
		Features map[string]bool

		// If this is non-zero, profile blocking events such that one even is
		// sampled every N nanoseconds.
		// https://golang.org/pkg/runtime/#SetBlockProfileRate
		BlockProfileRate int
		UserAgent        string

		// Chains is a list of lists of certificate filenames. Each inner list is
		// a chain, starting with the issuing intermediate, followed by one or
		// more additional certificates, up to and including a root.
		Chains [][]string
	}

	Syslog  cmd.SyslogConfig
	Beeline cmd.BeelineConfig
}

func main() {
	grpcAddr := flag.String("addr", "", "gRPC listen address override")
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
	err = features.Set(c.Publisher.Features)
	cmd.FailOnError(err, "Failed to set feature flags")

	runtime.SetBlockProfileRate(c.Publisher.BlockProfileRate)

	if *grpcAddr != "" {
		c.Publisher.GRPC.Address = *grpcAddr
	}
	if *debugAddr != "" {
		c.Publisher.DebugAddr = *debugAddr
	}
	if c.Publisher.UserAgent == "" {
		c.Publisher.UserAgent = "certificate-transparency-go/1.0"
	}

	bc, err := c.Beeline.Load()
	cmd.FailOnError(err, "Failed to load Beeline config")
	beeline.Init(bc)
	defer beeline.Close()

	scope, logger := cmd.StatsAndLogging(c.Syslog, c.Publisher.DebugAddr)
	defer logger.AuditPanic()
	logger.Info(cmd.VersionString())

	if c.Publisher.Chains == nil {
		logger.AuditErr("No chain files provided")
		os.Exit(1)
	}

	bundles := make(map[issuance.IssuerNameID][]ct.ASN1Cert)
	for _, files := range c.Publisher.Chains {
		chain, err := issuance.LoadChain(files)
		cmd.FailOnError(err, "failed to load chain.")
		issuer := chain[0]
		id := issuer.NameID()
		if _, exists := bundles[id]; exists {
			cmd.Fail(fmt.Sprintf("Got multiple chains configured for issuer %q", issuer.Subject.CommonName))
		}
		bundles[id] = publisher.GetCTBundleForChain(chain)
	}

	tlsConfig, err := c.Publisher.TLS.Load()
	cmd.FailOnError(err, "TLS config")

	clk := cmd.Clock()

	pubi := publisher.New(bundles, c.Publisher.UserAgent, logger, scope)

	start, stop, err := bgrpc.NewServer(c.Publisher.GRPC).Add(
		&pubpb.Publisher_ServiceDesc, pubi).Build(tlsConfig, scope, clk)
	cmd.FailOnError(err, "Unable to setup Publisher gRPC server")

	go cmd.CatchSignals(logger, stop)
	cmd.FailOnError(start(), "Publisher gRPC service failed")
}

func init() {
	cmd.RegisterCommand("boulder-publisher", main)
}
