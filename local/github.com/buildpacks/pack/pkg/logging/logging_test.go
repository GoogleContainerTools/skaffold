package logging_test

import (
	"bytes"
	"testing"

	"github.com/heroku/color"
	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"

	"github.com/buildpacks/pack/pkg/logging"
	h "github.com/buildpacks/pack/testhelpers"
)

func TestLogging(t *testing.T) {
	color.Disable(true)
	defer color.Disable(false)
	spec.Run(t, "Logging", testLogging, spec.Parallel(), spec.Report(report.Terminal{}))
}

func testLogging(t *testing.T, when spec.G, it spec.S) {
	when("#GetWriterForLevel", func() {
		when("isSelectableWriter", func() {
			it("returns Logger for appropriate level", func() {
				outCons, output := h.MockWriterAndOutput()
				errCons, errOutput := h.MockWriterAndOutput()
				logger := logging.NewLogWithWriters(outCons, errCons)

				infoLogger := logging.GetWriterForLevel(logger, logging.InfoLevel)
				_, _ = infoLogger.Write([]byte("info test"))
				h.AssertEq(t, output(), "info test")

				errorLogger := logging.GetWriterForLevel(logger, logging.ErrorLevel)
				_, _ = errorLogger.Write([]byte("error test"))
				h.AssertEq(t, errOutput(), "error test")
			})
		})

		when("doesn't implement isSelectableWriter", func() {
			it("returns one Writer for all levels", func() {
				var w bytes.Buffer
				logger := logging.NewSimpleLogger(&w)
				writer := logging.GetWriterForLevel(logger, logging.InfoLevel)
				_, _ = writer.Write([]byte("info test\n"))
				h.AssertEq(t, w.String(), "info test\n")

				writer = logging.GetWriterForLevel(logger, logging.ErrorLevel)
				_, _ = writer.Write([]byte("error test\n"))
				h.AssertEq(t, w.String(), "info test\nerror test\n")
			})
		})
	})

	when("IsQuiet", func() {
		when("implements isSelectableWriter", func() {
			it("return true for quiet mode", func() {
				var w bytes.Buffer
				logger := logging.NewLogWithWriters(&w, &w)
				h.AssertEq(t, logging.IsQuiet(logger), false)

				logger.WantQuiet(true)
				h.AssertEq(t, logging.IsQuiet(logger), true)
			})
		})

		when("doesn't implement isSelectableWriter", func() {
			it("always returns false", func() {
				var w bytes.Buffer
				logger := logging.NewSimpleLogger(&w)
				h.AssertEq(t, logging.IsQuiet(logger), false)
			})
		})
	})

	when("#Tip", func() {
		it("prepends `Tip:` to string", func() {
			var w bytes.Buffer
			logger := logging.NewSimpleLogger(&w)
			logging.Tip(logger, "test")
			h.AssertContains(t, w.String(), "Tip: "+"test")
		})
	})
}
