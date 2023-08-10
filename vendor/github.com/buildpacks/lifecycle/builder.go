package lifecycle

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
	io2 "github.com/buildpacks/lifecycle/internal/io"
	"github.com/buildpacks/lifecycle/launch"
	"github.com/buildpacks/lifecycle/layers"
	"github.com/buildpacks/lifecycle/platform"
)

type BuildEnv interface {
	AddRootDir(baseDir string) error
	AddEnvDir(envDir string, defaultAction env.ActionType) error
	WithPlatform(platformDir string) ([]string, error)
	List() []string
}

type BuildpackStore interface {
	Lookup(bpID, bpVersion string) (buildpack.Buildpack, error)
}

type Buildpack interface {
	Build(bpPlan buildpack.Plan, config buildpack.BuildConfig, bpEnv buildpack.BuildEnv) (buildpack.BuildResult, error)
	ConfigFile() *buildpack.Descriptor
	Detect(config *buildpack.DetectConfig, bpEnv buildpack.BuildEnv) buildpack.DetectRun
}

type Builder struct {
	AppDir         string
	LayersDir      string
	PlatformDir    string
	Platform       Platform
	Group          buildpack.Group
	Plan           platform.BuildPlan
	Out, Err       io.Writer
	Logger         Logger
	BuildpackStore BuildpackStore
}

func (b *Builder) Build() (*platform.BuildMetadata, error) {
	b.Logger.Debug("Starting build")

	// ensure layers sbom directory is removed
	if err := os.RemoveAll(filepath.Join(b.LayersDir, "sbom")); err != nil {
		return nil, errors.Wrap(err, "cleaning layers sbom directory")
	}

	config, err := b.BuildConfig()
	if err != nil {
		return nil, err
	}

	processMap := newProcessMap()
	plan := b.Plan
	var bom []buildpack.BOMEntry
	var bomFiles []buildpack.BOMFile
	var slices []layers.Slice
	var labels []buildpack.Label

	bpEnv := env.NewBuildEnv(os.Environ())

	for _, bp := range b.Group.Group {
		b.Logger.Debugf("Running build for buildpack %s", bp)

		b.Logger.Debug("Looking up buildpack")
		bpTOML, err := b.BuildpackStore.Lookup(bp.ID, bp.Version)
		if err != nil {
			return nil, err
		}

		b.Logger.Debug("Finding plan")
		bpPlan := plan.Find(bp.ID)

		br, err := bpTOML.Build(bpPlan, config, bpEnv)
		if err != nil {
			return nil, err
		}

		b.Logger.Debug("Updating buildpack processes")
		updateDefaultProcesses(br.Processes, api.MustParse(bp.API), b.Platform.API())

		bom = append(bom, br.BOM...)
		bomFiles = append(bomFiles, br.BOMFiles...)
		labels = append(labels, br.Labels...)
		plan = plan.Filter(br.MetRequires)

		b.Logger.Debug("Updating process list")
		warning := processMap.add(br.Processes)
		if warning != "" {
			b.Logger.Warn(warning)
		}

		slices = append(slices, br.Slices...)

		b.Logger.Debugf("Finished running build for buildpack %s", bp)
	}

	if b.Platform.API().LessThan("0.4") {
		config.Logger.Debug("Updating BOM entries")
		for i := range bom {
			bom[i].ConvertMetadataToVersion()
		}
	}

	if b.Platform.API().AtLeast("0.8") {
		b.Logger.Debug("Copying sBOM files")
		err = b.copyBOMFiles(config.LayersDir, bomFiles)
		if err != nil {
			return nil, err
		}
	}

	b.Logger.Debug("Listing processes")
	procList := processMap.list()

	b.Logger.Debug("Finished build")
	return &platform.BuildMetadata{
		BOM:                         bom,
		Buildpacks:                  b.Group.Group,
		Labels:                      labels,
		Processes:                   procList,
		Slices:                      slices,
		BuildpackDefaultProcessType: processMap.defaultType,
	}, nil
}

// copyBOMFiles() copies any BOM files written by buildpacks during the Build() process
// to their appropriate locations, in preparation for its final application layer.
// This function handles both BOMs that are associated with a layer directory and BOMs that are not
// associated with a layer directory, since "bomFile.LayerName" will be "" in the latter case.
//
// Before:
// /layers
// └── buildpack.id
//     ├── A
//     │   └── ...
//     ├── A.sbom.cdx.json
//     └── launch.sbom.cdx.json
//
// After:
// /layers
// └── sbom
//     └── launch
//         └── buildpack.id
//             ├── A
//             │   └── sbom.cdx.json
//             └── sbom.cdx.json
func (b *Builder) copyBOMFiles(layersDir string, bomFiles []buildpack.BOMFile) error {
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

			return io2.Copy(bomFile.Path, filepath.Join(targetDir, name))
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

// we set default = true for web processes when platformAPI >= 0.6 and buildpackAPI < 0.6
func updateDefaultProcesses(processes []launch.Process, buildpackAPI *api.Version, platformAPI *api.Version) {
	if platformAPI.LessThan("0.6") || buildpackAPI.AtLeast("0.6") {
		return
	}

	for i := range processes {
		if processes[i].Type == "web" {
			processes[i].Default = true
		}
	}
}

func (b *Builder) BuildConfig() (buildpack.BuildConfig, error) {
	appDir, err := filepath.Abs(b.AppDir)
	if err != nil {
		return buildpack.BuildConfig{}, err
	}
	platformDir, err := filepath.Abs(b.PlatformDir)
	if err != nil {
		return buildpack.BuildConfig{}, err
	}
	layersDir, err := filepath.Abs(b.LayersDir)
	if err != nil {
		return buildpack.BuildConfig{}, err
	}

	return buildpack.BuildConfig{
		AppDir:      appDir,
		PlatformDir: platformDir,
		LayersDir:   layersDir,
		Out:         b.Out,
		Err:         b.Err,
		Logger:      b.Logger,
	}, nil
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
func (m processMap) list() []launch.Process {
	var keys []string
	for proc := range m.typeToProcess {
		keys = append(keys, proc)
	}
	sort.Strings(keys)
	result := []launch.Process{}
	for _, key := range keys {
		result = append(result, m.typeToProcess[key].NoDefault()) // we set the default to false so it won't be part of metadata.toml
	}
	return result
}
