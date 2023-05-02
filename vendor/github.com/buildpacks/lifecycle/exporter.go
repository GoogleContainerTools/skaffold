package lifecycle

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/buildpacks/imgutil"
	"github.com/pkg/errors"

	"github.com/buildpacks/lifecycle/api"
	"github.com/buildpacks/lifecycle/buildpack"
	"github.com/buildpacks/lifecycle/cmd"
	"github.com/buildpacks/lifecycle/launch"
	"github.com/buildpacks/lifecycle/layers"
	"github.com/buildpacks/lifecycle/platform"
)

type Cache interface {
	Exists() bool
	Name() string
	SetMetadata(metadata platform.CacheMetadata) error
	RetrieveMetadata() (platform.CacheMetadata, error)
	AddLayerFile(tarPath string, sha string) error
	ReuseLayer(sha string) error
	RetrieveLayer(sha string) (io.ReadCloser, error)
	Commit() error
}

type Exporter struct {
	Buildpacks   []buildpack.GroupBuildpack
	LayerFactory LayerFactory
	Logger       Logger
	PlatformAPI  *api.Version
}

//go:generate mockgen -package testmock -destination testmock/layer_factory.go github.com/buildpacks/lifecycle LayerFactory
type LayerFactory interface {
	DirLayer(id string, dir string) (layers.Layer, error)
	LauncherLayer(path string) (layers.Layer, error)
	ProcessTypesLayer(metadata launch.Metadata) (layers.Layer, error)
	SliceLayers(dir string, slices []layers.Slice) ([]layers.Layer, error)
}

type LauncherConfig struct {
	Path     string
	Metadata platform.LauncherMetadata
}

type ExportOptions struct {
	LayersDir          string
	AppDir             string
	WorkingImage       imgutil.Image
	RunImageRef        string
	OrigMetadata       platform.LayersMetadata
	AdditionalNames    []string
	LauncherConfig     LauncherConfig
	Stack              platform.StackMetadata
	Project            platform.ProjectMetadata
	DefaultProcessType string
}

func (e *Exporter) Export(opts ExportOptions) (platform.ExportReport, error) {
	var err error

	opts.LayersDir, err = filepath.Abs(opts.LayersDir)
	if err != nil {
		return platform.ExportReport{}, errors.Wrapf(err, "layers dir absolute path")
	}

	opts.AppDir, err = filepath.Abs(opts.AppDir)
	if err != nil {
		return platform.ExportReport{}, errors.Wrapf(err, "app dir absolute path")
	}

	meta := platform.LayersMetadata{}
	meta.RunImage.TopLayer, err = opts.WorkingImage.TopLayer()
	if err != nil {
		return platform.ExportReport{}, errors.Wrap(err, "get run image top layer SHA")
	}

	meta.RunImage.Reference = opts.RunImageRef
	meta.Stack = opts.Stack

	buildMD := &platform.BuildMetadata{}
	if _, err := toml.DecodeFile(launch.GetMetadataFilePath(opts.LayersDir), buildMD); err != nil {
		return platform.ExportReport{}, errors.Wrap(err, "read build metadata")
	}

	// buildpack-provided layers
	if err := e.addBuildpackLayers(opts, &meta); err != nil {
		return platform.ExportReport{}, err
	}

	if e.PlatformAPI.AtLeast("0.8") {
		if err := e.addSBOMLaunchLayer(opts, &meta); err != nil {
			return platform.ExportReport{}, err
		}
	}

	// app layers (split into 1 or more slices)
	if err := e.addAppLayers(opts, buildMD.Slices, &meta); err != nil {
		return platform.ExportReport{}, errors.Wrap(err, "exporting app layers")
	}

	// launcher layers (launcher binary, launcher config, process symlinks)
	if err := e.addLauncherLayers(opts, buildMD, &meta); err != nil {
		return platform.ExportReport{}, err
	}

	if err := e.setLabels(opts, meta, buildMD); err != nil {
		return platform.ExportReport{}, err
	}

	if err := e.setEnv(opts, buildMD.ToLaunchMD()); err != nil {
		return platform.ExportReport{}, err
	}

	if e.PlatformAPI.AtLeast("0.6") {
		e.Logger.Debugf("Setting WORKDIR: '%s'", opts.AppDir)
		if err := e.setWorkingDir(opts); err != nil {
			return platform.ExportReport{}, errors.Wrap(err, "setting workdir")
		}
	}

	entrypoint, err := e.entrypoint(buildMD.ToLaunchMD(), opts.DefaultProcessType, buildMD.BuildpackDefaultProcessType)
	if err != nil {
		return platform.ExportReport{}, errors.Wrap(err, "determining entrypoint")
	}
	e.Logger.Debugf("Setting ENTRYPOINT: '%s'", entrypoint)
	if err = opts.WorkingImage.SetEntrypoint(entrypoint); err != nil {
		return platform.ExportReport{}, errors.Wrap(err, "setting entrypoint")
	}

	if err = opts.WorkingImage.SetCmd(); err != nil { // Note: Command intentionally empty
		return platform.ExportReport{}, errors.Wrap(err, "setting cmd")
	}

	report := platform.ExportReport{}
	report.Build, err = e.makeBuildReport(opts.LayersDir)
	if err != nil {
		return platform.ExportReport{}, err
	}
	report.Image, err = saveImage(opts.WorkingImage, opts.AdditionalNames, e.Logger)
	if err != nil {
		return platform.ExportReport{}, err
	}
	if !e.supportsManifestSize() {
		// unset manifest size in report.toml for old platform API versions
		report.Image.ManifestSize = 0
	}

	return report, nil
}

