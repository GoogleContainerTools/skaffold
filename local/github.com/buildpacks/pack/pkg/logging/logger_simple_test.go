package logging_test

import (
	"bytes"
	"testing"

	"github.com/sclevine/spec"

	"github.com/buildpacks/pack/pkg/logging"
	h "github.com/buildpacks/pack/testhelpers"
)

const (
	debugMatcher = `^\d{4}\/\d{2}\/\d{2} \d{2}:\d{2}:\d{2}\.\d{6} DEBUG:  \w*\n$`
	infoMatcher  = `^\d{4}\/\d{2}\/\d{2} \d{2}:\d{2}:\d{2}\.\d{6} INFO:   \w*\n$`
	warnMatcher  = `^\d{4}\/\d{2}\/\d{2} \d{2}:\d{2}:\d{2}\.\d{6} WARN:   \w*\n$`
	errorMatcher = `^\d{4}\/\d{2}\/\d{2} \d{2}:\d{2}:\d{2}\.\d{6} ERROR:  \w*\n$`
)

func TestSimpleLogger(t *testing.T) {
	spec.Run(t, "SimpleLogger", func(t *testing.T, when spec.G, it spec.S) {
		var w bytes.Buffer
		var logger logging.Logger

		it.Before(func() {
			logger = logging.NewSimpleLogger(&w)
		})

		it.After(func() {
			w.Reset()
		})

		it("should print debug messages properly", func() {
			logger.Debug("test")
			h.AssertMatch(t, w.String(), debugMatcher)
		})

		it("should format debug messages properly", func() {
			logger.Debugf("test%s", "foo")
			h.AssertMatch(t, w.String(), debugMatcher)
		})

		it("should print info messages properly", func() {
			logger.Info("test")
			h.AssertMatch(t, w.String(), infoMatcher)
		})

		it("should format info messages properly", func() {
			logger.Infof("test%s", "foo")
			h.AssertMatch(t, w.String(), infoMatcher)
		})

		it("should print error messages properly", func() {
			logger.Error("test")
			h.AssertMatch(t, w.String(), errorMatcher)
		})

		it("should format error messages properly", func() {
			logger.Errorf("test%s", "foo")
			h.AssertMatch(t, w.String(), errorMatcher)
		})

		it("should print warn messages properly", func() {
			logger.Warn("test")
			h.AssertMatch(t, w.String(), warnMatcher)
		})

		it("should format warn messages properly", func() {
			logger.Warnf("test%s", "foo")
			h.AssertMatch(t, w.String(), warnMatcher)
		})

		it("shouldn't be verbose by default", func() {
			h.AssertFalse(t, logger.IsVerbose())
		})

		it("should not format writer messages", func() {
			_, _ = logger.Writer().Write([]byte("test"))
			h.AssertEq(t, w.String(), "test")
		})
	})
}
