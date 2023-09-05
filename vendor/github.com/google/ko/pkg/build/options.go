// Copyright 2018 ko Build Authors All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package build

import (
	"strings"

	v1 "github.com/google/go-containerregistry/pkg/v1"
)

// WithBaseImages is a functional option for overriding the base images
// that are used for different images.
func WithBaseImages(gb GetBase) Option {
	return func(gbo *gobuildOpener) error {
		gbo.getBase = gb
		return nil
	}
}

// WithCreationTime is a functional option for overriding the creation
// time given to images.
func WithCreationTime(t v1.Time) Option {
	return func(gbo *gobuildOpener) error {
		gbo.creationTime = t
		return nil
	}
}

// WithKoDataCreationTime is a functional option for overriding the creation
// time given to the files in the kodata directory.
func WithKoDataCreationTime(t v1.Time) Option {
	return func(gbo *gobuildOpener) error {
		gbo.kodataCreationTime = t
		return nil
	}
}

// WithDisabledOptimizations is a functional option for disabling optimizations
// when compiling.
func WithDisabledOptimizations() Option {
	return func(gbo *gobuildOpener) error {
		gbo.disableOptimizations = true
		return nil
	}
}

// WithDisabledSBOM is a functional option for disabling SBOM generation.
func WithDisabledSBOM() Option {
	return func(gbo *gobuildOpener) error {
		gbo.sbom = nil
		return nil
	}
}

// WithTrimpath is a functional option that controls whether the `-trimpath`
// flag is added to `go build`.
func WithTrimpath(v bool) Option {
	return func(gbo *gobuildOpener) error {
		gbo.trimpath = v
		return nil
	}
}

// WithConfig is a functional option for providing GoReleaser Build influenced
// build settings for importpaths.
//
// Set a fully qualified importpath (e.g. github.com/my-user/my-repo/cmd/app)
// as the mapping key for the respective Config.
func WithConfig(buildConfigs map[string]Config) Option {
	return func(gbo *gobuildOpener) error {
		gbo.buildConfigs = buildConfigs
		return nil
	}
}

// WithPlatforms is a functional option for building certain platforms for
// multi-platform base images. To build everything from the base, use "all",
// otherwise use a list of platform specs, i.e.:
//
// platform = <os>[/<arch>[/<variant>]]
// allowed = "all" | []string{platform[,platform]*}
//
// Note: a string of comma-separated platforms (i.e. "platform[,platform]*")
// has been deprecated and only exist for backwards compatibility reasons,
// which will be removed in the future.
func WithPlatforms(platforms ...string) Option {
	return func(gbo *gobuildOpener) error {
		if len(platforms) == 1 {
			// TODO: inform users that they are using deprecated flow?
			platforms = strings.Split(platforms[0], ",")
		}
		gbo.platforms = platforms
		return nil
	}
}

// WithLabel is a functional option for adding labels to built images.
func WithLabel(k, v string) Option {
	return func(gbo *gobuildOpener) error {
		if gbo.labels == nil {
			gbo.labels = map[string]string{}
		}
		gbo.labels[k] = v
		return nil
	}
}

// withBuilder is a functional option for overriding the way go binaries
// are built.
func withBuilder(b builder) Option {
	return func(gbo *gobuildOpener) error {
		gbo.build = b
		return nil
	}
}

// WithGoVersionSBOM is a functional option to direct ko to use
// go version -m for SBOM format.
func WithGoVersionSBOM() Option {
	return func(gbo *gobuildOpener) error {
		gbo.sbom = goversionm
		return nil
	}
}

// WithSPDX is a functional option to direct ko to use
// SPDX for SBOM format.
func WithSPDX(version string) Option {
	return func(gbo *gobuildOpener) error {
		gbo.sbom = spdx(version)
		return nil
	}
}

// WithCycloneDX is a functional option to direct ko to use CycloneDX for SBOM
// format.
func WithCycloneDX() Option {
	return func(gbo *gobuildOpener) error {
		gbo.sbom = cycloneDX()
		return nil
	}
}

// withSBOMber is a functional option for overriding the way SBOMs
// are generated.
func withSBOMber(sbom sbomber) Option {
	return func(gbo *gobuildOpener) error {
		gbo.sbom = sbom
		return nil
	}
}

// WithJobs limits the number of concurrent builds.
func WithJobs(jobs int) Option {
	return func(gbo *gobuildOpener) error {
		gbo.jobs = jobs
		return nil
	}
}

// WithSBOMDir is a functional option for overriding the directory
func WithSBOMDir(dir string) Option {
	return func(gbo *gobuildOpener) error {
		gbo.sbomDir = dir
		return nil
	}
}
