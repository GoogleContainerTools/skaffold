package phase

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"

	"github.com/pkg/errors"

	"github.com/buildpacks/lifecycle/api"
	"github.com/buildpacks/lifecycle/buildpack"
	"github.com/buildpacks/lifecycle/env"
	"github.com/buildpacks/lifecycle/internal/encoding"
	"github.com/buildpacks/lifecycle/internal/fsutil"
	"github.com/buildpacks/lifecycle/launch"
	"github.com/buildpacks/lifecycle/layers"
	"github.com/buildpacks/lifecycle/log"
	"github.com/buildpacks/lifecycle/platform"
	"github.com/buildpacks/lifecycle/platform/files"
)

type Platform interface {
	API() *api.Version
}

// BuildEnv encapsulates modifications that the lifecycle can make to buildpacks' build environment.
//
//go:generate mockgen -package testmock -destination testmock/build_env.go github.com/buildpacks/lifecycle/phase BuildEnv
type BuildEnv interface {
	AddRootDir(baseDir string) error
	AddEnvDir(envDir string, defaultAction env.ActionType) error
	WithOverrides(platformDir string, baseConfigDir string) ([]string, error)
	List() []string
}

type Builder struct {
	AppDir         string
	BuildConfigDir string
	LayersDir      string
	PlatformDir    string
	BuildExecutor  buildpack.BuildExecutor
	DirStore       DirStore
	Group          buildpack.Group
	Logger         log.Logger
	Out, Err       io.Writer
	Plan           files.Plan
	PlatformAPI    *api.Version
	AnalyzeMD      files.Analyzed
}

func (b *Builder) Build() (*files.BuildMetadata, error) {
	defer log.NewMeasurement("Builder", b.Logger)()

	// ensure layers SBOM directory is removed
	if err := os.RemoveAll(filepath.Join(b.LayersDir, "sbom")); err != nil {
		return nil, errors.Wrap(err, "cleaning layers SBOM directory")
	}

	var (
		bomFiles  []buildpack.BOMFile
		buildBOM  []buildpack.BOMEntry
		labels    []buildpack.Label
		launchBOM []buildpack.BOMEntry
		slices    []layers.Slice
	)
	processMap := newProcessMap()
	inputs := b.getBuildInputs()

	filteredPlan := b.Plan

	for _, bp := range b.Group.Group {
		b.Logger.Debugf("Running build for buildpack %s", bp)

		b.Logger.Debug("Looking up buildpack")
		bpTOML, err := b.DirStore.LookupBp(bp.ID, bp.Version)
		if err != nil {
			return nil, err
		}

		b.Logger.Debug("Finding plan")
		inputs.Plan = filteredPlan.Find(buildpack.KindBuildpack, bp.ID)

		br, err := b.BuildExecutor.Build(*bpTOML, inputs, b.Logger)
		if err != nil {
			return nil, err
		}

		b.Logger.Debug("Updating buildpack processes")

		bomFiles = append(bomFiles, br.BOMFiles...)
		buildBOM = append(buildBOM, br.BuildBOM...)
		filteredPlan = filteredPlan.Filter(br.MetRequires)
		labels = append(labels, br.Labels...)
		launchBOM = append(launchBOM, br.LaunchBOM...)
		slices = append(slices, br.Slices...)

		b.Logger.Debug("Updating process list")
		warning := processMap.add(br.Processes)
		if warning != "" {
			b.Logger.Warn(warning)
		}

		b.Logger.Debugf("Finished running build for buildpack %s", bp)
	}

	if b.PlatformAPI.AtLeast("0.8") {
		b.Logger.Debug("Copying SBOM files")
		if err := b.copySBOMFiles(inputs.LayersDir, bomFiles); err != nil {
			return nil, err
		}
	}

	if b.PlatformAPI.AtLeast("0.9") {
		b.Logger.Debug("Creating SBOM files for legacy BOM")
		if err := encoding.WriteJSON(filepath.Join(b.LayersDir, "sbom", "launch", "sbom.legacy.json"), launchBOM); err != nil {
			return nil, errors.Wrap(err, "encoding launch bom")
		}
		if err := encoding.WriteJSON(filepath.Join(b.LayersDir, "sbom", "build", "sbom.legacy.json"), buildBOM); err != nil {
			return nil, errors.Wrap(err, "encoding build bom")
		}
		launchBOM = []buildpack.BOMEntry{}
	}

	b.Logger.Debug("Listing processes")
	procList := processMap.list(b.PlatformAPI)

	// Don't redundantly print `extension = true` and `optional = true` in metadata.toml and metadata label
	for i, ext := range b.Group.GroupExtensions {
		b.Group.GroupExtensions[i] = ext.NoExtension().NoOpt()
	}

	return &files.BuildMetadata{
		BOM:                         launchBOM,
		Buildpacks:                  b.Group.Group,
		Extensions:                  b.Group.GroupExtensions,
		Labels:                      labels,
		Processes:                   procList,
		Slices:                      slices,
		BuildpackDefaultProcessType: processMap.defaultType,
	}, nil
}

