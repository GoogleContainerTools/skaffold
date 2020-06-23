package lifecycle

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/buildpacks/imgutil"
	"github.com/pkg/errors"

	"github.com/buildpacks/lifecycle/archive"
	"github.com/buildpacks/lifecycle/cmd"
	"github.com/buildpacks/lifecycle/launch"
)

type Cache interface {
	Name() string
	SetMetadata(metadata CacheMetadata) error
	RetrieveMetadata() (CacheMetadata, error)
	AddLayerFile(tarPath string, sha string) error
	ReuseLayer(sha string) error
	RetrieveLayer(sha string) (io.ReadCloser, error)
	Commit() error
}

type Exporter struct {
	Buildpacks   []Buildpack
	ArtifactsDir string
	Logger       Logger
	UID, GID     int
	tarHashes    map[string]string // Stores hashes of layer tarballs for reuse between the export and cache steps.
}

type LauncherConfig struct {
	Path     string
	Metadata LauncherMetadata
}

type SliceLayer struct {
	ID      string
	TarPath string
	SHA     string
}

type ExportOptions struct {
	LayersDir          string
	AppDir             string
	WorkingImage       imgutil.Image
	RunImageRef        string
	OrigMetadata       LayersMetadata
	AdditionalNames    []string
	LauncherConfig     LauncherConfig
	Stack              StackMetadata
	Project            ProjectMetadata
	DefaultProcessType string
}

