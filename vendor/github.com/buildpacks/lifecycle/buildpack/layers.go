package buildpack

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/pkg/errors"

	"github.com/buildpacks/lifecycle/api"
	"github.com/buildpacks/lifecycle/launch"
)

type LayersDir struct {
	Path      string
	layers    []Layer
	name      string
	Buildpack GroupBuildpack
	Store     *StoreTOML
}

func ReadLayersDir(layersDir string, bp GroupBuildpack, logger Logger) (LayersDir, error) {
	path := filepath.Join(layersDir, launch.EscapeID(bp.ID))
	logger.Debugf("Reading buildpack directory: %s", path)
	bpDir := LayersDir{
		name:      bp.ID,
		Path:      path,
		layers:    []Layer{},
		Buildpack: bp,
	}

	fis, err := ioutil.ReadDir(path)
	if err != nil && !os.IsNotExist(err) {
		return LayersDir{}, err
	}

	names := map[string]struct{}{}
	var tomls []string
	for _, fi := range fis {
		logger.Debugf("Reading buildpack directory item: %s", fi.Name())
		if fi.IsDir() {
			bpDir.layers = append(bpDir.layers, *bpDir.NewLayer(fi.Name(), bp.API, logger))
			names[fi.Name()] = struct{}{}
			continue
		}
		if strings.HasSuffix(fi.Name(), ".toml") {
			tomls = append(tomls, filepath.Join(path, fi.Name()))
		}
	}

	for _, tf := range tomls {
		name := strings.TrimSuffix(filepath.Base(tf), ".toml")
		if name == "store" {
			var bpStore StoreTOML
			_, err := toml.DecodeFile(tf, &bpStore)
			if err != nil {
				return LayersDir{}, errors.Wrapf(err, "failed decoding store.toml for buildpack %q", bp.ID)
			}
			bpDir.Store = &bpStore
			continue
		}
		if name == "launch" {
			// don't treat launch.toml as a layer
			continue
		}
		if name == "build" && api.MustParse(bp.API).AtLeast("0.5") {
			// if the buildpack API supports build.toml don't treat it as a layer
			continue
		}
		if _, ok := names[name]; !ok {
			bpDir.layers = append(bpDir.layers, *bpDir.NewLayer(name, bp.API, logger))
		}
	}
	sort.Slice(bpDir.layers, func(i, j int) bool {
		return bpDir.layers[i].identifier < bpDir.layers[j].identifier
	})
	return bpDir, nil
}

func (d *LayersDir) FindLayers(f func(layer Layer) bool) []Layer {
	var selectedLayers []Layer
	for _, l := range d.layers {
		if f(l) {
			selectedLayers = append(selectedLayers, l)
		}
	}
	return selectedLayers
}

func MadeLaunch(l Layer) bool {
	md, err := l.Read()
	return err == nil && md.Launch
}

func MadeCached(l Layer) bool {
	md, err := l.Read()
	return err == nil && md.Cache
}

func Malformed(l Layer) bool {
	_, err := l.Read()
	return err != nil
}

func (d *LayersDir) NewLayer(name, buildpackAPI string, logger Logger) *Layer {
	return &Layer{
		layerDir: layerDir{
			path:       filepath.Join(d.Path, name),
			identifier: fmt.Sprintf("%s:%s", d.Buildpack.ID, name),
		},
		api:    buildpackAPI,
		logger: logger,
	}
}

type Layer struct { // TODO: need to refactor so api and logger won't be part of this struct
	layerDir
	api    string
	logger Logger
}

type layerDir struct {
	identifier string
	path       string
}

func (l *Layer) Name() string {
	return filepath.Base(l.path)
}

func (l *Layer) HasLocalContents() bool {
	_, err := ioutil.ReadDir(l.path)

	return !os.IsNotExist(err)
}

func (l *Layer) Identifier() string {
	return l.identifier
}

func (l *Layer) Path() string {
	return l.path
}

func (l *Layer) Read() (LayerMetadata, error) {
	tomlPath := l.Path() + ".toml"
	layerMetadataFile, msg, err := DecodeLayerMetadataFile(tomlPath, l.api)
	if err != nil {
		return LayerMetadata{}, err
	}
	if msg != "" {
		if api.MustParse(l.api).LessThan("0.6") {
			l.logger.Warn(msg)
		} else {
			return LayerMetadata{}, errors.New(msg)
		}
	}
	var sha string
	shaBytes, err := ioutil.ReadFile(l.Path() + ".sha")
	if err != nil && !os.IsNotExist(err) { // if the sha file doesn't exist, an empty sha will be returned
		return LayerMetadata{}, err
	}
	if err == nil {
		sha = string(shaBytes)
	}
	return LayerMetadata{SHA: sha, LayerMetadataFile: layerMetadataFile}, nil
}

func (l *Layer) Remove() error {
	if err := os.RemoveAll(l.path); err != nil && !os.IsNotExist(err) {
		return err
	}
	if err := os.Remove(l.path + ".sha"); err != nil && !os.IsNotExist(err) {
		return err
	}
	if err := os.Remove(l.path + ".toml"); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

func (l *Layer) WriteMetadata(metadata LayerMetadataFile) error {
	path := l.path + ".toml"
	if err := os.MkdirAll(filepath.Dir(path), 0777); err != nil {
		return err
	}
	return EncodeLayerMetadataFile(metadata, path, l.api)
}

func (l *Layer) WriteSha(sha string) error {
	if err := ioutil.WriteFile(l.path+".sha", []byte(sha), 0666); err != nil {
		return err
	} // #nosec G306
	return nil
}
