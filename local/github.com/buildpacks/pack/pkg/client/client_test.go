package client

import (
	"bytes"
	"os"
	"testing"

	dockerClient "github.com/docker/docker/client"
	"github.com/golang/mock/gomock"
	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"

	"github.com/buildpacks/pack/pkg/logging"
	"github.com/buildpacks/pack/pkg/testmocks"
	h "github.com/buildpacks/pack/testhelpers"
)

func TestClient(t *testing.T) {
	spec.Run(t, "Client", testClient, spec.Report(report.Terminal{}))
}

func testClient(t *testing.T, when spec.G, it spec.S) {
	when("#NewClient", func() {
		it("default works", func() {
			_, err := NewClient()
			h.AssertNil(t, err)
		})

		when("docker env is messed up", func() {
			var dockerHost string
			var dockerHostKey = "DOCKER_HOST"
			it.Before(func() {
				dockerHost = os.Getenv(dockerHostKey)
				h.AssertNil(t, os.Setenv(dockerHostKey, "fake-value"))
			})

			it.After(func() {
				h.AssertNil(t, os.Setenv(dockerHostKey, dockerHost))
			})

			it("returns errors", func() {
				_, err := NewClient()
				h.AssertError(t, err, "docker client")
			})
		})
	})

	when("#WithLogger", func() {
		it("uses logger provided", func() {
			var w bytes.Buffer
			logger := logging.NewSimpleLogger(&w)
			cl, err := NewClient(WithLogger(logger))
			h.AssertNil(t, err)
			h.AssertSameInstance(t, cl.logger, logger)
		})
	})

	when("#WithImageFactory", func() {
		it("uses image factory provided", func() {
			mockController := gomock.NewController(t)
			mockImageFactory := testmocks.NewMockImageFactory(mockController)
			cl, err := NewClient(WithImageFactory(mockImageFactory))
			h.AssertNil(t, err)
			h.AssertSameInstance(t, cl.imageFactory, mockImageFactory)
		})
	})

	when("#WithFetcher", func() {
		it("uses image factory provided", func() {
			mockController := gomock.NewController(t)
			mockFetcher := testmocks.NewMockImageFetcher(mockController)
			cl, err := NewClient(WithFetcher(mockFetcher))
			h.AssertNil(t, err)
			h.AssertSameInstance(t, cl.imageFetcher, mockFetcher)
		})
	})

	when("#WithDownloader", func() {
		it("uses image factory provided", func() {
			mockController := gomock.NewController(t)
			mockDownloader := testmocks.NewMockBlobDownloader(mockController)
			cl, err := NewClient(WithDownloader(mockDownloader))
			h.AssertNil(t, err)
			h.AssertSameInstance(t, cl.downloader, mockDownloader)
		})
	})

	when("#WithDockerClient", func() {
		it("uses docker client provided", func() {
			docker, err := dockerClient.NewClientWithOpts(
				dockerClient.FromEnv,
			)
			h.AssertNil(t, err)
			cl, err := NewClient(WithDockerClient(docker))
			h.AssertNil(t, err)
			h.AssertSameInstance(t, cl.docker, docker)
		})
	})

	when("#WithExperimental", func() {
		it("sets experimental = true", func() {
			cl, err := NewClient(WithExperimental(true))
			h.AssertNil(t, err)
			h.AssertEq(t, cl.experimental, true)
		})

		it("sets experimental = false", func() {
			cl, err := NewClient(WithExperimental(true))
			h.AssertNil(t, err)
			h.AssertEq(t, cl.experimental, true)
		})
	})

	when("#WithRegistryMirror", func() {
		it("uses registry mirrors provided", func() {
			registryMirrors := map[string]string{
				"index.docker.io": "10.0.0.1",
			}

			cl, err := NewClient(WithRegistryMirrors(registryMirrors))
			h.AssertNil(t, err)
			h.AssertEq(t, cl.registryMirrors, registryMirrors)
		})
	})
}
