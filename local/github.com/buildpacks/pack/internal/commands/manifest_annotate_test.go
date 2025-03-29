package commands_test

import (
	"bytes"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/heroku/color"
	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"
	"github.com/spf13/cobra"

	"github.com/buildpacks/pack/internal/commands"
	"github.com/buildpacks/pack/internal/commands/testmocks"
	"github.com/buildpacks/pack/pkg/client"
	"github.com/buildpacks/pack/pkg/logging"
	h "github.com/buildpacks/pack/testhelpers"
)

func TestManifestAnnotationsCommand(t *testing.T) {
	color.Disable(true)
	defer color.Disable(false)

	spec.Run(t, "Commands", testManifestAnnotateCommand, spec.Random(), spec.Report(report.Terminal{}))
}

func testManifestAnnotateCommand(t *testing.T, when spec.G, it spec.S) {
	var (
		command        *cobra.Command
		logger         *logging.LogWithWriters
		outBuf         bytes.Buffer
		mockController *gomock.Controller
		mockClient     *testmocks.MockPackClient
	)

	it.Before(func() {
		logger = logging.NewLogWithWriters(&outBuf, &outBuf)
		mockController = gomock.NewController(t)
		mockClient = testmocks.NewMockPackClient(mockController)

		command = commands.ManifestAnnotate(logger, mockClient)
	})

	when("args are valid", func() {
		var (
			indexRepoName string
			repoName      string
		)
		it.Before(func() {
			indexRepoName = h.NewRandomIndexRepoName()
			repoName = "busybox@sha256:6457d53fb065d6f250e1504b9bc42d5b6c65941d57532c072d929dd0628977d0"
		})

		when("index exists", func() {
			when("--os is provided", func() {
				it.Before(func() {
					mockClient.EXPECT().
						AnnotateManifest(
							gomock.Any(),
							gomock.Eq(client.ManifestAnnotateOptions{
								IndexRepoName: indexRepoName,
								RepoName:      repoName,
								OS:            "linux",
								Annotations:   map[string]string{},
							}),
						).
						Return(nil)
				})

				it("should annotate images with given flags", func() {
					command.SetArgs([]string{indexRepoName, repoName, "--os", "linux"})
					h.AssertNilE(t, command.Execute())
				})
			})

			when("--arch is provided", func() {
				it.Before(func() {
					mockClient.EXPECT().
						AnnotateManifest(
							gomock.Any(),
							gomock.Eq(client.ManifestAnnotateOptions{
								IndexRepoName: indexRepoName,
								RepoName:      repoName,
								OSArch:        "amd64",
								Annotations:   map[string]string{},
							}),
						).
						Return(nil)
				})

				it("should annotate images with given flags", func() {
					command.SetArgs([]string{indexRepoName, repoName, "--arch", "amd64"})
					h.AssertNilE(t, command.Execute())
				})
			})

			when("--variant is provided", func() {
				it.Before(func() {
					mockClient.EXPECT().
						AnnotateManifest(
							gomock.Any(),
							gomock.Eq(client.ManifestAnnotateOptions{
								IndexRepoName: indexRepoName,
								RepoName:      repoName,
								OSVariant:     "V6",
								Annotations:   map[string]string{},
							}),
						).
						Return(nil)
				})

				it("should annotate images with given flags", func() {
					command.SetArgs([]string{indexRepoName, repoName, "--variant", "V6"})
					h.AssertNilE(t, command.Execute())
				})
			})

			when("--annotations are provided", func() {
				it.Before(func() {
					mockClient.EXPECT().
						AnnotateManifest(
							gomock.Any(),
							gomock.Eq(client.ManifestAnnotateOptions{
								IndexRepoName: indexRepoName,
								RepoName:      repoName,
								Annotations:   map[string]string{"foo": "bar"},
							}),
						).
						Return(nil)
				})

				it("should annotate images with given flags", func() {
					command.SetArgs([]string{indexRepoName, repoName, "--annotations", "foo=bar"})
					h.AssertNilE(t, command.Execute())
				})
			})

			when("--help", func() {
				it("should have help flag", func() {
					command.SetArgs([]string{"--help"})
					h.AssertNilE(t, command.Execute())
				})
			})
		})
	})

	when("args are invalid", func() {
		it("errors a message when no options are provided", func() {
			command.SetArgs([]string{"foo", "bar"})
			h.AssertError(t, command.Execute(), "one of --os, --arch, or --variant is required")
		})

		it("errors when missing mandatory arguments", func() {
			command.SetArgs([]string{"some-index"})
			err := command.Execute()
			h.AssertNotNil(t, err)
			h.AssertError(t, err, "accepts 2 arg(s), received 1")
		})

		it("errors when annotations are invalid", func() {
			command.SetArgs([]string{"some-index", "some-manifest", "--annotations", "some-key"})
			err := command.Execute()
			h.AssertEq(t, err.Error(), `invalid argument "some-key" for "--annotations" flag: some-key must be formatted as key=value`)
		})
	})
}
