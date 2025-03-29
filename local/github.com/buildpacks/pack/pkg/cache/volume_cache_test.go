package cache_test

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/buildpacks/pack/internal/config"
	"github.com/buildpacks/pack/pkg/cache"
	"github.com/buildpacks/pack/pkg/logging"

	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/volume"
	"github.com/docker/docker/client"
	"github.com/docker/docker/daemon/names"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/heroku/color"
	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"

	h "github.com/buildpacks/pack/testhelpers"
)

func TestVolumeCache(t *testing.T) {
	h.RequireDocker(t)
	color.Disable(true)
	defer color.Disable(false)

	spec.Run(t, "VolumeCache", testCache, spec.Sequential(), spec.Report(report.Terminal{}))
}

func testCache(t *testing.T, when spec.G, it spec.S) {
	var (
		dockerClient client.CommonAPIClient
		outBuf       bytes.Buffer
		logger       logging.Logger
	)

	it.Before(func() {
		var err error
		dockerClient, err = client.NewClientWithOpts(client.FromEnv, client.WithVersion("1.38"))
		h.AssertNil(t, err)
		logger = logging.NewSimpleLogger(&outBuf)
	})

	when("#NewVolumeCache", func() {
		when("volume cache name is empty", func() {
			it("adds suffix to calculated name", func() {
				ref, err := name.ParseReference("my/repo", name.WeakValidation)
				h.AssertNil(t, err)
				subject, _ := cache.NewVolumeCache(ref, cache.CacheInfo{}, "some-suffix", dockerClient, logger)
				if !strings.HasSuffix(subject.Name(), ".some-suffix") {
					t.Fatalf("Calculated volume name '%s' should end with '.some-suffix'", subject.Name())
				}
			})

			it("reusing the same cache for the same repo name", func() {
				ref, err := name.ParseReference("my/repo", name.WeakValidation)
				h.AssertNil(t, err)

				subject, _ := cache.NewVolumeCache(ref, cache.CacheInfo{}, "some-suffix", dockerClient, logger)
				expected, _ := cache.NewVolumeCache(ref, cache.CacheInfo{}, "some-suffix", dockerClient, logger)
				if subject.Name() != expected.Name() {
					t.Fatalf("The same repo name should result in the same volume")
				}
			})

			it("supplies different volumes for different tags", func() {
				ref, err := name.ParseReference("my/repo:other-tag", name.WeakValidation)
				h.AssertNil(t, err)

				subject, _ := cache.NewVolumeCache(ref, cache.CacheInfo{}, "some-suffix", dockerClient, logger)

				ref, err = name.ParseReference("my/repo", name.WeakValidation)
				h.AssertNil(t, err)
				notExpected, _ := cache.NewVolumeCache(ref, cache.CacheInfo{}, "some-suffix", dockerClient, logger)
				if subject.Name() == notExpected.Name() {
					t.Fatalf("Different image tags should result in different volumes")
				}
			})

			it("supplies different volumes for different registries", func() {
				ref, err := name.ParseReference("registry.com/my/repo:other-tag", name.WeakValidation)
				h.AssertNil(t, err)

				subject, _ := cache.NewVolumeCache(ref, cache.CacheInfo{}, "some-suffix", dockerClient, logger)

				ref, err = name.ParseReference("my/repo", name.WeakValidation)
				h.AssertNil(t, err)
				notExpected, _ := cache.NewVolumeCache(ref, cache.CacheInfo{}, "some-suffix", dockerClient, logger)
				if subject.Name() == notExpected.Name() {
					t.Fatalf("Different image registries should result in different volumes")
				}
			})

			it("resolves implied tag", func() {
				ref, err := name.ParseReference("my/repo:latest", name.WeakValidation)
				h.AssertNil(t, err)

				subject, _ := cache.NewVolumeCache(ref, cache.CacheInfo{}, "some-suffix", dockerClient, logger)

				ref, err = name.ParseReference("my/repo", name.WeakValidation)
				h.AssertNil(t, err)
				expected, _ := cache.NewVolumeCache(ref, cache.CacheInfo{}, "some-suffix", dockerClient, logger)
				h.AssertEq(t, subject.Name(), expected.Name())
			})

			it("resolves implied registry", func() {
				ref, err := name.ParseReference("index.docker.io/my/repo", name.WeakValidation)
				h.AssertNil(t, err)

				subject, _ := cache.NewVolumeCache(ref, cache.CacheInfo{}, "some-suffix", dockerClient, logger)

				ref, err = name.ParseReference("my/repo", name.WeakValidation)
				h.AssertNil(t, err)
				expected, _ := cache.NewVolumeCache(ref, cache.CacheInfo{}, "some-suffix", dockerClient, logger)
				h.AssertEq(t, subject.Name(), expected.Name())
			})

			it("includes human readable information", func() {
				ref, err := name.ParseReference("myregistryhost:5000/fedora/httpd:version1.0", name.WeakValidation)
				h.AssertNil(t, err)

				subject, _ := cache.NewVolumeCache(ref, cache.CacheInfo{}, "some-suffix", dockerClient, logger)

				h.AssertContains(t, subject.Name(), "fedora_httpd_version1.0")
				h.AssertTrue(t, names.RestrictedNamePattern.MatchString(subject.Name()))
			})

			when("PACK_VOLUME_KEY", func() {
				when("is set", func() {
					it.After(func() {
						h.AssertNil(t, os.Unsetenv("PACK_VOLUME_KEY"))
					})

					it("uses it to construct the volume name", func() {
						ref, err := name.ParseReference("my/repo:some-tag", name.WeakValidation)
						h.AssertNil(t, err)

						nameFromNewKey, _ := cache.NewVolumeCache(ref, cache.CacheInfo{}, "some-suffix", dockerClient, logger) // sources a new key
						h.AssertNil(t, os.Setenv("PACK_VOLUME_KEY", "some-volume-key"))
						nameFromEnvKey, _ := cache.NewVolumeCache(ref, cache.CacheInfo{}, "some-suffix", dockerClient, logger) // sources key from env
						h.AssertNotEq(t, nameFromNewKey.Name(), nameFromEnvKey.Name())
					})
				})

				when("is unset", func() {
					var tmpPackHome string

					it.Before(func() {
						var err error
						tmpPackHome, err = os.MkdirTemp("", "")
						h.AssertNil(t, err)
						h.AssertNil(t, os.Setenv("PACK_HOME", tmpPackHome))
					})

					it.After(func() {
						h.AssertNil(t, os.RemoveAll(tmpPackHome))
					})

					when("~/.pack/volume-keys.toml contains key for repo name", func() {
						it("sources the key from ~/.pack/volume-keys.toml", func() {
							ref, err := name.ParseReference("my/repo:some-tag", name.WeakValidation)
							h.AssertNil(t, err)

							nameFromNewKey, _ := cache.NewVolumeCache(ref, cache.CacheInfo{}, "some-suffix", dockerClient, logger) // sources a new key

							cfgContents := `
[volume-keys]
"index.docker.io/my/repo:some-tag" = "SOME_VOLUME_KEY"
`
							h.AssertNil(t, os.WriteFile(filepath.Join(tmpPackHome, "volume-keys.toml"), []byte(cfgContents), 0755)) // overrides the key that was set

							nameFromConfigKey, _ := cache.NewVolumeCache(ref, cache.CacheInfo{}, "some-suffix", dockerClient, logger) // sources key from config
							h.AssertNotEq(t, nameFromNewKey.Name(), nameFromConfigKey.Name())
						})
					})

					when("~/.pack/volume-keys.toml missing key for repo name", func() {
						it("generates a new key and saves it to ~/.pack/volume-keys.toml", func() {
							ref, err := name.ParseReference("my/repo:some-tag", name.WeakValidation)
							h.AssertNil(t, err)

							nameFromNewKey, _ := cache.NewVolumeCache(ref, cache.CacheInfo{}, "some-suffix", dockerClient, logger)    // sources a new key
							nameFromConfigKey, _ := cache.NewVolumeCache(ref, cache.CacheInfo{}, "some-suffix", dockerClient, logger) // sources same key from config
							h.AssertEq(t, nameFromNewKey.Name(), nameFromConfigKey.Name())

							cfg, err := config.ReadVolumeKeys(filepath.Join(tmpPackHome, "volume-keys.toml"))
							h.AssertNil(t, err)
							h.AssertNotNil(t, cfg.VolumeKeys["index.docker.io/my/repo:some-tag"])
						})

						when("containerized pack", func() {
							it.Before(func() {
								cache.RunningInContainer = func() bool {
									return true
								}
							})

							it("logs a warning", func() {
								ref, err := name.ParseReference("my/repo:some-tag", name.WeakValidation)
								h.AssertNil(t, err)

								_, _ = cache.NewVolumeCache(ref, cache.CacheInfo{}, "some-suffix", dockerClient, logger) // sources a new key
								_, _ = cache.NewVolumeCache(ref, cache.CacheInfo{}, "some-suffix", dockerClient, logger) // sources same key from config
								h.AssertContains(t, outBuf.String(), "PACK_VOLUME_KEY is unset; set this environment variable to a secret value to avoid creating a new volume cache on every build")
								h.AssertEq(t, strings.Count(outBuf.String(), "PACK_VOLUME_KEY is unset"), 1) // the second call to NewVolumeCache reads from the config
							})
						})
					})
				})
			})
		})

		when("volume cache name is not empty", func() {
			volumeName := "test-volume-name"
			cacheInfo := cache.CacheInfo{
				Format: cache.CacheVolume,
				Source: volumeName,
			}

			it("named volume created without suffix", func() {
				ref, err := name.ParseReference("my/repo", name.WeakValidation)
				h.AssertNil(t, err)

				subject, _ := cache.NewVolumeCache(ref, cacheInfo, "some-suffix", dockerClient, logger)

				if volumeName != subject.Name() {
					t.Fatalf("Volume name '%s' should be same as the name specified '%s'", subject.Name(), volumeName)
				}
			})

			it("reusing the same cache for the same repo name", func() {
				ref, err := name.ParseReference("my/repo", name.WeakValidation)
				h.AssertNil(t, err)

				subject, _ := cache.NewVolumeCache(ref, cacheInfo, "some-suffix", dockerClient, logger)

				expected, _ := cache.NewVolumeCache(ref, cacheInfo, "some-suffix", dockerClient, logger)
				if subject.Name() != expected.Name() {
					t.Fatalf("The same repo name should result in the same volume")
				}
			})

			it("supplies different volumes for different registries", func() {
				ref, err := name.ParseReference("registry.com/my/repo:other-tag", name.WeakValidation)
				h.AssertNil(t, err)

				subject, _ := cache.NewVolumeCache(ref, cache.CacheInfo{}, "some-suffix", dockerClient, logger)

				ref, err = name.ParseReference("my/repo", name.WeakValidation)
				h.AssertNil(t, err)
				notExpected, _ := cache.NewVolumeCache(ref, cache.CacheInfo{}, "some-suffix", dockerClient, logger)
				if subject.Name() == notExpected.Name() {
					t.Fatalf("Different image registries should result in different volumes")
				}
			})

			it("resolves implied tag", func() {
				ref, err := name.ParseReference("my/repo:latest", name.WeakValidation)
				h.AssertNil(t, err)

				subject, _ := cache.NewVolumeCache(ref, cache.CacheInfo{}, "some-suffix", dockerClient, logger)

				ref, err = name.ParseReference("my/repo", name.WeakValidation)
				h.AssertNil(t, err)
				expected, _ := cache.NewVolumeCache(ref, cache.CacheInfo{}, "some-suffix", dockerClient, logger)
				h.AssertEq(t, subject.Name(), expected.Name())
			})

			it("resolves implied registry", func() {
				ref, err := name.ParseReference("index.docker.io/my/repo", name.WeakValidation)
				h.AssertNil(t, err)

				subject, _ := cache.NewVolumeCache(ref, cache.CacheInfo{}, "some-suffix", dockerClient, logger)

				ref, err = name.ParseReference("my/repo", name.WeakValidation)
				h.AssertNil(t, err)
				expected, _ := cache.NewVolumeCache(ref, cache.CacheInfo{}, "some-suffix", dockerClient, logger)
				h.AssertEq(t, subject.Name(), expected.Name())
			})

			it("includes human readable information", func() {
				ref, err := name.ParseReference("myregistryhost:5000/fedora/httpd:version1.0", name.WeakValidation)
				h.AssertNil(t, err)

				subject, _ := cache.NewVolumeCache(ref, cache.CacheInfo{}, "some-suffix", dockerClient, logger)

				h.AssertContains(t, subject.Name(), "fedora_httpd_version1.0")
				h.AssertTrue(t, names.RestrictedNamePattern.MatchString(subject.Name()))
			})
		})
	})

	when("#Clear", func() {
		var (
			volumeName   string
			dockerClient client.CommonAPIClient
			subject      *cache.VolumeCache
			ctx          context.Context
		)

		it.Before(func() {
			var err error
			dockerClient, err = client.NewClientWithOpts(client.FromEnv, client.WithVersion("1.38"))
			h.AssertNil(t, err)
			ctx = context.TODO()

			ref, err := name.ParseReference(h.RandString(10), name.WeakValidation)
			h.AssertNil(t, err)

			subject, _ = cache.NewVolumeCache(ref, cache.CacheInfo{}, "some-suffix", dockerClient, logger)
			volumeName = subject.Name()
		})

		when("there is a cache volume", func() {
			it.Before(func() {
				dockerClient.VolumeCreate(context.TODO(), volume.CreateOptions{
					Name: volumeName,
				})
			})

			it("removes the volume", func() {
				err := subject.Clear(ctx)
				h.AssertNil(t, err)

				volumes, err := dockerClient.VolumeList(context.TODO(), volume.ListOptions{
					Filters: filters.NewArgs(filters.KeyValuePair{
						Key:   "name",
						Value: volumeName,
					}),
				})
				h.AssertNil(t, err)
				h.AssertEq(t, len(volumes.Volumes), 0)
			})
		})

		when("there is no cache volume", func() {
			it("does not fail", func() {
				err := subject.Clear(ctx)
				h.AssertNil(t, err)
			})
		})
	})

	when("#Type", func() {
		it("returns the cache type", func() {
			ref, err := name.ParseReference("my/repo", name.WeakValidation)
			h.AssertNil(t, err)
			subject, _ := cache.NewVolumeCache(ref, cache.CacheInfo{}, "some-suffix", dockerClient, logger)
			expected := cache.Volume
			h.AssertEq(t, subject.Type(), expected)
		})
	})
}
