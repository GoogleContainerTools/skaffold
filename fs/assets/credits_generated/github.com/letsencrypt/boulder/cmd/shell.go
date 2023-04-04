// This package provides utilities that underlie the specific commands.
package cmd

import (
	"context"
	"encoding/json"
	"expvar"
	"fmt"
	"log"
	"log/syslog"
	"net/http"
	"net/http/pprof"
	"os"
	"os/signal"
	"path"
	"runtime"
	"strings"
	"syscall"
	"time"

	"google.golang.org/grpc/grpclog"

	"github.com/go-redis/redis/v8"
	"github.com/go-sql-driver/mysql"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/letsencrypt/boulder/core"
	blog "github.com/letsencrypt/boulder/log"
)

func command() string {
	return path.Base(os.Args[0])
}

// Because we don't know when this init will be called with respect to
// flag.Parse() and other flag definitions, we can't rely on the regular
// flag mechanism. But this one is fine.
func init() {
	for _, v := range os.Args {
		if v == "--version" || v == "-version" {
			fmt.Println(VersionString())
			os.Exit(0)
		}
	}
}

// mysqlLogger implements the mysql.Logger interface.
type mysqlLogger struct {
	blog.Logger
}

func (m mysqlLogger) Print(v ...interface{}) {
	m.AuditErrf("[mysql] %s", fmt.Sprint(v...))
}

// grpcLogger implements the grpclog.LoggerV2 interface.
type grpcLogger struct {
	blog.Logger
}

// Ensure that fatal logs exit, because we use neither the gRPC default logger
// nor the stdlib default logger, both of which would call os.Exit(1) for us.
func (log grpcLogger) Fatal(args ...interface{}) {
	log.Error(args...)
	os.Exit(1)
}
func (log grpcLogger) Fatalf(format string, args ...interface{}) {
	log.Errorf(format, args...)
	os.Exit(1)
}
func (log grpcLogger) Fatalln(args ...interface{}) {
	log.Errorln(args...)
	os.Exit(1)
}

// Treat all gRPC error logs as potential audit events.
func (log grpcLogger) Error(args ...interface{}) {
	log.Logger.AuditErr(fmt.Sprint(args...))
}
func (log grpcLogger) Errorf(format string, args ...interface{}) {
	log.Logger.AuditErrf(format, args...)
}
func (log grpcLogger) Errorln(args ...interface{}) {
	log.Logger.AuditErr(fmt.Sprintln(args...))
}

// Pass through most Warnings, but filter out a few noisy ones.
func (log grpcLogger) Warning(args ...interface{}) {
	log.Logger.Warning(fmt.Sprint(args...))
}
func (log grpcLogger) Warningf(format string, args ...interface{}) {
	log.Logger.Warningf(format, args...)
}
func (log grpcLogger) Warningln(args ...interface{}) {
	msg := fmt.Sprintln(args...)
	// See https://github.com/letsencrypt/boulder/issues/4628
	if strings.Contains(msg, `ccResolverWrapper: error parsing service config: no JSON service config provided`) {
		return
	}
	// See https://github.com/letsencrypt/boulder/issues/4379
	if strings.Contains(msg, `Server.processUnaryRPC failed to write status: connection error: desc = "transport is closing"`) {
		return
	}
	// Since we've already formatted the message, just pass through to .Warning()
	log.Logger.Warning(msg)
}

// Don't log any INFO-level gRPC stuff. In practice this is all noise, like
// failed TXT lookups for service discovery (we only use A records).
func (log grpcLogger) Info(args ...interface{})                 {}
func (log grpcLogger) Infof(format string, args ...interface{}) {}
func (log grpcLogger) Infoln(args ...interface{})               {}

// V returns true if the verbosity level l is less than the verbosity we want to
// log at.
func (log grpcLogger) V(l int) bool {
	// We always return false. This causes gRPC to not log some things which are
	// only logged conditionally if the logLevel is set below a certain value.
	// TODO: Use the wrapped log.Logger.stdoutLevel and log.Logger.syslogLevel
	// to determine a correct return value here.
	return false
}

// promLogger implements the promhttp.Logger interface.
type promLogger struct {
	blog.Logger
}

func (log promLogger) Println(args ...interface{}) {
	log.AuditErr(fmt.Sprint(args...))
}

type redisLogger struct {
	blog.Logger
}

func (rl redisLogger) Printf(ctx context.Context, format string, v ...interface{}) {
	rl.Infof(format, v...)
}

// logWriter implements the io.Writer interface.
type logWriter struct {
	blog.Logger
}

func (lw logWriter) Write(p []byte) (n int, err error) {
	// Lines received by logWriter will always have a trailing newline.
	lw.Logger.Info(strings.Trim(string(p), "\n"))
	return
}

// StatsAndLogging constructs a prometheus registerer and an AuditLogger based
// on its config parameters, and return them both. It also spawns off an HTTP
// server on the provided port to report the stats and provide pprof profiling
// handlers. NewLogger and newStatsRegistry will call os.Exit on errors.
// Also sets the constructed AuditLogger as the default logger, and configures
// the mysql and grpc packages to use our logger.
// This must be called before any gRPC code is called, because gRPC's SetLogger
// doesn't use any locking.
func StatsAndLogging(logConf SyslogConfig, addr string) (prometheus.Registerer, blog.Logger) {
	logger := NewLogger(logConf)
	return newStatsRegistry(addr, logger), logger
}