func (e *Exporter) addBuildpackLayers(opts ExportOptions, meta *platform.LayersMetadata) error {
	for _, bp := range e.Buildpacks {
		bpDir, err := buildpack.ReadLayersDir(opts.LayersDir, bp, e.Logger)
		e.Logger.Debugf("Processing buildpack directory: %s", bpDir.Path)
		if err != nil {
			return errors.Wrapf(err, "reading layers for buildpack '%s'", bp.ID)
		}
		bpMD := buildpack.LayersMetadata{
			ID:      bp.ID,
			Version: bp.Version,
			Layers:  map[string]buildpack.LayerMetadata{},
			Store:   bpDir.Store,
		}
		for _, fsLayer := range bpDir.FindLayers(buildpack.MadeLaunch) {
			fsLayer := fsLayer
			e.Logger.Debugf("Processing launch layer: %s", fsLayer.Path())
			lmd, err := fsLayer.Read()
			if err != nil {
				return errors.Wrapf(err, "reading '%s' metadata", fsLayer.Identifier())
			}

			if fsLayer.HasLocalContents() {
				layer, err := e.LayerFactory.DirLayer(fsLayer.Identifier(), fsLayer.Path())
				if err != nil {
					return errors.Wrapf(err, "creating layer")
				}
				origLayerMetadata := opts.OrigMetadata.MetadataForBuildpack(bp.ID).Layers[fsLayer.Name()]
				lmd.SHA, err = e.addOrReuseLayer(opts.WorkingImage, layer, origLayerMetadata.SHA)
				if err != nil {
					return err
				}
			} else {
				if lmd.Cache {
					return fmt.Errorf("layer '%s' is cache=true but has no contents", fsLayer.Identifier())
				}
				origLayerMetadata, ok := opts.OrigMetadata.MetadataForBuildpack(bp.ID).Layers[fsLayer.Name()]
				if !ok {
					return fmt.Errorf("cannot reuse '%s', previous image has no metadata for layer '%s'", fsLayer.Identifier(), fsLayer.Identifier())
				}

				e.Logger.Infof("Reusing layer '%s'\n", fsLayer.Identifier())
				e.Logger.Debugf("Layer '%s' SHA: %s\n", fsLayer.Identifier(), origLayerMetadata.SHA)
				if err := opts.WorkingImage.ReuseLayer(origLayerMetadata.SHA); err != nil {
					return errors.Wrapf(err, "reusing layer: '%s'", fsLayer.Identifier())
				}
				lmd.SHA = origLayerMetadata.SHA
			}
			bpMD.Layers[fsLayer.Name()] = lmd
		}
		meta.Buildpacks = append(meta.Buildpacks, bpMD)

		if malformedLayers := bpDir.FindLayers(buildpack.Malformed); len(malformedLayers) > 0 {
			ids := make([]string, 0, len(malformedLayers))
			for _, ml := range malformedLayers {
				ids = append(ids, ml.Identifier())
			}
			return fmt.Errorf("failed to parse metadata for layers '%s'", ids)
		}
	}
	return nil
}

