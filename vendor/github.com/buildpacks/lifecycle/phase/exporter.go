package phase

import (
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/buildpacks/imgutil"
	"github.com/buildpacks/imgutil/local"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/pkg/errors"

	"github.com/buildpacks/lifecycle/api"
	"github.com/buildpacks/lifecycle/buildpack"
	"github.com/buildpacks/lifecycle/image"
	"github.com/buildpacks/lifecycle/internal/fsutil"
	"github.com/buildpacks/lifecycle/launch"
	"github.com/buildpacks/lifecycle/layers"
	"github.com/buildpacks/lifecycle/log"
	"github.com/buildpacks/lifecycle/platform"
	"github.com/buildpacks/lifecycle/platform/files"
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
	VerifyLayer(sha string) error
}

type Exporter struct {
	Buildpacks   []buildpack.GroupElement
	LayerFactory LayerFactory
	Logger       log.Logger
	PlatformAPI  *api.Version
}

// LayerFactory given a directory on the local filesystem will return a `layers.Layer`
// that can be used to construct an OCI image.
//
//go:generate mockgen -package testmock -destination testmock/layer_factory.go github.com/buildpacks/lifecycle/phase LayerFactory
type LayerFactory interface {
	DirLayer(id string, dir string, createdBy string) (layers.Layer, error)
	LauncherLayer(path string) (layers.Layer, error)
	ProcessTypesLayer(metadata launch.Metadata) (layers.Layer, error)
	SliceLayers(dir string, slices []layers.Slice) ([]layers.Layer, error)
	TarLayer(withID string, fromTarPath string, createdBy string) (layer layers.Layer, err error)
}

type LauncherConfig struct {
	Path     string
	SBOMDir  string
	Metadata files.LauncherMetadata
}

type ExportOptions struct {
	// WorkingImage is the image to save.
	WorkingImage imgutil.Image
	// AdditionalNames are additional tags to save to, besides WorkingImage.Name().
	AdditionalNames []string
	// ExtendedDir is the location of extension-provided layers.
	ExtendedDir string
	// AppDir is the source directory.
	AppDir string
	// LayersDir is the location of buildpack-provided layers.
	LayersDir string
	// OrigMetadata was read from the previous image during the `analyze` phase, and is used to determine if a previously-uploaded layer can be re-used.
	OrigMetadata files.LayersMetadata
	// LauncherConfig is the launcher config.
	LauncherConfig LauncherConfig
	// DefaultProcessType is the user-provided default process type.
	DefaultProcessType string
	// RunImageRef is the run image reference for the layer metadata label.
	RunImageRef string
	// RunImageForExport is run image metadata for the layer metadata label for Platform API >= 0.12.
	RunImageForExport files.RunImageForExport
	// Project is project metadata for the project metadata label.
	Project files.ProjectMetadata
}