func NewLogger(logConf SyslogConfig) blog.Logger {
	var logger blog.Logger
	if logConf.SyslogLevel >= 0 {
		syslogger, err := syslog.Dial(
			"",
			"",
			syslog.LOG_INFO, // default, not actually used
			command())
		FailOnError(err, "Could not connect to Syslog")
		syslogLevel := int(syslog.LOG_INFO)
		if logConf.SyslogLevel != 0 {
			syslogLevel = logConf.SyslogLevel
		}
		logger, err = blog.New(syslogger, logConf.StdoutLevel, syslogLevel)
		FailOnError(err, "Could not connect to Syslog")
	} else {
		logger = blog.StdoutLogger(logConf.StdoutLevel)
	}

	_ = blog.Set(logger)
	_ = mysql.SetLogger(mysqlLogger{logger})
	grpclog.SetLoggerV2(grpcLogger{logger})
	log.SetOutput(logWriter{logger})
	redis.SetLogger(redisLogger{logger})

	// Periodically log the current timestamp, to ensure syslog timestamps match
	// Boulder's conception of time.
	go func() {
		for {
			time.Sleep(time.Minute)
			logger.Info(fmt.Sprintf("time=%s", time.Now().Format(time.RFC3339Nano)))
		}
	}()
	return logger
}

func newVersionCollector() prometheus.Collector {
	buildTime := core.Unspecified
	if core.GetBuildTime() != core.Unspecified {
		// core.BuildTime is set by our Makefile using the shell command 'date
		// -u' which outputs in a consistent format across all POSIX systems.
		bt, err := time.Parse(time.UnixDate, core.BuildTime)
		if err != nil {
			// Should never happen unless the Makefile is changed.
			buildTime = "Unparsable"
		} else {
			buildTime = bt.Format(time.RFC3339)
		}
	}
	return prometheus.NewGaugeFunc(
		prometheus.GaugeOpts{
			Name: "version",
			Help: fmt.Sprintf(
				"A metric with a constant value of '1' labeled by the short commit-id (buildId), build timestamp in RFC3339 format (buildTime), and Go release tag like 'go1.3' (goVersion) from which %s was built.",
				command(),
			),
			ConstLabels: prometheus.Labels{
				"buildId":   core.GetBuildID(),
				"buildTime": buildTime,
				"goVersion": runtime.Version(),
			},
		},
		func() float64 { return 1 },
	)
}

func newStatsRegistry(addr string, logger blog.Logger) prometheus.Registerer {
	registry := prometheus.NewRegistry()
	registry.MustRegister(collectors.NewGoCollector())
	registry.MustRegister(collectors.NewProcessCollector(
		collectors.ProcessCollectorOpts{}))
	registry.MustRegister(newVersionCollector())

	mux := http.NewServeMux()
	// Register the available pprof handlers. These are all registered on
	// DefaultServeMux just by importing pprof, but since we eschew
	// DefaultServeMux, we need to explicitly register them on our own mux.
	mux.Handle("/debug/pprof/", http.HandlerFunc(pprof.Index))
	mux.Handle("/debug/pprof/profile", http.HandlerFunc(pprof.Profile))
	mux.Handle("/debug/pprof/symbol", http.HandlerFunc(pprof.Symbol))
	mux.Handle("/debug/pprof/trace", http.HandlerFunc(pprof.Trace))
	// These handlers are defined in runtime/pprof instead of net/http/pprof, and
	// have to be accessed through net/http/pprof's Handler func.
	mux.Handle("/debug/pprof/goroutine", pprof.Handler("goroutine"))
	mux.Handle("/debug/pprof/block", pprof.Handler("block"))
	mux.Handle("/debug/pprof/heap", pprof.Handler("heap"))
	mux.Handle("/debug/pprof/mutex", pprof.Handler("mutex"))
	mux.Handle("/debug/pprof/threadcreate", pprof.Handler("threadcreate"))

	mux.Handle("/debug/vars", expvar.Handler())
	mux.Handle("/metrics", promhttp.HandlerFor(registry, promhttp.HandlerOpts{
		ErrorLog: promLogger{logger},
	}))

	server := http.Server{
		Addr:        addr,
		Handler:     mux,
		ReadTimeout: time.Minute,
	}
	go func() {
		err := server.ListenAndServe()
		if err != nil {
			logger.Errf("unable to boot debug server on %s: %v", addr, err)
			os.Exit(1)
		}
	}()
	return registry
}

// Fail exits and prints an error message to stderr and the logger audit log.
func Fail(msg string) {
	logger := blog.Get()
	logger.AuditErr(msg)
	os.Exit(1)
}

// FailOnError exits and prints an error message, but only if we encountered
// a problem and err != nil. err is required but msg can be "".
func FailOnError(err error, msg string) {
	if err == nil {
		return
	}
	if msg == "" {
		Fail(err.Error())
	} else {
		Fail(fmt.Sprintf("%s: %s", msg, err))
	}
}

// ReadConfigFile takes a file path as an argument and attempts to
// unmarshal the content of the file into a struct containing a
// configuration of a boulder component. Any config keys in the JSON
// file which do not correspond to expected keys in the config struct
// will result in errors.
func ReadConfigFile(filename string, out interface{}) error {
	file, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	decoder := json.NewDecoder(file)
	decoder.DisallowUnknownFields()
	return decoder.Decode(out)
}

// VersionString produces a friendly Application version string.
func VersionString() string {
	return fmt.Sprintf("Versions: %s=(%s %s) Golang=(%s) BuildHost=(%s)", command(), core.GetBuildID(), core.GetBuildTime(), runtime.Version(), core.GetBuildHost())
}

// CatchSignals catches SIGTERM, SIGINT, SIGHUP and executes a callback
// method before exiting
func CatchSignals(logger blog.Logger, callback func()) {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGTERM)
	signal.Notify(sigChan, syscall.SIGINT)
	signal.Notify(sigChan, syscall.SIGHUP)

	<-sigChan
	if callback != nil {
		callback()
	}

	os.Exit(0)
}
