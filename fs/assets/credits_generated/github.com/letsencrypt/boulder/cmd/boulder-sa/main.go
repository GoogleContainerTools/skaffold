package notmain

import (
	"flag"
	"os"

	"github.com/honeycombio/beeline-go"
	"github.com/letsencrypt/boulder/cmd"
	"github.com/letsencrypt/boulder/features"
	bgrpc "github.com/letsencrypt/boulder/grpc"
	rocsp_config "github.com/letsencrypt/boulder/rocsp/config"
	"github.com/letsencrypt/boulder/sa"
	sapb "github.com/letsencrypt/boulder/sa/proto"
)

type Config struct {
	SA struct {
		cmd.ServiceConfig
		DB          cmd.DBConfig
		ReadOnlyDB  cmd.DBConfig
		IncidentsDB cmd.DBConfig
		Redis       *rocsp_config.RedisConfig
		// TODO(#6285): Remove this field, as it is no longer used.
		Issuers map[string]int

		Features map[string]bool

		// Max simultaneous SQL queries caused by a single RPC.
		ParallelismPerRPC int
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

	err = features.Set(c.SA.Features)
	cmd.FailOnError(err, "Failed to set feature flags")

	if *grpcAddr != "" {
		c.SA.GRPC.Address = *grpcAddr
	}
	if *debugAddr != "" {
		c.SA.DebugAddr = *debugAddr
	}

	bc, err := c.Beeline.Load()
	cmd.FailOnError(err, "Failed to load Beeline config")
	beeline.Init(bc)
	defer beeline.Close()

	scope, logger := cmd.StatsAndLogging(c.Syslog, c.SA.DebugAddr)
	defer logger.AuditPanic()
	logger.Info(cmd.VersionString())

	dbMap, err := sa.InitWrappedDb(c.SA.DB, scope, logger)
	cmd.FailOnError(err, "While initializing dbMap")

	dbReadOnlyMap := dbMap
	if c.SA.ReadOnlyDB != (cmd.DBConfig{}) {
		dbReadOnlyMap, err = sa.InitWrappedDb(c.SA.ReadOnlyDB, scope, logger)
		cmd.FailOnError(err, "While initializing dbReadOnlyMap")
	}

	dbIncidentsMap := dbMap
	if c.SA.IncidentsDB != (cmd.DBConfig{}) {
		dbIncidentsMap, err = sa.InitWrappedDb(c.SA.IncidentsDB, scope, logger)
		cmd.FailOnError(err, "While initializing dbIncidentsMap")
	}

	clk := cmd.Clock()

	parallel := c.SA.ParallelismPerRPC
	if parallel < 1 {
		parallel = 1
	}

	tls, err := c.SA.TLS.Load()
	cmd.FailOnError(err, "TLS config")

	saroi, err := sa.NewSQLStorageAuthorityRO(dbReadOnlyMap, dbIncidentsMap, parallel, clk, logger)
	cmd.FailOnError(err, "Failed to create read-only SA impl")

	sai, err := sa.NewSQLStorageAuthorityWrapping(saroi, dbMap, scope)
	cmd.FailOnError(err, "Failed to create SA impl")

	start, stop, err := bgrpc.NewServer(c.SA.GRPC).Add(
		&sapb.StorageAuthorityReadOnly_ServiceDesc, saroi).Add(
		&sapb.StorageAuthority_ServiceDesc, sai).Build(
		tls, scope, clk)
	cmd.FailOnError(err, "Unable to setup SA gRPC server")

	go cmd.CatchSignals(logger, stop)
	cmd.FailOnError(start(), "SA gRPC service failed")
}

func init() {
	cmd.RegisterCommand("boulder-sa", main)
}