func (e *Exporter) Export(opts ExportOptions) (files.Report, error) {
	var err error
	defer log.NewMeasurement("Exporter", e.Logger)()

	if e.PlatformAPI.AtLeast("0.11") {
		if err = e.copyBuildpacksioSBOMs(opts); err != nil {
			return files.Report{}, errors.Wrapf(err, "failed to copy buildpacksio SBOMs")
		}
	}

	opts.LayersDir, err = filepath.Abs(opts.LayersDir)
	if err != nil {
		return files.Report{}, errors.Wrapf(err, "layers dir absolute path")
	}

	opts.AppDir, err = filepath.Abs(opts.AppDir)
	if err != nil {
		return files.Report{}, errors.Wrapf(err, "app dir absolute path")
	}

	meta := files.LayersMetadata{}
	meta.RunImage.TopLayer, err = opts.WorkingImage.TopLayer()
	if err != nil {
		return files.Report{}, errors.Wrap(err, "get run image top layer SHA")
	}
	meta.RunImage.Reference = opts.RunImageRef

	if e.PlatformAPI.AtLeast("0.12") {
		meta.RunImage.Image = opts.RunImageForExport.Image
		meta.RunImage.Mirrors = opts.RunImageForExport.Mirrors
	}
	// ensure we always copy the new RunImage into the old stack to preserve old behavior
	meta.Stack = &files.Stack{RunImage: opts.RunImageForExport}

	buildMD, err := files.Handler.ReadBuildMetadata(launch.GetMetadataFilePath(opts.LayersDir), e.PlatformAPI)
	if err != nil {
		return files.Report{}, errors.Wrap(err, "read build metadata")
	}

	// extension-provided layers
	if err := e.addExtensionLayers(opts); err != nil {
		return files.Report{}, err
	}

	// buildpack-provided layers
	if err := e.addBuildpackLayers(opts, &meta); err != nil {
		return files.Report{}, err
	}

	if e.PlatformAPI.AtLeast("0.8") {
		if err := e.addSBOMLaunchLayer(opts, &meta); err != nil {
			return files.Report{}, err
		}
	}

	// app layers (split into 1 or more slices)
	if err := e.addAppLayers(opts, buildMD.Slices, &meta); err != nil {
		return files.Report{}, errors.Wrap(err, "exporting app layers")
	}

	// launcher layers (launcher binary, launcher config, process symlinks)
	if err := e.addLauncherLayers(opts, buildMD, &meta); err != nil {
		return files.Report{}, err
	}

	if err := e.setLabels(opts, meta, buildMD); err != nil {
		return files.Report{}, err
	}

	if err := e.setEnv(opts); err != nil {
		return files.Report{}, err
	}

	e.Logger.Debugf("Setting WORKDIR: '%s'", opts.AppDir)
	if err := e.setWorkingDir(opts); err != nil {
		return files.Report{}, errors.Wrap(err, "setting workdir")
	}

	entrypoint, err := e.entrypoint(buildMD.ToLaunchMD(), opts.DefaultProcessType, buildMD.BuildpackDefaultProcessType)
	if err != nil {
		return files.Report{}, errors.Wrap(err, "determining entrypoint")
	}
	e.Logger.Debugf("Setting ENTRYPOINT: '%s'", entrypoint)
	if err = opts.WorkingImage.SetEntrypoint(entrypoint); err != nil {
		return files.Report{}, errors.Wrap(err, "setting entrypoint")
	}

	if err = opts.WorkingImage.SetCmd(); err != nil { // Note: Command intentionally empty
		return files.Report{}, errors.Wrap(err, "setting cmd")
	}

	report := files.Report{}
	report.Build, err = e.makeBuildReport(opts.LayersDir)
	if err != nil {
		return files.Report{}, err
	}
	report.Image, err = saveImage(opts.WorkingImage, opts.AdditionalNames, e.Logger)
	if err != nil {
		return files.Report{}, err
	}
	return report, nil
}

func SBOMExtensions() []string {
	return []string{buildpack.ExtensionCycloneDX, buildpack.ExtensionSPDX, buildpack.ExtensionSyft}
}

func (e *Exporter) copyBuildpacksioSBOMs(opts ExportOptions) error {
	targetBuildDir := filepath.Join(opts.LayersDir, "sbom", "build", launch.EscapeID("buildpacksio/lifecycle"))
	if err := e.copyDefaultSBOMsForComponent("lifecycle", targetBuildDir); err != nil {
		return err
	}

	targetLaunchDir := filepath.Join(opts.LayersDir, "sbom", "launch", launch.EscapeID("buildpacksio/lifecycle"), "launcher")
	switch {
	case opts.LauncherConfig.SBOMDir == "" ||
		opts.LauncherConfig.SBOMDir == platform.DefaultBuildpacksioSBOMDir:
		return e.copyDefaultSBOMsForComponent("launcher", targetLaunchDir)
	default:
		// if provided a custom launcher SBOM directory, copy all files that look like sboms in that directory
		return e.copyLauncherSBOMs(opts.LauncherConfig.SBOMDir, targetLaunchDir)
	}
}

