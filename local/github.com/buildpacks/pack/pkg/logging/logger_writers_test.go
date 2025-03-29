package logging_test

import (
	"fmt"
	"io"
	"testing"
	"time"

	"github.com/apex/log"
	"github.com/heroku/color"
	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"

	"github.com/buildpacks/pack/internal/style"
	"github.com/buildpacks/pack/pkg/logging"
	h "github.com/buildpacks/pack/testhelpers"
)

func TestLogWithWriters(t *testing.T) {
	spec.Run(t, "LogWithWriters", testLogWithWriters, spec.Parallel(), spec.Report(report.Terminal{}))
}

func testLogWithWriters(t *testing.T, when spec.G, it spec.S) {
	var (
		logger           *logging.LogWithWriters
		outCons, errCons *color.Console
		fOut, fErr       func() string
		timeFmt          = "2006/01/02 15:04:05.000000"
		testTime         = "2019/05/15 01:01:01.000000"
	)

	it.Before(func() {
		outCons, fOut = h.MockWriterAndOutput()
		errCons, fErr = h.MockWriterAndOutput()
		logger = logging.NewLogWithWriters(outCons, errCons, logging.WithClock(func() time.Time {
			clock, _ := time.Parse(timeFmt, testTime)
			return clock
		}))
	})

	when("default", func() {
		it("has no time and color", func() {
			logger.Info(color.HiBlueString("test"))
			h.AssertEq(t, fOut(), "\x1b[94mtest\x1b[0m\n")
		})

		it("will not log debug messages", func() {
			logger.Debug("debug_")
			logger.Debugf("debugf")

			output := fOut()
			h.AssertNotContains(t, output, "debug_\n")
			h.AssertNotContains(t, output, "debugf\n")
		})

		it("logs info and warning messages to standard writer", func() {
			logger.Info("info_")
			logger.Infof("infof")
			logger.Warn("warn_")
			logger.Warnf("warnf")

			output := fOut()
			h.AssertContains(t, output, "info_\n")
			h.AssertContains(t, output, "infof\n")
			h.AssertContains(t, output, "warn_\n")
			h.AssertContains(t, output, "warnf\n")
		})

		it("logs error to error writer", func() {
			logger.Error("error_")
			logger.Errorf("errorf")

			output := fErr()
			h.AssertContains(t, output, "error_\n")
			h.AssertContains(t, output, "errorf\n")
		})

		it("will return correct writers", func() {
			h.AssertSameInstance(t, logger.Writer(), outCons)
			h.AssertSameInstance(t, logger.WriterForLevel(logging.DebugLevel), io.Discard)
		})

		it("is only verbose for debug level", func() {
			h.AssertFalse(t, logger.IsVerbose())

			logger.Level = log.DebugLevel
			h.AssertTrue(t, logger.IsVerbose())
		})
	})

	when("time is set to true", func() {
		it("time is logged in info", func() {
			logger.WantTime(true)
			logger.Info("test")
			h.AssertEq(t, fOut(), "2019/05/15 01:01:01.000000 test\n")
		})

		it("time is logged in error", func() {
			logger.WantTime(true)
			logger.Error("test")
			h.AssertEq(t, fErr(), fmt.Sprintf("2019/05/15 01:01:01.000000 %stest\n", style.Error("ERROR: ")))
		})

		when("WriterForLevel", func() {
			it("time is logged in info", func() {
				logger.WantTime(true)
				writer := logger.WriterForLevel(logging.InfoLevel)
				writer.Write([]byte("test\n"))
				h.AssertEq(t, fOut(), "2019/05/15 01:01:01.000000 test\n")
			})

			it("time is logged in error", func() {
				logger.WantTime(true)
				writer := logger.WriterForLevel(logging.ErrorLevel)
				writer.Write([]byte("test\n"))
				// The writer doesn't prepend the level
				h.AssertEq(t, fErr(), "2019/05/15 01:01:01.000000 test\n")
			})
		})
	})

	when("colors are disabled", func() {
		it("don't display colors", func() {
			outCons.DisableColors(true)
			logger.Info(color.HiBlueString("test"))
			h.AssertEq(t, fOut(), "test\n")
		})
	})

	when("quiet is set to true", func() {
		it.Before(func() {
			logger.WantQuiet(true)
		})

		it("will not log debug or info messages", func() {
			logger.Debug("debug_")
			logger.Debugf("debugf")
			logger.Info("info_")
			logger.Infof("infof")

			output := fOut()
			h.AssertNotContains(t, output, "debug_\n")
			h.AssertNotContains(t, output, "debugf\n")
			h.AssertNotContains(t, output, "info_\n")
			h.AssertNotContains(t, output, "infof\n")
		})

		it("logs warnings to standard writer", func() {
			logger.Warn("warn_")
			logger.Warnf("warnf")

			output := fOut()
			h.AssertContains(t, output, "warn_\n")
			h.AssertContains(t, output, "warnf\n")
		})

		it("logs error to error writer", func() {
			logger.Error("error_")
			logger.Errorf("errorf")

			output := fErr()
			h.AssertContains(t, output, "error_\n")
			h.AssertContains(t, output, "errorf\n")
		})

		it("will return correct writers", func() {
			h.AssertSameInstance(t, logger.Writer(), outCons)
			h.AssertSameInstance(t, logger.WriterForLevel(logging.DebugLevel), io.Discard)
			h.AssertSameInstance(t, logger.WriterForLevel(logging.InfoLevel), io.Discard)
		})
	})

	when("verbose is set to true", func() {
		it.Before(func() {
			logger.WantVerbose(true)
		})

		it("all messages are logged", func() {
			logger.Debug("debug_")
			logger.Debugf("debugf")
			logger.Info("info_")
			logger.Infof("infof")
			logger.Warn("warn_")
			logger.Warnf("warnf")

			output := fOut()
			h.AssertContains(t, output, "debug_")
			h.AssertContains(t, output, "debugf")
			h.AssertContains(t, output, "info_")
			h.AssertContains(t, output, "infof")
			h.AssertContains(t, output, "warn_")
			h.AssertContains(t, output, "warnf")
		})

		it("logs error to error writer", func() {
			logger.Error("error_")
			logger.Errorf("errorf")

			output := fErr()
			h.AssertContains(t, output, "error_\n")
			h.AssertContains(t, output, "errorf\n")
		})

		it("will return correct writers", func() {
			h.AssertSameInstance(t, logger.Writer(), outCons)
			assertLogWriterHasOut(t, logger.WriterForLevel(logging.DebugLevel), outCons)
			assertLogWriterHasOut(t, logger.WriterForLevel(logging.InfoLevel), outCons)
			assertLogWriterHasOut(t, logger.WriterForLevel(logging.WarnLevel), outCons)
			assertLogWriterHasOut(t, logger.WriterForLevel(logging.ErrorLevel), errCons)
		})
	})

	it("will convert an empty string to a line feed", func() {
		logger.Info("")
		expected := "\n"
		h.AssertEq(t, fOut(), expected)
	})
}

func assertLogWriterHasOut(t *testing.T, writer io.Writer, out io.Writer) {
	logWriter, ok := writer.(hasWriter)
	h.AssertTrue(t, ok)
	h.AssertSameInstance(t, logWriter.Writer(), out)
}

type hasWriter interface {
	Writer() io.Writer
}
