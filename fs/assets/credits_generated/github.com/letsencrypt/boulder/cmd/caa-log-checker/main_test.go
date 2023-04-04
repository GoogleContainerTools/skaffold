package notmain

import (
	"compress/gzip"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/letsencrypt/boulder/test"
)

// A timestamp which matches the format we put in our logs. Note that it has
// sub-second precision only out to microseconds (not nanoseconds), and must
// include the timezone indicator.
// 0001-01-01T01:01:01.001001+00:00
var testTime = time.Time{}.Add(time.Hour + time.Minute + time.Second + time.Millisecond + time.Microsecond).Local()

func TestOpenFile(t *testing.T) {
	tmpPlain, err := os.CreateTemp(os.TempDir(), "plain")
	test.AssertNotError(t, err, "failed to create temporary file")
	defer os.Remove(tmpPlain.Name())
	_, err = tmpPlain.Write([]byte("test-1\ntest-2"))
	test.AssertNotError(t, err, "failed to write to temp file")
	tmpPlain.Close()

	tmpGzip, err := os.CreateTemp(os.TempDir(), "gzip-*.gz")
	test.AssertNotError(t, err, "failed to create temporary file")
	defer os.Remove(tmpGzip.Name())
	gzipWriter := gzip.NewWriter(tmpGzip)
	_, err = gzipWriter.Write([]byte("test-1\ntest-2"))
	test.AssertNotError(t, err, "failed to write to temp file")
	gzipWriter.Flush()
	gzipWriter.Close()
	tmpGzip.Close()

	checkFile := func(path string) {
		t.Helper()
		scanner, err := openFile(path)
		test.AssertNotError(t, err, fmt.Sprintf("failed to open %q", path))
		var lines []string
		for scanner.Scan() {
			lines = append(lines, scanner.Text())
		}
		test.AssertNotError(t, scanner.Err(), fmt.Sprintf("failed to read from %q", path))
		test.AssertEquals(t, len(lines), 2)
		test.AssertDeepEquals(t, lines, []string{"test-1", "test-2"})
	}

	checkFile(tmpPlain.Name())
	checkFile(tmpGzip.Name())
}

func TestLoadIssuanceLog(t *testing.T) {

	for _, tc := range []struct {
		name        string
		loglines    string
		expMap      map[string][]time.Time
		expEarliest time.Time
		expLatest   time.Time
		expErrStr   string
	}{
		{
			"empty file",
			"",
			map[string][]time.Time{},
			time.Time{},
			time.Time{},
			"",
		},
		{
			"no matches",
			"some text\nsome other text",
			map[string][]time.Time{},
			time.Time{},
			time.Time{},
			"",
		},
		{
			"bad json",
			"Certificate request - successful JSON=this is not valid json",
			map[string][]time.Time{},
			time.Time{},
			time.Time{},
			"failed to unmarshal JSON",
		},
		{
			"bad timestamp",
			"2009-11-10 23:00:00 UTC Certificate request - successful JSON={}",
			map[string][]time.Time{},
			time.Time{},
			time.Time{},
			"failed to parse timestamp",
		},
		{
			"normal behavior",
			`header
0001-01-01T01:01:01.001001+00:00 Certificate request - successful JSON={"SerialNumber": "1", "Names":["example.com"], "Requester":0}
0001-01-01T02:01:01.001001+00:00 Certificate request - successful JSON={"SerialNumber": "2", "Names":["2.example.com", "3.example.com"], "Requester":0}
filler
0001-01-01T03:01:01.001001+00:00 Certificate request - successful JSON={"SerialNumber": "3", "Names":["2.example.com"], "Requester":0}
trailer`,
			map[string][]time.Time{
				"example.com":   {testTime},
				"2.example.com": {testTime.Add(time.Hour), testTime.Add(2 * time.Hour)},
				"3.example.com": {testTime.Add(time.Hour)},
			},
			testTime,
			testTime.Add(2 * time.Hour),
			"",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			tmp, err := os.CreateTemp(os.TempDir(), "TestLoadIssuanceLog")
			test.AssertNotError(t, err, "failed to create temporary log file")
			defer os.Remove(tmp.Name())
			_, err = tmp.Write([]byte(tc.loglines))
			test.AssertNotError(t, err, "failed to write temporary log file")
			err = tmp.Close()
			test.AssertNotError(t, err, "failed to close temporary log file")

			resMap, resEarliest, resLatest, resError := loadIssuanceLog(tmp.Name())
			if tc.expErrStr != "" {
				test.AssertError(t, resError, "loadIssuanceLog should have errored")
				test.AssertContains(t, resError.Error(), tc.expErrStr)
				return
			}
			test.AssertNotError(t, resError, "loadIssuanceLog shouldn't have errored")
			test.AssertDeepEquals(t, resMap, tc.expMap)
			test.AssertEquals(t, resEarliest, tc.expEarliest)
			test.AssertEquals(t, resLatest, tc.expLatest)
		})
	}
}