func (e *Exporter) Export(opts ExportOptions) error {
	var err error

	opts.LayersDir, err = filepath.Abs(opts.LayersDir)
	if err != nil {
		return errors.Wrapf(err, "layers dir absolute path")
	}

	opts.AppDir, err = filepath.Abs(opts.AppDir)
	if err != nil {
		return errors.Wrapf(err, "app dir absolute path")
	}

	meta := LayersMetadata{}
	meta.RunImage.TopLayer, err = opts.WorkingImage.TopLayer()
	if err != nil {
		return errors.Wrap(err, "get run image top layer SHA")
	}

	meta.RunImage.Reference = opts.RunImageRef
	meta.Stack = opts.Stack

	buildMD := &BuildMetadata{}
	if _, err := toml.DecodeFile(launch.GetMetadataFilePath(opts.LayersDir), buildMD); err != nil {
		return errors.Wrap(err, "read build metadata")
	}

	// creating app layers (slices + app dir)
	appSlices, err := e.createAppSliceLayers(opts.AppDir, buildMD.Slices)
	if err != nil {
		return errors.Wrap(err, "creating app layers")
	}

	// launcher
	meta.Launcher.SHA, err = e.addOrReuseLayer(opts.WorkingImage, &layer{path: opts.LauncherConfig.Path, identifier: "launcher"}, opts.OrigMetadata.Launcher.SHA)
	if err != nil {
		return errors.Wrap(err, "exporting launcher layer")
	}

	// layers
	for _, bp := range e.Buildpacks {
		bpDir, err := readBuildpackLayersDir(opts.LayersDir, bp)
		if err != nil {
			return errors.Wrapf(err, "reading layers for buildpack '%s'", bp.ID)
		}
		bpMD := BuildpackLayersMetadata{
			ID:      bp.ID,
			Version: bp.Version,
			Layers:  map[string]BuildpackLayerMetadata{},
			Store:   bpDir.store,
		}
		for _, layer := range bpDir.findLayers(forLaunch) {
			layer := layer
			lmd, err := layer.read()
			if err != nil {
				return errors.Wrapf(err, "reading '%s' metadata", layer.Identifier())
			}

			if layer.hasLocalContents() {
				origLayerMetadata := opts.OrigMetadata.MetadataForBuildpack(bp.ID).Layers[layer.name()]
				lmd.SHA, err = e.addOrReuseLayer(opts.WorkingImage, &layer, origLayerMetadata.SHA)
				if err != nil {
					return err
				}
			} else {
				if lmd.Cache {
					return fmt.Errorf("layer '%s' is cache=true but has no contents", layer.Identifier())
				}
				origLayerMetadata, ok := opts.OrigMetadata.MetadataForBuildpack(bp.ID).Layers[layer.name()]
				if !ok {
					return fmt.Errorf("cannot reuse '%s', previous image has no metadata for layer '%s'", layer.Identifier(), layer.Identifier())
				}

				e.Logger.Infof("Reusing layer '%s'\n", layer.Identifier())
				e.Logger.Debugf("Layer '%s' SHA: %s\n", layer.Identifier(), origLayerMetadata.SHA)
				if err := opts.WorkingImage.ReuseLayer(origLayerMetadata.SHA); err != nil {
					return errors.Wrapf(err, "reusing layer: '%s'", layer.Identifier())
				}
				lmd.SHA = origLayerMetadata.SHA
			}
			bpMD.Layers[layer.name()] = lmd
		}
		meta.Buildpacks = append(meta.Buildpacks, bpMD)

		if malformedLayers := bpDir.findLayers(forMalformed); len(malformedLayers) > 0 {
			ids := make([]string, 0, len(malformedLayers))
			for _, ml := range malformedLayers {
				ids = append(ids, ml.Identifier())
			}
			return fmt.Errorf("failed to parse metadata for layers '%s'", ids)
		}
	}

	// app
	meta.App, err = e.addSliceLayers(opts.WorkingImage, appSlices, opts.OrigMetadata.App)
	if err != nil {
		return errors.Wrap(err, "exporting slice layers")
	}

	// config
	meta.Config.SHA, err = e.addOrReuseLayer(opts.WorkingImage, &layer{path: filepath.Join(opts.LayersDir, "config"), identifier: "config"}, opts.OrigMetadata.Config.SHA)
	if err != nil {
		return errors.Wrap(err, "exporting config layer")
	}

	data, err := json.Marshal(meta)
	if err != nil {
		return errors.Wrap(err, "marshall metadata")
	}

	if err = opts.WorkingImage.SetLabel(LayerMetadataLabel, string(data)); err != nil {
		return errors.Wrap(err, "set app image metadata label")
	}

	buildMD.Launcher = opts.LauncherConfig.Metadata
	buildJSON, err := json.Marshal(buildMD)
	if err != nil {
		return errors.Wrap(err, "parse build metadata")
	}
	if err := opts.WorkingImage.SetLabel(BuildMetadataLabel, string(buildJSON)); err != nil {
		return errors.Wrap(err, "set build image metadata label")
	}

	projectJSON, err := json.Marshal(opts.Project)
	if err != nil {
		return errors.Wrap(err, "parse project metadata")
	}
	if err := opts.WorkingImage.SetLabel(ProjectMetadataLabel, string(projectJSON)); err != nil {
		return errors.Wrap(err, "set project metadata label")
	}

	if err = opts.WorkingImage.SetEnv(cmd.EnvLayersDir, opts.LayersDir); err != nil {
		return errors.Wrapf(err, "set app image env %s", cmd.EnvLayersDir)
	}

	if err = opts.WorkingImage.SetEnv(cmd.EnvAppDir, opts.AppDir); err != nil {
		return errors.Wrapf(err, "set app image env %s", cmd.EnvAppDir)
	}

	if opts.DefaultProcessType != "" {
		if !buildMD.hasProcess(opts.DefaultProcessType) {
			return processTypeError(buildMD, opts.DefaultProcessType)
		}

		if err = opts.WorkingImage.SetEnv(cmd.EnvProcessType, opts.DefaultProcessType); err != nil {
			return errors.Wrapf(err, "set app image env %s", cmd.EnvProcessType)
		}
	}

	if err = opts.WorkingImage.SetEntrypoint(opts.LauncherConfig.Path); err != nil {
		return errors.Wrap(err, "setting entrypoint")
	}

	if err = opts.WorkingImage.SetCmd(); err != nil { // Note: Command intentionally empty
		return errors.Wrap(err, "setting cmd")
	}

	return saveImage(opts.WorkingImage, opts.AdditionalNames, e.Logger)
}

func processTypeError(buildMD *BuildMetadata, defaultProcessType string) error {
	var typeList []string
	for _, p := range buildMD.Processes {
		typeList = append(typeList, p.Type)
	}
	return fmt.Errorf("default process type '%s' not present in list %+v", defaultProcessType, typeList)
}