func (b *Builder) getBuildInputs() buildpack.BuildInputs {
	return buildpack.BuildInputs{
		AppDir:         b.AppDir,
		BuildConfigDir: b.BuildConfigDir,
		LayersDir:      b.LayersDir,
		PlatformDir:    b.PlatformDir,
		Env:            env.NewBuildEnv(os.Environ()),
		TargetEnv:      platform.EnvVarsFor(&fsutil.Detect{}, b.AnalyzeMD.RunImageTarget(), b.Logger),
		Out:            b.Out,
		Err:            b.Err,
	}
}

// copySBOMFiles() copies any BOM files written by buildpacks during the Build() process
// to their appropriate locations, in preparation for its final application layer.
// This function handles both BOMs that are associated with a layer directory and BOMs that are not
// associated with a layer directory, since "bomFile.LayerName" will be "" in the latter case.
//
// Before:
// /layers
// └── buildpack.id
//
//	├── A
//	│   └── ...
//	├── A.sbom.cdx.json
//	└── launch.sbom.cdx.json
//
// After:
// /layers
// └── sbom
//
//	└── launch
//	    └── buildpack.id
//	        ├── A
//	        │   └── sbom.cdx.json
//	        └── sbom.cdx.json
func (b *Builder) copySBOMFiles(layersDir string, bomFiles []buildpack.BOMFile) error {
	var (
		buildSBOMDir  = filepath.Join(layersDir, "sbom", "build")
		cacheSBOMDir  = filepath.Join(layersDir, "sbom", "cache")
		launchSBOMDir = filepath.Join(layersDir, "sbom", "launch")
		copyBOMFileTo = func(bomFile buildpack.BOMFile, sbomDir string) error {
			targetDir := filepath.Join(sbomDir, launch.EscapeID(bomFile.BuildpackID), bomFile.LayerName)
			err := os.MkdirAll(targetDir, os.ModePerm)
			if err != nil {
				return err
			}

			name, err := bomFile.Name()
			if err != nil {
				return err
			}

			return fsutil.Copy(bomFile.Path, filepath.Join(targetDir, name))
		}
	)

	for _, bomFile := range bomFiles {
		switch bomFile.LayerType {
		case buildpack.LayerTypeBuild:
			if err := copyBOMFileTo(bomFile, buildSBOMDir); err != nil {
				return err
			}
		case buildpack.LayerTypeCache:
			if err := copyBOMFileTo(bomFile, cacheSBOMDir); err != nil {
				return err
			}
		case buildpack.LayerTypeLaunch:
			if err := copyBOMFileTo(bomFile, launchSBOMDir); err != nil {
				return err
			}
		}
	}

	return nil
}

type processMap struct {
	typeToProcess map[string]launch.Process
	defaultType   string
}

func newProcessMap() processMap {
	return processMap{
		typeToProcess: make(map[string]launch.Process),
		defaultType:   "",
	}
}

// This function adds the processes from listToAdd to processMap
// it sets m.defaultType to the last default process
// if a non-default process overrides a default process, it returns a warning and unset m.defaultType
func (m *processMap) add(listToAdd []launch.Process) string {
	warning := ""
	for _, procToAdd := range listToAdd {
		if procToAdd.Default {
			m.defaultType = procToAdd.Type
			warning = ""
		} else if procToAdd.Type == m.defaultType {
			// non-default process overrides a default process
			m.defaultType = ""
			warning = fmt.Sprintf("Warning: redefining the following default process type with a process not marked as default: %s\n", procToAdd.Type)
		}
		m.typeToProcess[procToAdd.Type] = procToAdd
	}
	return warning
}

// list returns a sorted array of processes.
// The array is sorted based on the process types.
// The list is sorted for reproducibility.
func (m processMap) list(platformAPI *api.Version) []launch.Process {
	var keys []string
	for proc := range m.typeToProcess {
		keys = append(keys, proc)
	}
	sort.Strings(keys)
	result := []launch.Process{}
	for _, key := range keys {
		result = append(result, m.typeToProcess[key].NoDefault().WithPlatformAPI(platformAPI)) // we set the default to false so it won't be part of metadata.toml
	}
	return result
}
