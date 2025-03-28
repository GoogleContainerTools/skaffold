package name_test

import (
	"io"
	"testing"

	"github.com/buildpacks/pack/pkg/dist"

	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"

	"github.com/buildpacks/pack/internal/name"
	"github.com/buildpacks/pack/pkg/logging"
	h "github.com/buildpacks/pack/testhelpers"
)

func TestTranslateRegistry(t *testing.T) {
	spec.Run(t, "TranslateRegistry", testTranslateRegistry, spec.Report(report.Terminal{}))
}

func testTranslateRegistry(t *testing.T, when spec.G, it spec.S) {
	var (
		assert = h.NewAssertionManager(t)
		logger = logging.NewSimpleLogger(io.Discard)
	)

	when("#TranslateRegistry", func() {
		it("doesn't translate when there are no mirrors", func() {
			input := "index.docker.io/my/buildpack:0.1"

			output, err := name.TranslateRegistry(input, nil, logger)
			assert.Nil(err)
			assert.Equal(output, input)
		})

		it("doesn't translate when there are is no matching mirrors", func() {
			input := "index.docker.io/my/buildpack:0.1"
			registryMirrors := map[string]string{
				"us.gcr.io": "10.0.0.1",
			}

			output, err := name.TranslateRegistry(input, registryMirrors, logger)
			assert.Nil(err)
			assert.Equal(output, input)
		})

		it("translates when there is a mirror", func() {
			input := "index.docker.io/my/buildpack:0.1"
			expected := "10.0.0.1/my/buildpack:0.1"
			registryMirrors := map[string]string{
				"index.docker.io": "10.0.0.1",
			}

			output, err := name.TranslateRegistry(input, registryMirrors, logger)
			assert.Nil(err)
			assert.Equal(output, expected)
		})

		it("prefers the wildcard mirror translation", func() {
			input := "index.docker.io/my/buildpack:0.1"
			expected := "10.0.0.2/my/buildpack:0.1"
			registryMirrors := map[string]string{
				"index.docker.io": "10.0.0.1",
				"*":               "10.0.0.2",
			}

			output, err := name.TranslateRegistry(input, registryMirrors, logger)
			assert.Nil(err)
			assert.Equal(output, expected)
		})

		it("translate a buildpack referenced by a digest", func() {
			input := "buildpack/bp@sha256:7f48a442c056cd19ea48462e05faa2837ac3a13732c47616d20f11f8c847a8c4"
			expected := "myregistry.com/buildpack/bp@sha256:7f48a442c056cd19ea48462e05faa2837ac3a13732c47616d20f11f8c847a8c4"
			registryMirrors := map[string]string{
				"index.docker.io": "myregistry.com",
			}

			output, err := name.TranslateRegistry(input, registryMirrors, logger)
			assert.Nil(err)
			assert.Equal(output, expected)
		})
	})

	when("#AppendSuffix", func() {
		when("[os] is provided", func() {
			when("[arch]] is provided", func() {
				when("[arch-variant] is provided", func() {
					when("tag is provided", func() {
						it("append [os]-[arch]-[arch-variant] to the given tag", func() {
							input := "my.registry.com/my-repo/my-image:some-tag"
							target := dist.Target{
								OS:          "linux",
								Arch:        "amd64",
								ArchVariant: "v6",
							}

							result, err := name.AppendSuffix(input, target)
							assert.Nil(err)
							assert.Equal(result, "my.registry.com/my-repo/my-image:some-tag-linux-amd64-v6")
						})
					})
					when("tag is not provided", func() {
						it("add tag: [os]-[arch]-[arch-variant] to the given <image>", func() {
							input := "my.registry.com/my-repo/my-image"
							target := dist.Target{
								OS:          "linux",
								Arch:        "amd64",
								ArchVariant: "v6",
							}

							result, err := name.AppendSuffix(input, target)
							assert.Nil(err)
							assert.Equal(result, "my.registry.com/my-repo/my-image:linux-amd64-v6")
						})
					})
				})
				when("[arch-variant] is not provided", func() {
					when("tag is provided", func() {
						// my.registry.com/my-repo/my-image:some-tag
						it("append [os]-[arch] to the given tag", func() {
							input := "my.registry.com/my-repo/my-image:some-tag"
							target := dist.Target{
								OS:   "linux",
								Arch: "amd64",
							}

							result, err := name.AppendSuffix(input, target)
							assert.Nil(err)
							assert.Equal(result, "my.registry.com/my-repo/my-image:some-tag-linux-amd64")
						})
					})
					when("tag is NOT provided", func() {
						// my.registry.com/my-repo/my-image
						it("add tag: [os]-[arch] to the given <image>", func() {
							input := "my.registry.com/my-repo/my-image"
							target := dist.Target{
								OS:   "linux",
								Arch: "amd64",
							}

							result, err := name.AppendSuffix(input, target)
							assert.Nil(err)
							assert.Equal(result, "my.registry.com/my-repo/my-image:linux-amd64")
						})
					})
				})
			})

			when("[arch] is not provided", func() {
				when("tag is provided", func() {
					// my.registry.com/my-repo/my-image:some-tag
					it("append [os] to the given tag", func() {
						input := "my.registry.com/my-repo/my-image:some-tag"
						target := dist.Target{
							OS: "linux",
						}

						result, err := name.AppendSuffix(input, target)
						assert.Nil(err)
						assert.Equal(result, "my.registry.com/my-repo/my-image:some-tag-linux")
					})
				})
				when("tag is not provided", func() {
					// my.registry.com/my-repo/my-image
					it("add tag: [os] to the given <image>", func() {
						input := "my.registry.com/my-repo/my-image"
						target := dist.Target{
							OS: "linux",
						}

						result, err := name.AppendSuffix(input, target)
						assert.Nil(err)
						assert.Equal(result, "my.registry.com/my-repo/my-image:linux")
					})
				})
			})
		})

		when("[os] is not provided", func() {
			it("doesn't append anything and return the same <image> name", func() {
				input := "my.registry.com/my-repo/my-image"
				target := dist.Target{}

				result, err := name.AppendSuffix(input, target)
				assert.Nil(err)
				assert.Equal(result, input)
			})
		})
	})
}