func (e *Exporter) Cache(layersDir string, cacheStore Cache) error {
	var err error
	origMeta, err := cacheStore.RetrieveMetadata()
	if err != nil {
		return errors.Wrap(err, "metadata for previous cache")
	}
	meta := CacheMetadata{}

	for _, bp := range e.Buildpacks {
		bpDir, err := readBuildpackLayersDir(layersDir, bp)
		if err != nil {
			return errors.Wrapf(err, "reading layers for buildpack '%s'", bp.ID)
		}

		bpMD := BuildpackLayersMetadata{
			ID:      bp.ID,
			Version: bp.Version,
			Layers:  map[string]BuildpackLayerMetadata{},
		}
		for _, layer := range bpDir.findLayers(forCached) {
			layer := layer
			if !layer.hasLocalContents() {
				return fmt.Errorf("failed to cache layer '%s' because it has no contents", layer.Identifier())
			}
			lmd, err := layer.read()
			if err != nil {
				return errors.Wrapf(err, "reading %q metadata", layer.Identifier())
			}
			origLayerMetadata := origMeta.MetadataForBuildpack(bp.ID).Layers[layer.name()]
			if lmd.SHA, err = e.addOrReuseCacheLayer(cacheStore, &layer, origLayerMetadata.SHA); err != nil {
				return err
			}
			bpMD.Layers[layer.name()] = lmd
		}
		meta.Buildpacks = append(meta.Buildpacks, bpMD)
	}

	if err := cacheStore.SetMetadata(meta); err != nil {
		return errors.Wrap(err, "setting cache metadata")
	}
	if err := cacheStore.Commit(); err != nil {
		return errors.Wrap(err, "committing cache")
	}

	return nil
}

func (e *Exporter) tarLayer(layer identifiableLayer) (string, string, error) {
	tarPath := filepath.Join(e.ArtifactsDir, launch.EscapeID(layer.Identifier())+".tar")
	if e.tarHashes == nil {
		e.tarHashes = make(map[string]string)
	}
	if sha, ok := e.tarHashes[tarPath]; ok {
		e.Logger.Debugf("Reusing tarball for layer %q with SHA: %s\n", layer.Identifier(), sha)
		return tarPath, sha, nil
	}
	e.Logger.Debugf("Writing tarball for layer %q\n", layer.Identifier())
	sha, err := archive.WriteTarFile(layer.Path(), tarPath, e.UID, e.GID)
	if err != nil {
		return "", "", err
	}
	e.tarHashes[tarPath] = sha
	return tarPath, sha, nil
}

func (e *Exporter) addOrReuseLayer(image imgutil.Image, layer identifiableLayer, previousSHA string) (string, error) {
	tarPath, sha, err := e.tarLayer(layer)
	if err != nil {
		return "", errors.Wrapf(err, "tarring layer '%s'", layer.Identifier())
	}
	if sha == previousSHA {
		e.Logger.Infof("Reusing layer '%s'\n", layer.Identifier())
		e.Logger.Debugf("Layer '%s' SHA: %s\n", layer.Identifier(), sha)
		return sha, image.ReuseLayer(previousSHA)
	}
	e.Logger.Infof("Adding layer '%s'\n", layer.Identifier())
	e.Logger.Debugf("Layer '%s' SHA: %s\n", layer.Identifier(), sha)
	return sha, image.AddLayerWithDiffID(tarPath, sha)
}

func (e *Exporter) addOrReuseCacheLayer(cache Cache, layer identifiableLayer, previousSHA string) (string, error) {
	tarPath, sha, err := e.tarLayer(layer)
	if err != nil {
		return "", errors.Wrapf(err, "tarring layer %q", layer.Identifier())
	}
	if sha == previousSHA {
		e.Logger.Infof("Reusing cache layer '%s'\n", layer.Identifier())
		e.Logger.Debugf("Layer '%s' SHA: %s\n", layer.Identifier(), sha)
		return sha, cache.ReuseLayer(previousSHA)
	}
	e.Logger.Infof("Adding cache layer '%s'\n", layer.Identifier())
	e.Logger.Debugf("Layer '%s' SHA: %s\n", layer.Identifier(), sha)
	return sha, cache.AddLayerFile(tarPath, sha)
}

