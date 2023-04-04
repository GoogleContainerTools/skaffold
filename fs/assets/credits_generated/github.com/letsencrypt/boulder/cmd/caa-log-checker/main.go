package notmain

import (
	"bufio"
	"compress/gzip"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/letsencrypt/boulder/cmd"
	blog "github.com/letsencrypt/boulder/log"
)

var raIssuanceLineRE = regexp.MustCompile(`Certificate request - successful JSON=(.*)`)

// TODO: Extract the "Valid for issuance: (true|false)" field too.
var vaCAALineRE = regexp.MustCompile(`Checked CAA records for ([a-z0-9-.*]+), \[Present: (true|false)`)

type issuanceEvent struct {
	SerialNumber string
	Names        []string
	Requester    int64

	issuanceTime time.Time
}

func openFile(path string) (*bufio.Scanner, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	var reader io.Reader
	reader = f
	if strings.HasSuffix(path, ".gz") {
		reader, err = gzip.NewReader(f)
		if err != nil {
			return nil, err
		}
	}
	scanner := bufio.NewScanner(reader)
	return scanner, nil
}

func parseTimestamp(line []byte) (time.Time, error) {
	datestamp, err := time.Parse(time.RFC3339, string(line[0:32]))
	if err != nil {
		return time.Time{}, err
	}
	return datestamp, nil
}

// loadIssuanceLog processes a single issuance (RA) log file. It returns a map
// of names to slices of timestamps at which certificates for those names were
// issued. It also returns the earliest and latest timestamps seen, to allow
// CAA log processing to quickly skip irrelevant entries.
func loadIssuanceLog(path string) (map[string][]time.Time, time.Time, time.Time, error) {
	scanner, err := openFile(path)
	if err != nil {
		return nil, time.Time{}, time.Time{}, fmt.Errorf("failed to open %q: %w", path, err)
	}

	linesCount := 0
	earliest := time.Time{}
	latest := time.Time{}

	issuanceMap := map[string][]time.Time{}
	for scanner.Scan() {
		line := scanner.Bytes()
		linesCount++

		matches := raIssuanceLineRE.FindSubmatch(line)
		if matches == nil {
			continue
		}
		if len(matches) != 2 {
			return nil, earliest, latest, fmt.Errorf("line %d: unexpected number of regex matches", linesCount)
		}

		var ie issuanceEvent
		err := json.Unmarshal(matches[1], &ie)
		if err != nil {
			return nil, earliest, latest, fmt.Errorf("line %d: failed to unmarshal JSON: %w", linesCount, err)
		}

		// Populate the issuance time from the syslog timestamp, rather than the
		// ResponseTime member of the JSON. This makes testing a lot simpler because
		// of how we mess with time sometimes. Given that these timestamps are
		// generated on the same system, they should be tightly coupled anyway.
		ie.issuanceTime, err = parseTimestamp(line)
		if err != nil {
			return nil, earliest, latest, fmt.Errorf("line %d: failed to parse timestamp: %w", linesCount, err)
		}

		if earliest.IsZero() || ie.issuanceTime.Before(earliest) {
			earliest = ie.issuanceTime
		}
		if latest.IsZero() || ie.issuanceTime.After(latest) {
			latest = ie.issuanceTime
		}
		for _, name := range ie.Names {
			issuanceMap[name] = append(issuanceMap[name], ie.issuanceTime)
		}
	}
	err = scanner.Err()
	if err != nil {
		return nil, earliest, latest, err
	}

	return issuanceMap, earliest, latest, nil
}