func (e *Exporter) copyLauncherSBOMs(srcDir string, dstDir string) error {
	sboms, err := fsutil.FilesWithExtensions(srcDir, SBOMExtensions())
	if err != nil {
		e.Logger.Warnf("Failed to list contents of directory %s", srcDir)
		return err
	}

	if err := os.MkdirAll(dstDir, 0755); err != nil {
		return err
	}

	for _, sbom := range sboms {
		dstPath := filepath.Join(dstDir, filepath.Base(sbom))
		err = fsutil.Copy(sbom, dstPath)
		if err != nil {
			e.Logger.Warnf("failed while copying SBOM from %s to %s", sbom, dstPath)
			return err
		}
	}
	return nil
}

func (e *Exporter) copyDefaultSBOMsForComponent(component, dstDir string) error {
	for _, extension := range SBOMExtensions() {
		srcFilename := fmt.Sprintf("%s.%s", component, extension)
		srcPath := filepath.Join(platform.DefaultBuildpacksioSBOMDir, srcFilename)
		if _, err := os.Stat(srcPath); err != nil {
			if os.IsNotExist(err) {
				e.Logger.Warnf("Did not find SBOM %s in %s", srcFilename, platform.DefaultBuildpacksioSBOMDir)
				continue
			} else {
				return err
			}
		}
		// create target directory if not exists
		if err := os.MkdirAll(dstDir, os.ModePerm); err != nil {
			return err
		}
		dstPath := filepath.Join(dstDir, extension)
		e.Logger.Debugf("Copying SBOM %s to %s", srcFilename, dstPath)
		if err := fsutil.Copy(srcPath, dstPath); err != nil {
			return err
		}
	}
	return nil
}

func (e *Exporter) addExtensionLayers(opts ExportOptions) error {
	if !e.PlatformAPI.AtLeast("0.12") || opts.ExtendedDir == "" {
		return nil
	}
	parentPath := filepath.Join(opts.ExtendedDir, "run")
	extendedRunImage, extendedRunImagePath, err := image.FromLayoutPath(parentPath)
	if err != nil {
		return err
	}
	if extendedRunImage == nil {
		return nil
	}
	allLayers, err := extendedRunImage.Layers()
	if err != nil {
		return err
	}
	configFile, err := extendedRunImage.ConfigFile()
	if err != nil {
		return err
	}
	history := configFile.History
	for idx, l := range allLayers {
		layerDigest, blobExists, err := getLayerInfo(l, extendedRunImagePath)
		if err != nil {
			return fmt.Errorf("failed to get layer info: %w", err)
		}
		if !blobExists {
			// the referenced blob is a run image (base image) layer, not an extension-provided layer
			continue
		}
		h := getHistoryForNonEmptyLayerAtIndex(history, idx)
		_, extID := parseHistory(h)
		withID := fmt.Sprintf("%s:extended-layer-%d", extID, idx)
		fromTarPath := filepath.Join(extendedRunImagePath, "blobs", layerDigest.Algorithm, layerDigest.Hex)
		layer, err := e.LayerFactory.TarLayer(withID, fromTarPath, h.CreatedBy)
		if err != nil {
			return fmt.Errorf("failed to create layer from tar path %s: %w", fromTarPath, err)
		}
		if _, err = e.addExtensionLayer(opts.WorkingImage, layer); err != nil {
			return fmt.Errorf("failed to add or reuse extension layer: %w", err)
		}
	}
	return nil
}

func getLayerInfo(layer v1.Layer, parentDir string) (v1.Hash, bool, error) {
	_, err := layer.DiffID()
	if err != nil {
		if _, ok := err.(*fs.PathError); ok {
			return v1.Hash{}, false, nil // the blob doesn't exist
		}
		return v1.Hash{}, false, err
	}
	layerDigest, err := layer.Digest()
	if err != nil {
		return v1.Hash{}, false, err
	}
	layerPath := filepath.Join(parentDir, "blobs", layerDigest.Algorithm, layerDigest.Hex)
	if _, err := os.Stat(layerPath); err != nil {
		if os.IsNotExist(err) {
			return v1.Hash{}, false, nil // the blob doesn't exist
		}
		return v1.Hash{}, false, err
	}
	return layerDigest, true, nil
}