func (e *Exporter) createAppSliceLayers(appDir string, slices []Slice) ([]SliceLayer, error) {
	var appSlices []SliceLayer

	for index, slice := range slices {
		var allGlobMatches []string
		for _, path := range slice.Paths {
			globMatches, err := filepath.Glob(e.toAbs(appDir, path))
			if err != nil {
				return nil, errors.Wrap(err, "bad pattern for glob path")
			}
			allGlobMatches = append(allGlobMatches, globMatches...)
		}
		sliceLayerID := fmt.Sprintf("slice-%d", index+1)
		sliceLayer, err := e.createSliceLayer(appDir, sliceLayerID, allGlobMatches)
		if err != nil {
			return nil, errors.Wrap(err, "creating slice layer")
		}
		appSlices = append(appSlices, sliceLayer)
	}

	// finish-up by creating the actual app dir layer and place it at the end of the app slices
	// -------------
	// |  slice 1  |
	// -------------
	// |  slice 2  |
	// -------------
	// |  slice N  |
	// -------------
	// |  app dir  |
	// -------------
	tarPath := filepath.Join(e.ArtifactsDir, "app.tar")
	sha, err := archive.WriteTarFile(appDir, tarPath, e.UID, e.GID)
	if err != nil {
		return nil, errors.Wrapf(err, "exporting layer 'app'")
	}

	return append(appSlices, SliceLayer{
		ID:      "app",
		SHA:     sha,
		TarPath: tarPath,
	}), nil
}

func (e *Exporter) createSliceLayer(appDir, layerID string, files []string) (SliceLayer, error) {
	tarPath := filepath.Join(e.ArtifactsDir, launch.EscapeID(layerID)+".tar")
	sha, fileSet, err := archive.WriteFilesToTar(tarPath, e.UID, e.GID, files...)
	if err != nil {
		return SliceLayer{}, errors.Wrapf(err, "exporting slice layer '%s'", layerID)
	}

	// for this first iteration, just delete the actual files then revisit
	// the directories and delete if empty as a result of previous removal
	var dirs []string
	for file := range fileSet {
		stat, _ := os.Stat(file)
		if !stat.IsDir() {
			err = os.Remove(file)
			if err != nil {
				e.Logger.Errorf("failed to delete file %v", err)
			}
		} else {
			if file == appDir {
				continue
			}
			dirs = append(dirs, file)
		}
	}
	// sort the dirs by their path depth (deepest -> most shallow) to avoid NOT being able to delete a high level dir
	// that nested empty dirs.
	sort.SliceStable(dirs, func(i, j int) bool {
		return len(strings.Split(dirs[i], string(os.PathSeparator))) > len(strings.Split(dirs[j], string(os.PathSeparator)))
	})
	for _, dir := range dirs {
		if ok, err := isEmptyDir(dir); ok {
			if err != nil {
				e.Logger.Errorf("failed to check if directory is empty %v", err)
			}
			err = os.Remove(dir)
			if err != nil {
				e.Logger.Errorf("failed to delete directory %v", err)
			}
		}
	}

	return SliceLayer{
		ID:      layerID,
		SHA:     sha,
		TarPath: tarPath,
	}, nil
}

func (e *Exporter) addSliceLayers(image imgutil.Image, sliceLayers []SliceLayer, previousAppMD []LayerMetadata) ([]LayerMetadata, error) {
	var numberOfReusedLayers int
	var appMD []LayerMetadata

	for _, slice := range sliceLayers {
		var err error

		found := false
		for _, previous := range previousAppMD {
			if slice.SHA == previous.SHA {
				found = true
				break
			}
		}
		if found {
			err = image.ReuseLayer(slice.SHA)
			numberOfReusedLayers++
		} else {
			err = image.AddLayerWithDiffID(slice.TarPath, slice.SHA)
		}
		if err != nil {
			return nil, err
		}
		e.Logger.Debugf("Layer '%s' SHA: %s\n", slice.ID, slice.SHA)
		appMD = append(appMD, LayerMetadata{SHA: slice.SHA})
	}

	delta := len(sliceLayers) - numberOfReusedLayers
	if numberOfReusedLayers > 0 {
		e.Logger.Infof("Reusing %d/%d app layer(s)\n", numberOfReusedLayers, len(sliceLayers))
	}
	if delta != 0 {
		e.Logger.Infof("Adding %d/%d app layer(s)\n", delta, len(sliceLayers))
	}

	return appMD, nil
}

func (e *Exporter) toAbs(baseDir, path string) string {
	path = filepath.Clean(path)

	// force relative path to be absolute from the base dir
	if !filepath.IsAbs(path) {
		path = filepath.Join(baseDir, path)
	}
	// force an absolute path to be absolute from base dir
	if len(path) > len(baseDir) && path[:len(baseDir)] != baseDir {
		path = filepath.Join(baseDir, path)
		e.Logger.Warnf("found absolute path %s outside of %s", path, baseDir)
	}

	return path
}