// processCAALog processes a single CAA (VA) log file. It modifies the input map
// (of issuance names to times, as returned by `loadIssuanceLog`) to remove any
// timestamps which are covered by (i.e. less than 8 hours after) a CAA check
// for that name in the log file. It also prunes any names whose slice of
// issuance times becomes empty.
func processCAALog(path string, issuances map[string][]time.Time, earliest time.Time, latest time.Time, tolerance time.Duration) error {
	scanner, err := openFile(path)
	if err != nil {
		return fmt.Errorf("failed to open %q: %w", path, err)
	}

	linesCount := 0

	for scanner.Scan() {
		line := scanner.Bytes()
		linesCount++

		matches := vaCAALineRE.FindSubmatch(line)
		if matches == nil {
			continue
		}
		if len(matches) != 3 {
			return fmt.Errorf("line %d: unexpected number of regex matches", linesCount)
		}
		name := string(matches[1])
		present := string(matches[2])

		checkTime, err := parseTimestamp(line)
		if err != nil {
			return fmt.Errorf("line %d: failed to parse timestamp: %w", linesCount, err)
		}

		// Don't bother processing rows that definitely fall outside the period we
		// care about.
		if checkTime.After(latest) || checkTime.Before(earliest.Add(-8*time.Hour)) {
			continue
		}

		// TODO: Only remove covered issuance timestamps if the CAA check actually
		// said that we're allowed to issue (i.e. had "Valid for issuance: true").
		issuances[name] = removeCoveredTimestamps(issuances[name], checkTime, tolerance)
		if len(issuances[name]) == 0 {
			delete(issuances, name)
		}

		// If the CAA check didn't find any CAA records for w.x.y.z, then that means
		// that we checked the CAA records for x.y.z, y.z, and z as well, and are
		// covered for any issuance for those names.
		if present == "false" {
			labels := strings.Split(name, ".")
			for i := 1; i < len(labels)-1; i++ {
				tailName := strings.Join(labels[i:], ".")
				issuances[tailName] = removeCoveredTimestamps(issuances[tailName], checkTime, tolerance)
				if len(issuances[tailName]) == 0 {
					delete(issuances, tailName)
				}
			}
		}
	}

	return scanner.Err()
}

// removeCoveredTimestamps returns a new slice of timestamps which contains all
// timestamps that are *not* within 8 hours after the input timestamp.
func removeCoveredTimestamps(timestamps []time.Time, cover time.Time, tolerance time.Duration) []time.Time {
	r := make([]time.Time, 0)
	for _, ts := range timestamps {
		// Copy the timestamp into the results slice if it is before the covering
		// timestamp, or more than 8 hours after the covering timestamp (i.e. if
		// it is *not* covered by the covering timestamp).
		diff := ts.Sub(cover)
		if diff < -tolerance || diff > 8*time.Hour+tolerance {
			ts := ts
			r = append(r, ts)
		}
	}
	return r
}

// emitErrors returns nil if the input map is empty. Otherwise, it logs
// a line for each name and issuance time that was not covered by a CAA
// check, and return an error.
func emitErrors(log blog.Logger, remaining map[string][]time.Time) error {
	if len(remaining) == 0 {
		return nil
	}

	for name, timestamps := range remaining {
		for _, timestamp := range timestamps {
			log.Infof("CAA-checking log event not found for issuance of %s at %s", name, timestamp)
		}
	}

	return errors.New("Some CAA-checking log events not found")
}

func main() {
	logStdoutLevel := flag.Int("stdout-level", 6, "Minimum severity of messages to send to stdout")
	logSyslogLevel := flag.Int("syslog-level", 6, "Minimum severity of messages to send to syslog")
	raLog := flag.String("ra-log", "", "Path to a single boulder-ra log file")
	vaLogs := flag.String("va-logs", "", "List of paths to boulder-va logs, separated by commas")
	timeTolerance := flag.Duration("time-tolerance", 0, "How much slop to allow when comparing timestamps for ordering")
	earliestFlag := flag.String("earliest", "", "Deprecated.")
	latestFlag := flag.String("latest", "", "Deprecated.")

	flag.Parse()

	logger := cmd.NewLogger(cmd.SyslogConfig{
		StdoutLevel: *logStdoutLevel,
		SyslogLevel: *logSyslogLevel,
	})

	if *timeTolerance < 0 {
		cmd.Fail("value of -time-tolerance must be non-negative")
	}

	if *earliestFlag != "" || *latestFlag != "" {
		logger.Info("The -earliest and -latest flags are deprecated and ignored.")
	}

	// Build a map from hostnames to times at which those names were issued for.
	// Also retrieve the earliest and latest issuance times represented in the
	// data, so we can be more efficient when examining entries from the CAA log.
	issuanceMap, earliest, latest, err := loadIssuanceLog(*raLog)
	cmd.FailOnError(err, "failed to load issuance logs")

	// Try to pare the issuance map down to nothing by removing every entry which
	// is covered by a CAA check.
	for _, vaLog := range strings.Split(*vaLogs, ",") {
		err = processCAALog(vaLog, issuanceMap, earliest, latest, *timeTolerance)
		cmd.FailOnError(err, "failed to process CAA checking logs")
	}

	err = emitErrors(logger, issuanceMap)
	if err != nil {
		logger.AuditErrf("%s", err)
		os.Exit(1)
	}
}

func init() {
	cmd.RegisterCommand("caa-log-checker", main)
}
