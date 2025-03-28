package dist_test

import (
	"testing"

	"github.com/buildpacks/lifecycle/api"
	"github.com/heroku/color"
	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"

	"github.com/buildpacks/pack/pkg/buildpack"
	"github.com/buildpacks/pack/pkg/dist"
	h "github.com/buildpacks/pack/testhelpers"
)

func TestBuildpackDescriptor(t *testing.T) {
	color.Disable(true)
	defer color.Disable(false)
	spec.Run(t, "testBuildpackDescriptor", testBuildpackDescriptor, spec.Parallel(), spec.Report(report.Terminal{}))
}

func testBuildpackDescriptor(t *testing.T, when spec.G, it spec.S) {
	when("#EscapedID", func() {
		it("returns escaped ID", func() {
			bpDesc := dist.BuildpackDescriptor{
				WithInfo: dist.ModuleInfo{ID: "some/id"},
			}
			h.AssertEq(t, bpDesc.EscapedID(), "some_id")
		})
	})

	when("#EnsureStackSupport", func() {
		when("not validating against run image mixins", func() {
			it("ignores run-only mixins", func() {
				bp := dist.BuildpackDescriptor{
					WithInfo: dist.ModuleInfo{
						ID:      "some.buildpack.id",
						Version: "some.buildpack.version",
					},
					WithStacks: []dist.Stack{{
						ID:     "some.stack.id",
						Mixins: []string{"mixinA", "build:mixinB", "run:mixinD"},
					}},
				}

				providedMixins := []string{"mixinA", "build:mixinB", "mixinC"}
				h.AssertNil(t, bp.EnsureStackSupport("some.stack.id", providedMixins, false))
			})

			it("works with wildcard stack", func() {
				bp := dist.BuildpackDescriptor{
					WithInfo: dist.ModuleInfo{
						ID:      "some.buildpack.id",
						Version: "some.buildpack.version",
					},
					WithStacks: []dist.Stack{{
						ID:     "*",
						Mixins: []string{"mixinA", "build:mixinB", "run:mixinD"},
					}},
				}

				providedMixins := []string{"mixinA", "build:mixinB", "mixinC"}
				h.AssertNil(t, bp.EnsureStackSupport("some.stack.id", providedMixins, false))
			})

			it("returns an error with any missing (and non-ignored) mixins", func() {
				bp := dist.BuildpackDescriptor{
					WithInfo: dist.ModuleInfo{
						ID:      "some.buildpack.id",
						Version: "some.buildpack.version",
					},
					WithStacks: []dist.Stack{{
						ID:     "some.stack.id",
						Mixins: []string{"mixinX", "mixinY", "run:mixinZ"},
					}},
				}

				providedMixins := []string{"mixinA", "mixinB"}
				err := bp.EnsureStackSupport("some.stack.id", providedMixins, false)

				h.AssertError(t, err, "buildpack 'some.buildpack.id@some.buildpack.version' requires missing mixin(s): mixinX, mixinY")
			})
		})

		when("validating against run image mixins", func() {
			it("requires run-only mixins", func() {
				bp := dist.BuildpackDescriptor{
					WithInfo: dist.ModuleInfo{
						ID:      "some.buildpack.id",
						Version: "some.buildpack.version",
					},
					WithStacks: []dist.Stack{{
						ID:     "some.stack.id",
						Mixins: []string{"mixinA", "build:mixinB", "run:mixinD"},
					}},
				}

				providedMixins := []string{"mixinA", "build:mixinB", "mixinC", "run:mixinD"}

				h.AssertNil(t, bp.EnsureStackSupport("some.stack.id", providedMixins, true))
			})

			it("returns an error with any missing mixins", func() {
				bp := dist.BuildpackDescriptor{
					WithInfo: dist.ModuleInfo{
						ID:      "some.buildpack.id",
						Version: "some.buildpack.version",
					},
					WithStacks: []dist.Stack{{
						ID:     "some.stack.id",
						Mixins: []string{"mixinX", "mixinY", "run:mixinZ"},
					}},
				}

				providedMixins := []string{"mixinA", "mixinB"}

				err := bp.EnsureStackSupport("some.stack.id", providedMixins, true)

				h.AssertError(t, err, "buildpack 'some.buildpack.id@some.buildpack.version' requires missing mixin(s): mixinX, mixinY, run:mixinZ")
			})
		})

		it("returns an error when buildpack does not support stack", func() {
			bp := dist.BuildpackDescriptor{
				WithInfo: dist.ModuleInfo{
					ID:      "some.buildpack.id",
					Version: "some.buildpack.version",
				},
				WithStacks: []dist.Stack{{
					ID:     "some.stack.id",
					Mixins: []string{"mixinX", "mixinY"},
				}},
			}

			err := bp.EnsureStackSupport("some.nonexistent.stack.id", []string{"mixinA"}, true)

			h.AssertError(t, err, "buildpack 'some.buildpack.id@some.buildpack.version' does not support stack 'some.nonexistent.stack.id")
		})

		it("skips validating order buildpack", func() {
			bp := dist.BuildpackDescriptor{
				WithInfo: dist.ModuleInfo{
					ID:      "some.buildpack.id",
					Version: "some.buildpack.version",
				},
				WithStacks: []dist.Stack{},
			}

			h.AssertNil(t, bp.EnsureStackSupport("some.stack.id", []string{"mixinA"}, true))
		})
	})

	when("validating against run image target", func() {
		it("succeeds with no distribution", func() {
			bp := dist.BuildpackDescriptor{
				WithInfo: dist.ModuleInfo{
					ID:      "some.buildpack.id",
					Version: "some.buildpack.version",
				},
				WithTargets: []dist.Target{{
					OS:   "fake-os",
					Arch: "fake-arch",
				}},
			}

			h.AssertNil(t, bp.EnsureStackSupport("some.stack.id", []string{}, true))
			h.AssertNil(t, bp.EnsureTargetSupport("fake-os", "fake-arch", "fake-distro", "0.0"))
		})

		it("succeeds with no target and bin/build.exe", func() {
			bp := dist.BuildpackDescriptor{
				WithInfo: dist.ModuleInfo{
					ID:      "some.buildpack.id",
					Version: "some.buildpack.version",
				},
				WithWindowsBuild: true,
			}

			h.AssertNil(t, bp.EnsureStackSupport("some.stack.id", []string{}, true))
			h.AssertNil(t, bp.EnsureTargetSupport("windows", "amd64", "fake-distro", "0.0"))
		})

		it("succeeds with no target and bin/build", func() {
			bp := dist.BuildpackDescriptor{
				WithInfo: dist.ModuleInfo{
					ID:      "some.buildpack.id",
					Version: "some.buildpack.version",
				},
				WithLinuxBuild: true,
			}

			h.AssertNil(t, bp.EnsureStackSupport("some.stack.id", []string{}, true))
			h.AssertNil(t, bp.EnsureTargetSupport("linux", "amd64", "fake-distro", "0.0"))
		})

		it("returns an error when no match", func() {
			bp := dist.BuildpackDescriptor{
				WithInfo: dist.ModuleInfo{
					ID:      "some.buildpack.id",
					Version: "some.buildpack.version",
				},
				WithTargets: []dist.Target{{
					OS:   "fake-os",
					Arch: "fake-arch",
				}},
			}

			h.AssertNil(t, bp.EnsureStackSupport("some.stack.id", []string{}, true))
			h.AssertError(t, bp.EnsureTargetSupport("some-other-os", "fake-arch", "fake-distro", "0.0"),
				`unable to satisfy target os/arch constraints; build image: {"os":"some-other-os","arch":"fake-arch","distribution":{"name":"fake-distro","version":"0.0"}}, buildpack 'some.buildpack.id@some.buildpack.version': [{"os":"fake-os","arch":"fake-arch"}]`)
		})

		it("succeeds with distribution", func() {
			bp := dist.BuildpackDescriptor{
				WithInfo: dist.ModuleInfo{
					ID:      "some.buildpack.id",
					Version: "some.buildpack.version",
				},
				WithTargets: []dist.Target{{
					OS:   "fake-os",
					Arch: "fake-arch",
					Distributions: []dist.Distribution{
						{
							Name:    "fake-distro",
							Version: "0.1",
						},
						{
							Name:    "another-distro",
							Version: "0.22",
						},
					},
				}},
			}

			h.AssertNil(t, bp.EnsureStackSupport("some.stack.id", []string{}, true))
			h.AssertNil(t, bp.EnsureTargetSupport("fake-os", "fake-arch", "fake-distro", "0.1"))
		})

		it("returns an error when no distribution matches", func() {
			bp := dist.BuildpackDescriptor{
				WithInfo: dist.ModuleInfo{
					ID:      "some.buildpack.id",
					Version: "some.buildpack.version",
				},
				WithTargets: []dist.Target{{
					OS:   "fake-os",
					Arch: "fake-arch",
					Distributions: []dist.Distribution{
						{
							Name:    "fake-distro",
							Version: "0.1",
						},
						{
							Name:    "another-distro",
							Version: "0.22",
						},
					},
				}},
			}

			h.AssertNil(t, bp.EnsureStackSupport("some.stack.id", []string{}, true))
			h.AssertError(t, bp.EnsureTargetSupport("some-other-os", "fake-arch", "fake-distro", "0.0"),
				`unable to satisfy target os/arch constraints; build image: {"os":"some-other-os","arch":"fake-arch","distribution":{"name":"fake-distro","version":"0.0"}}, buildpack 'some.buildpack.id@some.buildpack.version': [{"os":"fake-os","arch":"fake-arch","distros":[{"name":"fake-distro","version":"0.1"},{"name":"another-distro","version":"0.22"}]}]`)
		})

		it("succeeds with missing arch", func() {
			bp := dist.BuildpackDescriptor{
				WithInfo: dist.ModuleInfo{
					ID:      "some.buildpack.id",
					Version: "some.buildpack.version",
				},
				WithTargets: []dist.Target{{
					OS: "fake-os",
				}},
			}

			h.AssertNil(t, bp.EnsureTargetSupport("fake-os", "fake-arch", "fake-distro", "0.1"))
		})
	})

	when("#Kind", func() {
		it("returns 'buildpack'", func() {
			bpDesc := dist.BuildpackDescriptor{}
			h.AssertEq(t, bpDesc.Kind(), buildpack.KindBuildpack)
		})
	})

	when("#API", func() {
		it("returns the api", func() {
			bpDesc := dist.BuildpackDescriptor{
				WithAPI: api.MustParse("0.99"),
			}
			h.AssertEq(t, bpDesc.API().String(), "0.99")
		})
	})

	when("#Info", func() {
		it("returns the module info", func() {
			info := dist.ModuleInfo{
				ID:      "some-id",
				Name:    "some-name",
				Version: "some-version",
			}
			bpDesc := dist.BuildpackDescriptor{
				WithInfo: info,
			}
			h.AssertEq(t, bpDesc.Info(), info)
		})
	})

	when("#Order", func() {
		it("returns the order", func() {
			order := dist.Order{
				dist.OrderEntry{Group: []dist.ModuleRef{
					{ModuleInfo: dist.ModuleInfo{
						ID: "some-id", Name: "some-name", Version: "some-version",
					}},
				}},
			}
			bpDesc := dist.BuildpackDescriptor{
				WithOrder: order,
			}
			h.AssertEq(t, bpDesc.Order(), order)
		})
	})

	when("#Stacks", func() {
		it("returns the stacks", func() {
			stacks := []dist.Stack{
				{ID: "some-id", Mixins: []string{"some-mixin"}},
			}
			bpDesc := dist.BuildpackDescriptor{
				WithStacks: stacks,
			}
			h.AssertEq(t, bpDesc.Stacks(), stacks)
		})
	})
}