func (e *Exporter) addLauncherLayers(opts ExportOptions, buildMD *platform.BuildMetadata, meta *platform.LayersMetadata) error {
	launcherLayer, err := e.LayerFactory.LauncherLayer(opts.LauncherConfig.Path)
	if err != nil {
		return errors.Wrap(err, "creating launcher layers")
	}
	meta.Launcher.SHA, err = e.addOrReuseLayer(opts.WorkingImage, launcherLayer, opts.OrigMetadata.Launcher.SHA)
	if err != nil {
		return errors.Wrap(err, "exporting launcher configLayer")
	}
	configLayer, err := e.LayerFactory.DirLayer("config", filepath.Join(opts.LayersDir, "config"))
	if err != nil {
		return errors.Wrapf(err, "creating layer '%s'", configLayer.ID)
	}
	meta.Config.SHA, err = e.addOrReuseLayer(opts.WorkingImage, configLayer, opts.OrigMetadata.Config.SHA)
	if err != nil {
		return errors.Wrap(err, "exporting config layer")
	}

	if err := e.launcherConfig(opts, buildMD, meta); err != nil {
		return err
	}
	return nil
}

func (e *Exporter) addAppLayers(opts ExportOptions, slices []layers.Slice, meta *platform.LayersMetadata) error {
	// creating app layers (slices + app dir)
	sliceLayers, err := e.LayerFactory.SliceLayers(opts.AppDir, slices)
	if err != nil {
		return errors.Wrap(err, "creating app layers")
	}

	var numberOfReusedLayers int
	for _, slice := range sliceLayers {
		var err error

		found := false
		for _, previous := range opts.OrigMetadata.App {
			if slice.Digest == previous.SHA {
				found = true
				break
			}
		}
		if found {
			err = opts.WorkingImage.ReuseLayer(slice.Digest)
			numberOfReusedLayers++
		} else {
			err = opts.WorkingImage.AddLayerWithDiffID(slice.TarPath, slice.Digest)
		}
		if err != nil {
			return err
		}
		e.Logger.Debugf("Layer '%s' SHA: %s\n", slice.ID, slice.Digest)
		meta.App = append(meta.App, platform.LayerMetadata{SHA: slice.Digest})
	}

	delta := len(sliceLayers) - numberOfReusedLayers
	if numberOfReusedLayers > 0 {
		e.Logger.Infof("Reusing %d/%d app layer(s)\n", numberOfReusedLayers, len(sliceLayers))
	}
	if delta != 0 {
		e.Logger.Infof("Adding %d/%d app layer(s)\n", delta, len(sliceLayers))
	}
	return nil
}

func (e *Exporter) setLabels(opts ExportOptions, meta platform.LayersMetadata, buildMD *platform.BuildMetadata) error {
	data, err := json.Marshal(meta)
	if err != nil {
		return errors.Wrap(err, "marshall metadata")
	}

	e.Logger.Infof("Adding label '%s'", platform.LayerMetadataLabel)
	if err = opts.WorkingImage.SetLabel(platform.LayerMetadataLabel, string(data)); err != nil {
		return errors.Wrap(err, "set app image metadata label")
	}

	buildMD.Launcher = opts.LauncherConfig.Metadata
	buildJSON, err := json.Marshal(buildMD)
	if err != nil {
		return errors.Wrap(err, "parse build metadata")
	}

	e.Logger.Infof("Adding label '%s'", platform.BuildMetadataLabel)
	if err := opts.WorkingImage.SetLabel(platform.BuildMetadataLabel, string(buildJSON)); err != nil {
		return errors.Wrap(err, "set build image metadata label")
	}

	projectJSON, err := json.Marshal(opts.Project)
	if err != nil {
		return errors.Wrap(err, "parse project metadata")
	}

	e.Logger.Infof("Adding label '%s'", platform.ProjectMetadataLabel)
	if err := opts.WorkingImage.SetLabel(platform.ProjectMetadataLabel, string(projectJSON)); err != nil {
		return errors.Wrap(err, "set project metadata label")
	}

	for _, label := range buildMD.Labels {
		e.Logger.Infof("Adding label '%s'", label.Key)
		if err := opts.WorkingImage.SetLabel(label.Key, label.Value); err != nil {
			return errors.Wrapf(err, "set buildpack-provided label '%s'", label.Key)
		}
	}
	return nil
}

