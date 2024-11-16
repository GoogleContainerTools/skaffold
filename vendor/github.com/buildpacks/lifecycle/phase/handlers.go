package phase

import (
	"github.com/buildpacks/lifecycle/buildpack"
	"github.com/buildpacks/lifecycle/log"
	"github.com/buildpacks/lifecycle/platform/files"
)

// CacheHandler wraps initialization of a cache image or cache volume.
//
//go:generate mockgen -package testmock -destination testmock/cache_handler.go github.com/buildpacks/lifecycle/phase CacheHandler
type CacheHandler interface {
	InitCache(imageRef, dir string, deletionEnabled bool) (Cache, error)
}

// DirStore is a repository of buildpacks and/or image extensions.
// Each element should be present on disk according to the format outlined in the Platform Interface Specification,
// namely: `/cnb/<buildpacks|extensions>/<id>/<version>/<root directory>`.
//
//go:generate mockgen -package testmock -destination testmock/dir_store.go github.com/buildpacks/lifecycle/phase DirStore
type DirStore interface {
	Lookup(kind, id, version string) (buildpack.Descriptor, error)
	LookupBp(id, version string) (*buildpack.BpDescriptor, error)
	LookupExt(id, version string) (*buildpack.ExtDescriptor, error)
}

// BuildpackAPIVerifier verifies a requested Buildpack API version.
//
//go:generate mockgen -package testmock -destination testmock/buildpack_api_verifier.go github.com/buildpacks/lifecycle/phase BuildpackAPIVerifier
type BuildpackAPIVerifier interface {
	VerifyBuildpackAPI(kind, name, requestedVersion string, logger log.Logger) error
}

// ConfigHandler reads configuration files for the lifecycle.
//
//go:generate mockgen -package testmock -destination testmock/config_handler.go github.com/buildpacks/lifecycle/phase ConfigHandler
type ConfigHandler interface {
	ReadAnalyzed(path string, logger log.Logger) (files.Analyzed, error)
	ReadGroup(path string) (buildpack.Group, error)
	ReadOrder(path string) (buildpack.Order, buildpack.Order, error)
	ReadRun(runPath string, logger log.Logger) (files.Run, error)
	ReadPlan(path string) (files.Plan, error)
}
