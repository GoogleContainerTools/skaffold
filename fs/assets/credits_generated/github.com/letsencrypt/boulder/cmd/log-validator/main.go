package notmain

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/honeycombio/beeline-go"
	"github.com/hpcloud/tail"
	"github.com/prometheus/client_golang/prometheus"

	"github.com/letsencrypt/boulder/cmd"
	blog "github.com/letsencrypt/boulder/log"
)

var errInvalidChecksum = errors.New("invalid checksum length")

func lineValid(text string) error {
	// Line format should match the following rsyslog omfile template:
	//
	//   template( name="LELogFormat" type="list" ) {
	//  	property(name="timereported" dateFormat="rfc3339")
	//  	constant(value=" ")
	//  	property(name="hostname" field.delimiter="46" field.number="1")
	//  	constant(value=" datacenter ")
	//  	property(name="syslogseverity")
	//  	constant(value=" ")
	//  	property(name="syslogtag")
	//  	property(name="msg" spifno1stsp="on" )
	//  	property(name="msg" droplastlf="on" )
	//  	constant(value="\n")
	//   }
	//
	// This should result in a log line that looks like this:
	//   timestamp hostname datacenter syslogseverity binary-name[pid]: checksum msg

	fields := strings.Split(text, " ")
	const errorPrefix = "log-validator:"
	// Extract checksum from line
	if len(fields) < 6 {
		return fmt.Errorf("%s line doesn't match expected format", errorPrefix)
	}
	checksum := fields[5]
	_, err := base64.RawURLEncoding.DecodeString(checksum)
	if err != nil || len(checksum) != 7 {
		return fmt.Errorf(
			"%s expected a 7 character base64 raw URL decodable string, got %q: %w",
			errorPrefix,
			checksum,
			errInvalidChecksum,
		)
	}

	// Reconstruct just the message portion of the line
	line := strings.Join(fields[6:], " ")

	// If we are fed our own output, treat it as always valid. This
	// prevents runaway scenarios where we generate ever-longer output.
	if strings.Contains(text, errorPrefix) {
		return nil
	}
	// Check the extracted checksum against the computed checksum
	if computedChecksum := blog.LogLineChecksum(line); checksum != computedChecksum {
		return fmt.Errorf("%s invalid checksum (expected %q, got %q)", errorPrefix, computedChecksum, checksum)
	}
	return nil
}

func validateFile(filename string) error {
	file, err := os.ReadFile(filename)
	if err != nil {
		return err
	}
	badFile := false
	for i, line := range strings.Split(string(file), "\n") {
		if line == "" {
			continue
		}
		err := lineValid(line)
		if err != nil {
			badFile = true
			fmt.Fprintf(os.Stderr, "[line %d] %s: %s\n", i+1, err, line)
		}
	}

	if badFile {
		return errors.New("file contained invalid lines")
	}
	return nil
}

// tailLogger is an adapter to the hpcloud/tail module's logging interface.
type tailLogger struct {
	blog.Logger
}

func (tl tailLogger) Fatal(v ...interface{}) {
	tl.AuditErr(fmt.Sprint(v...))
}
func (tl tailLogger) Fatalf(format string, v ...interface{}) {
	tl.AuditErrf(format, v...)
}
func (tl tailLogger) Fatalln(v ...interface{}) {
	tl.AuditErr(fmt.Sprint(v...) + "\n")
}
func (tl tailLogger) Panic(v ...interface{}) {
	tl.AuditErr(fmt.Sprint(v...))
}
func (tl tailLogger) Panicf(format string, v ...interface{}) {
	tl.AuditErrf(format, v...)
}
func (tl tailLogger) Panicln(v ...interface{}) {
	tl.AuditErr(fmt.Sprint(v...) + "\n")
}
func (tl tailLogger) Print(v ...interface{}) {
	tl.Info(fmt.Sprint(v...))
}
func (tl tailLogger) Printf(format string, v ...interface{}) {
	tl.Infof(format, v...)
}
func (tl tailLogger) Println(v ...interface{}) {
	tl.Info(fmt.Sprint(v...) + "\n")
}

type Config struct {
	Files []string

	DebugAddr string
	Syslog    cmd.SyslogConfig
	Beeline   cmd.BeelineConfig
}

func main() {
	configPath := flag.String("config", "", "File path to the configuration file for this service")
	checkFile := flag.String("check-file", "", "File path to a file to directly validate, if this argument is provided the config will not be parsed and only this file will be inspected")
	flag.Parse()

	if *checkFile != "" {
		err := validateFile(*checkFile)
		cmd.FailOnError(err, "validation failed")
		return
	}

	configBytes, err := os.ReadFile(*configPath)
	cmd.FailOnError(err, "failed to read config file")
	var config Config
	err = json.Unmarshal(configBytes, &config)
	cmd.FailOnError(err, "failed to parse config file")

	bc, err := config.Beeline.Load()
	cmd.FailOnError(err, "Failed to load Beeline config")
	beeline.Init(bc)
	defer beeline.Close()

	stats, logger := cmd.StatsAndLogging(config.Syslog, config.DebugAddr)
	lineCounter := prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "log_lines",
		Help: "A counter of log lines processed, with status",
	}, []string{"filename", "status"})
	stats.MustRegister(lineCounter)

	// Emit no more than 1 error line per second. This prevents consuming large
	// amounts of disk space in case there is problem that causes all log lines to
	// be invalid.
	outputLimiter := time.NewTicker(time.Second)

	var tailers []*tail.Tail
	for _, filename := range config.Files {
		t, err := tail.TailFile(filename, tail.Config{
			ReOpen:    true,
			MustExist: false, // sometimes files won't exist, so we must tolerate that
			Follow:    true,
			Logger:    tailLogger{logger},
		})
		cmd.FailOnError(err, "failed to tail file")

		go func() {
			for line := range t.Lines {
				if line.Err != nil {
					logger.Errf("error while tailing %s: %s", t.Filename, line.Err)
					continue
				}
				err := lineValid(line.Text)
				if err != nil {
					if errors.Is(err, errInvalidChecksum) {
						lineCounter.WithLabelValues(t.Filename, "invalid checksum length").Inc()
					} else {
						lineCounter.WithLabelValues(t.Filename, "bad").Inc()
					}
					select {
					case <-outputLimiter.C:
						logger.Errf("%s: %s %q", t.Filename, err, line.Text)
					default:
					}
				} else {
					lineCounter.WithLabelValues(t.Filename, "ok").Inc()
				}
			}
		}()

		tailers = append(tailers, t)
	}

	cmd.CatchSignals(logger, func() {
		for _, t := range tailers {
			// The tail module seems to have a race condition that will generate
			// errors like this on shutdown:
			// failed to stop tailing file: <filename>: Failed to detect creation of
			// <filename>: inotify watcher has been closed
			// This is probably related to the module's shutdown logic triggering the
			// "reopen" code path for files that are removed and then recreated.
			// These errors are harmless so we ignore them to allow clean shutdown.
			_ = t.Stop()
			t.Cleanup()
		}
	})
}

func init() {
	cmd.RegisterCommand("log-validator", main)
}