func (e *Exporter) setEnv(opts ExportOptions, launchMD launch.Metadata) error {
	e.Logger.Debugf("Setting %s=%s", cmd.EnvLayersDir, opts.LayersDir)
	if err := opts.WorkingImage.SetEnv(cmd.EnvLayersDir, opts.LayersDir); err != nil {
		return errors.Wrapf(err, "set app image env %s", cmd.EnvLayersDir)
	}

	e.Logger.Debugf("Setting %s=%s", cmd.EnvAppDir, opts.AppDir)
	if err := opts.WorkingImage.SetEnv(cmd.EnvAppDir, opts.AppDir); err != nil {
		return errors.Wrapf(err, "set app image env %s", cmd.EnvAppDir)
	}

	e.Logger.Debugf("Setting %s=%s", cmd.EnvPlatformAPI, e.PlatformAPI.String())
	if err := opts.WorkingImage.SetEnv(cmd.EnvPlatformAPI, e.PlatformAPI.String()); err != nil {
		return errors.Wrapf(err, "set app image env %s", cmd.EnvAppDir)
	}

	e.Logger.Debugf("Setting %s=%s", cmd.EnvDeprecationMode, cmd.DeprecationModeQuiet)
	if err := opts.WorkingImage.SetEnv(cmd.EnvDeprecationMode, cmd.DeprecationModeQuiet); err != nil {
		return errors.Wrapf(err, "set app image env %s", cmd.EnvAppDir)
	}

	if e.supportsMulticallLauncher() {
		path, err := opts.WorkingImage.Env("PATH")
		if err != nil {
			return errors.Wrap(err, "failed to get PATH from app image")
		}
		path = strings.Join([]string{launch.ProcessDir, launch.LifecycleDir, path}, string(os.PathListSeparator))
		e.Logger.Debugf("Prepending %s and %s to PATH", launch.ProcessDir, launch.LifecycleDir)
		if err := opts.WorkingImage.SetEnv("PATH", path); err != nil {
			return errors.Wrap(err, "set app image env PATH")
		}
	} else if opts.DefaultProcessType != "" {
		if _, ok := launchMD.FindProcessType(opts.DefaultProcessType); !ok {
			return processTypeError(launchMD, opts.DefaultProcessType)
		}
		e.Logger.Debugf("Setting %s=%s", cmd.EnvProcessType, opts.DefaultProcessType)
		if err := opts.WorkingImage.SetEnv(cmd.EnvProcessType, opts.DefaultProcessType); err != nil {
			return errors.Wrapf(err, "set app image env %s", cmd.EnvProcessType)
		}
	}
	return nil
}

func (e *Exporter) setWorkingDir(opts ExportOptions) error {
	return opts.WorkingImage.SetWorkingDir(opts.AppDir)
}

func (e *Exporter) entrypoint(launchMD launch.Metadata, userDefaultProcessType, buildpackDefaultProcessType string) (string, error) {
	if !e.supportsMulticallLauncher() {
		return launch.LauncherPath, nil
	}

	if userDefaultProcessType == "" && e.PlatformAPI.LessThan("0.6") && len(launchMD.Processes) == 1 {
		// if there is only one process, we set it to the default for platform API < 0.6
		e.Logger.Infof("Setting default process type '%s'", launchMD.Processes[0].Type)
		return launch.ProcessPath(launchMD.Processes[0].Type), nil
	}

	if userDefaultProcessType != "" {
		defaultProcess, ok := launchMD.FindProcessType(userDefaultProcessType)
		if !ok {
			if e.PlatformAPI.LessThan("0.6") {
				e.Logger.Warn(processTypeWarning(launchMD, userDefaultProcessType))
				return launch.LauncherPath, nil
			}
			return "", fmt.Errorf("tried to set %s to default but it doesn't exist", userDefaultProcessType)
		}
		e.Logger.Infof("Setting default process type '%s'", defaultProcess.Type)
		return launch.ProcessPath(defaultProcess.Type), nil
	}
	if buildpackDefaultProcessType == "" {
		e.Logger.Info("no default process type")
		return launch.LauncherPath, nil
	}
	e.Logger.Infof("Setting default process type '%s'", buildpackDefaultProcessType)
	return launch.ProcessPath(buildpackDefaultProcessType), nil
}

