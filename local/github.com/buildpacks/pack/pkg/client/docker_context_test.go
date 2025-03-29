package client_test

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/heroku/color"
	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"

	"github.com/buildpacks/pack/pkg/client"
	"github.com/buildpacks/pack/pkg/logging"
	h "github.com/buildpacks/pack/testhelpers"
)

func TestProcessDockerContext(t *testing.T) {
	color.Disable(true)
	defer color.Disable(false)
	spec.Run(t, "processDockerContext", testProcessDockerContext, spec.Report(report.Terminal{}))
}

const (
	rootFolder = "docker-context"
	happyCase  = "happy-cases"
	errorCase  = "error-cases"
)

func testProcessDockerContext(t *testing.T, when spec.G, it spec.S) {
	var (
		outBuf bytes.Buffer
		logger logging.Logger
	)

	it.Before(func() {
		logger = logging.NewLogWithWriters(&outBuf, &outBuf, logging.WithVerbose())
	})

	when("env DOCKER_HOST is set", func() {
		it.Before(func() {
			os.Setenv("DOCKER_HOST", "some-value")
		})

		it("docker context process is skipped", func() {
			err := client.ProcessDockerContext(logger)
			h.AssertNil(t, err)
			h.AssertContains(t, strings.TrimSpace(outBuf.String()), "'DOCKER_HOST=some-value' environment variable is being used")
		})
	})

	when("env DOCKER_HOST is empty", func() {
		it.Before(func() {
			os.Setenv("DOCKER_HOST", "")
		})

		when("config.json has currentContext", func() {
			when("currentContext is default", func() {
				it.Before(func() {
					setDockerConfig(t, happyCase, "default-context")
				})

				it("docker context process is skip", func() {
					err := client.ProcessDockerContext(logger)
					h.AssertNil(t, err)
					h.AssertContains(t, strings.TrimSpace(outBuf.String()), "docker context is default or empty, skipping it")
				})
			})

			when("currentContext is default but config doesn't exist", func() {
				it.Before(func() {
					setDockerConfig(t, errorCase, "empty-context")
				})

				it("throw an error", func() {
					err := client.ProcessDockerContext(logger)
					h.AssertNotNil(t, err)
					h.AssertError(t, err, "docker context 'some-bad-context' not found")
				})
			})

			when("currentContext is not default", func() {
				when("metadata has one endpoint", func() {
					it.Before(func() {
						setDockerConfig(t, happyCase, "custom-context")
					})

					it("docker endpoint host is being used", func() {
						err := client.ProcessDockerContext(logger)
						h.AssertNil(t, err)
						h.AssertContains(t, outBuf.String(), "using docker context 'desktop-linux' with endpoint = 'unix:///Users/user/.docker/run/docker.sock'")
					})
				})

				when("metadata has more than one endpoint", func() {
					it.Before(func() {
						setDockerConfig(t, happyCase, "two-endpoints-context")
					})

					it("docker endpoint host is being used", func() {
						err := client.ProcessDockerContext(logger)
						h.AssertNil(t, err)
						h.AssertContains(t, outBuf.String(), "using docker context 'desktop-linux' with endpoint = 'unix:///Users/user/.docker/run/docker.sock'")
					})
				})

				when("currentContext doesn't match metadata name", func() {
					it.Before(func() {
						setDockerConfig(t, errorCase, "current-context-does-not-match")
					})

					it("throw an error", func() {
						err := client.ProcessDockerContext(logger)
						h.AssertNotNil(t, err)
						h.AssertError(t, err, "context 'desktop-linux' doesn't match metadata name 'bad-name'")
					})
				})

				when("metadata doesn't contain a docker endpoint", func() {
					it.Before(func() {
						setDockerConfig(t, errorCase, "docker-endpoint-does-not-exist")
					})

					it("writes a warn message into the log", func() {
						err := client.ProcessDockerContext(logger)
						h.AssertNil(t, err)
						h.AssertContains(t, outBuf.String(), "docker endpoint doesn't exist for context 'desktop-linux'")
					})
				})

				when("metadata is invalid", func() {
					it.Before(func() {
						setDockerConfig(t, errorCase, "invalid-metadata")
					})

					it("throw an error", func() {
						err := client.ProcessDockerContext(logger)
						h.AssertNotNil(t, err)
						h.AssertError(t, err, "reading metadata for current context 'desktop-linux'")
					})
				})
			})
		})

		when("config.json is invalid", func() {
			it.Before(func() {
				setDockerConfig(t, errorCase, "invalid-config")
			})

			it("throw an error", func() {
				err := client.ProcessDockerContext(logger)
				h.AssertNotNil(t, err)
				h.AssertError(t, err, "reading configuration file")
			})
		})

		when("config.json doesn't have current context", func() {
			it.Before(func() {
				setDockerConfig(t, happyCase, "current-context-not-defined")
			})

			it("docker context process is skip", func() {
				err := client.ProcessDockerContext(logger)
				h.AssertNil(t, err)
				h.AssertContains(t, strings.TrimSpace(outBuf.String()), "docker context is default or empty, skipping it")
			})
		})

		when("docker config folder doesn't exists", func() {
			it.Before(func() {
				setDockerConfig(t, errorCase, "no-docker-folder")
			})

			it("docker context process is skip", func() {
				err := client.ProcessDockerContext(logger)
				h.AssertNil(t, err)
				h.AssertContains(t, strings.TrimSpace(outBuf.String()), "docker context is default or empty, skipping it")
			})
		})

		when("config.json config doesn't exists", func() {
			it.Before(func() {
				setDockerConfig(t, errorCase, "config-does-not-exist")
			})

			it("docker context process is skip", func() {
				err := client.ProcessDockerContext(logger)
				h.AssertNil(t, err)
				h.AssertContains(t, strings.TrimSpace(outBuf.String()), "docker context is default or empty, skipping it")
			})
		})
	})
}

func setDockerConfig(t *testing.T, test, context string) {
	t.Helper()
	contextDir, err := filepath.Abs(filepath.Join("testdata", rootFolder, test, context))
	h.AssertNil(t, err)
	err = os.Setenv("DOCKER_CONFIG", contextDir)
	h.AssertNil(t, err)
}