func getHistoryForNonEmptyLayerAtIndex(history []v1.History, idx int) v1.History {
	var processed int
	for _, h := range history {
		if h.EmptyLayer {
			continue
		}
		if processed == idx {
			return h
		}
		processed++
	}
	return v1.History{}
}

func parseHistory(history v1.History) (string, string) {
	r := strings.NewReader(history.CreatedBy)
	var (
		createdBy, extID string
	)
	n, err := fmt.Fscanf(r, layers.ExtensionLayerName, &createdBy, &extID)
	if err != nil || n != 2 {
		return history.CreatedBy, "from extensions"
	}
	return createdBy, extID
}

func isLocalImage(workingImage imgutil.Image) bool {
	if _, ok := workingImage.(*local.Image); ok {
		return true
	}
	return false
}

func (e *Exporter) addBuildpackLayers(opts ExportOptions, meta *files.LayersMetadata) error {
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

			createdBy := fmt.Sprintf(layers.BuildpackLayerName, fsLayer.Name(), fmt.Sprintf("%s@%s", bp.ID, bp.Version))
			if fsLayer.HasLocalContents() {
				layer, err := e.LayerFactory.DirLayer(fsLayer.Identifier(), fsLayer.Path(), createdBy)
				if err != nil {
					return errors.Wrapf(err, "creating layer")
				}
				origLayerMetadata := opts.OrigMetadata.LayersMetadataFor(bp.ID).Layers[fsLayer.Name()]
				lmd.SHA, err = e.addOrReuseBuildpackLayer(opts.WorkingImage, layer, origLayerMetadata.SHA, createdBy)
				if err != nil {
					return err
				}
			} else {
				if lmd.Cache {
					return fmt.Errorf("layer '%s' is cache=true but has no contents", fsLayer.Identifier())
				}
				origLayerMetadata, ok := opts.OrigMetadata.LayersMetadataFor(bp.ID).Layers[fsLayer.Name()]
				if !ok {
					return fmt.Errorf("cannot reuse '%s', previous image has no metadata for layer '%s'", fsLayer.Identifier(), fsLayer.Identifier())
				}

				e.Logger.Infof("Reusing layer '%s'\n", fsLayer.Identifier())
				e.Logger.Debugf("Layer '%s' SHA: %s\n", fsLayer.Identifier(), origLayerMetadata.SHA)
				if err := opts.WorkingImage.ReuseLayerWithHistory(origLayerMetadata.SHA, v1.History{CreatedBy: createdBy}); err != nil {
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

func (e *Exporter) addLauncherLayers(opts ExportOptions, buildMD *files.BuildMetadata, meta *files.LayersMetadata) error {
	launcherLayer, err := e.LayerFactory.LauncherLayer(opts.LauncherConfig.Path)
	if err != nil {
		return errors.Wrap(err, "creating launcher layers")
	}
	meta.Launcher.SHA, err = e.addOrReuseBuildpackLayer(opts.WorkingImage, launcherLayer, opts.OrigMetadata.Launcher.SHA, layers.LauncherLayerName)
	if err != nil {
		return errors.Wrap(err, "exporting launcher configLayer")
	}
	configLayer, err := e.LayerFactory.DirLayer("buildpacksio/lifecycle:config", filepath.Join(opts.LayersDir, "config"), layers.LauncherConfigLayerName)
	if err != nil {
		return errors.Wrapf(err, "creating layer '%s'", configLayer.ID)
	}
	meta.Config.SHA, err = e.addOrReuseBuildpackLayer(opts.WorkingImage, configLayer, opts.OrigMetadata.Config.SHA, layers.LauncherConfigLayerName)
	if err != nil {
		return errors.Wrap(err, "exporting config layer")
	}
	if err := e.launcherConfig(opts, buildMD, meta); err != nil {
		return err
	}
	return nil
}

func (e *Exporter) addAppLayers(opts ExportOptions, slices []layers.Slice, meta *files.LayersMetadata) error {
	// creating app layers (slices + app dir)
	e.Logger.Debugf("Adding app layers")

	sliceLayers, err := e.LayerFactory.SliceLayers(opts.AppDir, slices)
	if err != nil {
		return errors.Wrap(err, "creating app layers")
	}

	var numberOfReusedLayers int
	for _, slice := range sliceLayers {
		e.Logger.Debugf("Layer '%s' SHA: %s\n", slice.ID, slice.Digest)
		var err error

		found := false
		for _, previous := range opts.OrigMetadata.App {
			if slice.Digest == previous.SHA {
				found = true
				break
			}
		}
		if found {
			err = opts.WorkingImage.ReuseLayerWithHistory(slice.Digest, slice.History)
			numberOfReusedLayers++
		} else {
			err = opts.WorkingImage.AddLayerWithDiffIDAndHistory(slice.TarPath, slice.Digest, slice.History)
		}
		if err != nil {
			return err
		}
		meta.App = append(meta.App, files.LayerMetadata{SHA: slice.Digest})
	}

	delta := len(sliceLayers) - numberOfReusedLayers
	if numberOfReusedLayers > 0 {
		e.Logger.Infof("Reused %d/%d app layer(s)\n", numberOfReusedLayers, len(sliceLayers))
	}
	if delta != 0 {
		e.Logger.Infof("Added %d/%d app layer(s)\n", delta, len(sliceLayers))
	}
	return nil
}

func (e *Exporter) setLabels(opts ExportOptions, meta files.LayersMetadata, buildMD *files.BuildMetadata) error {
	data, err := json.Marshal(meta)
	if err != nil {
		return errors.Wrap(err, "marshall metadata")
	}

	e.Logger.Infof("Adding label '%s'", platform.LifecycleMetadataLabel)
	if err = opts.WorkingImage.SetLabel(platform.LifecycleMetadataLabel, string(data)); err != nil {
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

func (e *Exporter) setEnv(opts ExportOptions) error {
	e.Logger.Debugf("Setting %s=%s", platform.EnvLayersDir, opts.LayersDir)
	if err := opts.WorkingImage.SetEnv(platform.EnvLayersDir, opts.LayersDir); err != nil {
		return errors.Wrapf(err, "set app image env %s", platform.EnvLayersDir)
	}

	e.Logger.Debugf("Setting %s=%s", platform.EnvAppDir, opts.AppDir)
	if err := opts.WorkingImage.SetEnv(platform.EnvAppDir, opts.AppDir); err != nil {
		return errors.Wrapf(err, "set app image env %s", platform.EnvAppDir)
	}

	e.Logger.Debugf("Setting %s=%s", platform.EnvPlatformAPI, e.PlatformAPI.String())
	if err := opts.WorkingImage.SetEnv(platform.EnvPlatformAPI, e.PlatformAPI.String()); err != nil {
		return errors.Wrapf(err, "set app image env %s", platform.EnvAppDir)
	}

	e.Logger.Debugf("Setting %s=%s", platform.EnvDeprecationMode, platform.ModeQuiet)
	if err := opts.WorkingImage.SetEnv(platform.EnvDeprecationMode, platform.ModeQuiet); err != nil {
		return errors.Wrapf(err, "set app image env %s", platform.EnvAppDir)
	}

	path, err := opts.WorkingImage.Env("PATH")
	if err != nil {
		return errors.Wrap(err, "failed to get PATH from app image")
	}
	path = strings.Join([]string{launch.ProcessDir, launch.LifecycleDir, path}, string(os.PathListSeparator))
	e.Logger.Debugf("Prepending %s and %s to PATH", launch.ProcessDir, launch.LifecycleDir)
	if err := opts.WorkingImage.SetEnv("PATH", path); err != nil {
		return errors.Wrap(err, "set app image env PATH")
	}
	return nil
}

func (e *Exporter) setWorkingDir(opts ExportOptions) error {
	return opts.WorkingImage.SetWorkingDir(opts.AppDir)
}

func (e *Exporter) entrypoint(launchMD launch.Metadata, userDefaultProcessType, buildpackDefaultProcessType string) (string, error) {
	if userDefaultProcessType != "" {
		defaultProcess, ok := launchMD.FindProcessType(userDefaultProcessType)
		if !ok {
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

func (e *Exporter) launcherConfig(opts ExportOptions, buildMD *files.BuildMetadata, meta *files.LayersMetadata) error {
	launchMD := launch.Metadata{
		Processes: buildMD.Processes,
	}
	if len(buildMD.Processes) > 0 {
		processTypesLayer, err := e.LayerFactory.ProcessTypesLayer(launchMD)
		if err != nil {
			return errors.Wrapf(err, "creating layer '%s'", processTypesLayer.ID)
		}
		meta.ProcessTypes.SHA, err = e.addOrReuseBuildpackLayer(opts.WorkingImage, processTypesLayer, opts.OrigMetadata.ProcessTypes.SHA, layers.ProcessTypesLayerName)
		if err != nil {
			return errors.Wrapf(err, "exporting layer '%s'", processTypesLayer.ID)
		}
	}
	return nil
}

func (e *Exporter) addOrReuseBuildpackLayer(image imgutil.Image, layer layers.Layer, previousSHA, createdBy string) (string, error) {
	layer, err := e.LayerFactory.DirLayer(layer.ID, layer.TarPath, createdBy)
	if err != nil {
		return "", errors.Wrapf(err, "creating layer '%s'", layer.ID)
	}
	if layer.Digest == previousSHA {
		e.Logger.Infof("Reusing layer '%s'\n", layer.ID)
		e.Logger.Debugf("Layer '%s' SHA: %s\n", layer.ID, layer.Digest)
		return layer.Digest, image.ReuseLayerWithHistory(previousSHA, layer.History)
	}
	e.Logger.Infof("Adding layer '%s'\n", layer.ID)
	e.Logger.Debugf("Layer '%s' SHA: %s\n", layer.ID, layer.Digest)
	return layer.Digest, image.AddLayerWithDiffIDAndHistory(layer.TarPath, layer.Digest, layer.History)
}

func (e *Exporter) addExtensionLayer(image imgutil.Image, layer layers.Layer) (string, error) {
	return layer.Digest, image.AddLayerWithDiffIDAndHistory(layer.TarPath, layer.Digest, layer.History)
}

func (e *Exporter) makeBuildReport(layersDir string) (files.BuildReport, error) {
	if e.PlatformAPI.AtLeast("0.9") {
		return files.BuildReport{}, nil
	}
	var out []buildpack.BOMEntry
	for _, bp := range e.Buildpacks {
		var bpBuildReport files.BuildReport
		bpBuildTOML := filepath.Join(layersDir, launch.EscapeID(bp.ID), "build.toml")
		if _, err := toml.DecodeFile(bpBuildTOML, &bpBuildReport); err != nil && !os.IsNotExist(err) {
			return files.BuildReport{}, err
		}
		out = append(out, buildpack.WithBuildpack(bp, bpBuildReport.BOM)...)
	}
	return files.BuildReport{BOM: out}, nil
}

func (e *Exporter) addSBOMLaunchLayer(opts ExportOptions, meta *files.LayersMetadata) error {
	sbomLaunchDir, err := readLayersSBOM(opts.LayersDir, "launch", e.Logger)
	if err != nil {
		return errors.Wrap(err, "failed to read layers config sbom")
	}

	if sbomLaunchDir != nil {
		layer, err := e.LayerFactory.DirLayer(sbomLaunchDir.Identifier(), sbomLaunchDir.Path(), layers.SBOMLayerName)
		if err != nil {
			return errors.Wrapf(err, "creating layer")
		}

		var originalSHA string
		if opts.OrigMetadata.BOM != nil {
			originalSHA = opts.OrigMetadata.BOM.SHA
		}

		sha, err := e.addOrReuseBuildpackLayer(opts.WorkingImage, layer, originalSHA, layers.SBOMLayerName)
		if err != nil {
			return errors.Wrapf(err, "exporting layer '%s'", layer.ID)
		}

		meta.BOM = &files.LayerMetadata{SHA: sha}
	}

	return nil
}