// processTypes adds
func (e *Exporter) launcherConfig(opts ExportOptions, buildMD *platform.BuildMetadata, meta *platform.LayersMetadata) error {
	if e.supportsMulticallLauncher() {
		launchMD := launch.Metadata{
			Processes: buildMD.Processes,
		}
		if len(buildMD.Processes) > 0 {
			processTypesLayer, err := e.LayerFactory.ProcessTypesLayer(launchMD)
			if err != nil {
				return errors.Wrapf(err, "creating layer '%s'", processTypesLayer.ID)
			}
			meta.ProcessTypes.SHA, err = e.addOrReuseLayer(opts.WorkingImage, processTypesLayer, opts.OrigMetadata.ProcessTypes.SHA)
			if err != nil {
				return errors.Wrapf(err, "exporting layer '%s'", processTypesLayer.ID)
			}
		}
	}
	return nil
}

func (e *Exporter) supportsMulticallLauncher() bool {
	return e.PlatformAPI.AtLeast("0.4")
}

func (e *Exporter) supportsManifestSize() bool {
	return e.PlatformAPI.AtLeast("0.6")
}

func processTypeError(launchMD launch.Metadata, defaultProcessType string) error {
	return fmt.Errorf(processTypeWarning(launchMD, defaultProcessType))
}

func processTypeWarning(launchMD launch.Metadata, defaultProcessType string) string {
	var typeList []string
	for _, p := range launchMD.Processes {
		typeList = append(typeList, p.Type)
	}
	return fmt.Sprintf("default process type '%s' not present in list %+v", defaultProcessType, typeList)
}

func (e *Exporter) addOrReuseLayer(image imgutil.Image, layer layers.Layer, previousSHA string) (string, error) {
	layer, err := e.LayerFactory.DirLayer(layer.ID, layer.TarPath)
	if err != nil {
		return "", errors.Wrapf(err, "creating layer '%s'", layer.ID)
	}
	if layer.Digest == previousSHA {
		e.Logger.Infof("Reusing layer '%s'\n", layer.ID)
		e.Logger.Debugf("Layer '%s' SHA: %s\n", layer.ID, layer.Digest)
		return layer.Digest, image.ReuseLayer(previousSHA)
	}
	e.Logger.Infof("Adding layer '%s'\n", layer.ID)
	e.Logger.Debugf("Layer '%s' SHA: %s\n", layer.ID, layer.Digest)
	return layer.Digest, image.AddLayerWithDiffID(layer.TarPath, layer.Digest)
}

func (e *Exporter) makeBuildReport(layersDir string) (platform.BuildReport, error) {
	if e.PlatformAPI.LessThan("0.5") {
		return platform.BuildReport{}, nil
	}
	var out []buildpack.BOMEntry
	for _, bp := range e.Buildpacks {
		if api.MustParse(bp.API).LessThan("0.5") {
			continue
		}
		var bpBuildReport platform.BuildReport
		bpBuildTOML := filepath.Join(layersDir, launch.EscapeID(bp.ID), "build.toml")
		if _, err := toml.DecodeFile(bpBuildTOML, &bpBuildReport); err != nil && !os.IsNotExist(err) {
			return platform.BuildReport{}, err
		}
		out = append(out, buildpack.WithBuildpack(bp, bpBuildReport.BOM)...)
	}
	return platform.BuildReport{BOM: out}, nil
}

func (e *Exporter) addSBOMLaunchLayer(opts ExportOptions, meta *platform.LayersMetadata) error {
	sbomLaunchDir, err := readLayersSBOM(opts.LayersDir, "launch", e.Logger)
	if err != nil {
		return errors.Wrap(err, "failed to read layers config sbom")
	}

	if sbomLaunchDir != nil {
		layer, err := e.LayerFactory.DirLayer(sbomLaunchDir.Identifier(), sbomLaunchDir.Path())
		if err != nil {
			return errors.Wrapf(err, "creating layer")
		}

		var originalSHA string
		if opts.OrigMetadata.BOM != nil {
			originalSHA = opts.OrigMetadata.BOM.SHA
		}

		sha, err := e.addOrReuseLayer(opts.WorkingImage, layer, originalSHA)
		if err != nil {
			return errors.Wrapf(err, "exporting layer '%s'", layer.ID)
		}

		meta.BOM = &platform.LayerMetadata{SHA: sha}
	}

	return nil
}