func TestProcessCAALog(t *testing.T) {
	for _, tc := range []struct {
		name      string
		loglines  string
		issuances map[string][]time.Time
		earliest  time.Time
		latest    time.Time
		tolerance time.Duration
		expMap    map[string][]time.Time
		expErrStr string
	}{
		{
			"empty file",
			"",
			map[string][]time.Time{"example.com": {testTime}},
			time.Time{},
			time.Time{},
			time.Second,
			map[string][]time.Time{"example.com": {testTime}},
			"",
		},
		{
			"no matches",
			"",
			map[string][]time.Time{"example.com": {testTime}},
			time.Time{},
			time.Time{},
			time.Second,
			map[string][]time.Time{"example.com": {testTime}},
			"",
		},
		{
			"outside 8hr window",
			`header
0001-01-01T01:01:01.001001+00:00 Checked CAA records for example.com, [Present: true, ...
filler
0001-01-01T21:01:01.001001+00:00 Checked CAA records for example.com, [Present: true, ...
trailer`,
			map[string][]time.Time{"example.com": {testTime.Add(10 * time.Hour)}},
			testTime,
			testTime.Add(24 * time.Hour),
			time.Second,
			map[string][]time.Time{"example.com": {testTime.Add(10 * time.Hour)}},
			"",
		},
		{
			"outside earliest and latest",
			`header
0001-01-01T01:01:01.001001+00:00 Checked CAA records for example.com, [Present: true, ...
filler
0001-01-01T21:01:01.001001+00:00 Checked CAA records for example.com, [Present: true, ...
trailer`,
			map[string][]time.Time{"example.com": {testTime.Add(24 * time.Hour)}},
			testTime.Add(10 * time.Hour),
			testTime.Add(11 * time.Hour),
			time.Second,
			map[string][]time.Time{"example.com": {testTime.Add(24 * time.Hour)}},
			"",
		},
		{
			"present: false",
			`header
0001-01-01T01:01:01.001001+00:00 Checked CAA records for a.b.example.com, [Present: false, ...
trailer`,
			map[string][]time.Time{
				"a.b.example.com": {testTime.Add(time.Hour)},
				"b.example.com":   {testTime.Add(time.Hour)},
				"example.com":     {testTime.Add(time.Hour)},
				"other.com":       {testTime.Add(time.Hour)},
			},
			testTime,
			testTime.Add(2 * time.Hour),
			time.Second,
			map[string][]time.Time{"other.com": {testTime.Add(time.Hour)}},
			"",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fmt.Println(tc.name)
			tmp, err := os.CreateTemp(os.TempDir(), "TestProcessCAALog")
			test.AssertNotError(t, err, "failed to create temporary log file")
			defer os.Remove(tmp.Name())
			_, err = tmp.Write([]byte(tc.loglines))
			test.AssertNotError(t, err, "failed to write temporary log file")
			err = tmp.Close()
			test.AssertNotError(t, err, "failed to close temporary log file")

			resError := processCAALog(tmp.Name(), tc.issuances, tc.earliest, tc.latest, tc.tolerance)
			if tc.expErrStr != "" {
				test.AssertError(t, resError, "processCAALog should have errored")
				test.AssertContains(t, resError.Error(), tc.expErrStr)
				return
			}
			// Because processCAALog modifies its input map, we have to compare the
			// testcase's input against the testcase's expectation.
			test.AssertDeepEquals(t, tc.issuances, tc.expMap)
		})
	}
}

func TestRemoveCoveredTimestamps(t *testing.T) {
	for _, tc := range []struct {
		name       string
		timestamps []time.Time
		cover      time.Time
		tolerance  time.Duration
		expected   []time.Time
	}{
		{
			"empty input",
			[]time.Time{},
			testTime,
			time.Second,
			[]time.Time{},
		},
		{
			"normal functioning",
			[]time.Time{testTime.Add(-1 * time.Hour), testTime.Add(5 * time.Hour), testTime.Add(10 * time.Hour)},
			testTime,
			time.Second,
			[]time.Time{testTime.Add(-1 * time.Hour), testTime.Add(10 * time.Hour)},
		},
		{
			"tolerance",
			[]time.Time{testTime.Add(-1 * time.Second), testTime.Add(8*time.Hour + 1*time.Second)},
			testTime,
			time.Second,
			[]time.Time{},
		},
		{
			"intolerance",
			[]time.Time{testTime.Add(-2 * time.Second), testTime.Add(8*time.Hour + 2*time.Second)},
			testTime,
			time.Second,
			[]time.Time{testTime.Add(-2 * time.Second), testTime.Add(8*time.Hour + 2*time.Second)},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			result := removeCoveredTimestamps(tc.timestamps, tc.cover, tc.tolerance)
			test.AssertDeepEquals(t, result, tc.expected)
		})
	}
}
