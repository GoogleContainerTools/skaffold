package lifecycle

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
)

type bpLayersDir struct {
	path      string
	layers    []bpLayer
	name      string
	buildpack Buildpack
}

func readBuildpackLayersDir(layersDir string, buildpack Buildpack) (bpLayersDir, error) {
	path := filepath.Join(layersDir, buildpack.dir())
	bpDir := bpLayersDir{
		name:      buildpack.ID,
		path:      path,
		layers:    []bpLayer{},
		buildpack: buildpack,
	}

	fis, err := ioutil.ReadDir(path)
	if err != nil && !os.IsNotExist(err) {
		return bpLayersDir{}, err
	}

	names := map[string]struct{}{}
	for _, fi := range fis {
		if fi.IsDir() {
			bpDir.layers = append(bpDir.layers, *bpDir.newBPLayer(fi.Name()))
			names[fi.Name()] = struct{}{}
		}
	}

	tomls, err := filepath.Glob(filepath.Join(path, "*.toml"))
	if err != nil {
		return bpLayersDir{}, err
	}
	for _, toml := range tomls {
		name := strings.TrimSuffix(filepath.Base(toml), ".toml")
		if _, ok := names[name]; !ok {
			bpDir.layers = append(bpDir.layers, *bpDir.newBPLayer(name))
		}
	}
	return bpDir, nil
}

func launch(l bpLayer) bool {
	md, err := l.read()
	return err == nil && md.Launch
}

func malformed(l bpLayer) bool {
	_, err := l.read()
	return err != nil
}

func cached(l bpLayer) bool {
	md, err := l.read()
	return err == nil && md.Cache
}

func (bd *bpLayersDir) findLayers(f func(layer bpLayer) bool) []bpLayer {
	var selectedLayers []bpLayer
	for _, l := range bd.layers {
		if f(l) {
			selectedLayers = append(selectedLayers, l)
		}
	}
	return selectedLayers
}

func (bd *bpLayersDir) newBPLayer(name string) *bpLayer {
	return &bpLayer{
		layer{
			path:       filepath.Join(bd.path, name),
			identifier: fmt.Sprintf("%s:%s", bd.buildpack.ID, name),
		},
	}
}

type bpLayer struct {
	layer
}

func (bp *bpLayer) read() (BuildpackLayerMetadata, error) {
	var data BuildpackLayerMetadata
	tomlPath := bp.path + ".toml"
	fh, err := os.Open(tomlPath)
	if os.IsNotExist(err) {
		return BuildpackLayerMetadata{}, nil
	} else if err != nil {
		return BuildpackLayerMetadata{}, err
	}
	defer fh.Close()
	if _, err := toml.DecodeFile(tomlPath, &data); err != nil {
		return BuildpackLayerMetadata{}, err
	}
	sha, err := ioutil.ReadFile(bp.path + ".sha")
	if err != nil {
		if os.IsNotExist(err) {
			return data, nil
		}
		return BuildpackLayerMetadata{}, err
	}
	data.SHA = string(sha)
	return data, nil
}

func (bp *bpLayer) remove() error {
	if err := os.RemoveAll(bp.path); err != nil && !os.IsNotExist(err) {
		return err
	}
	if err := os.Remove(bp.path + ".sha"); err != nil && !os.IsNotExist(err) {
		return err
	}
	if err := os.Remove(bp.path + ".toml"); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

func (bp *bpLayer) writeMetadata(metadata BuildpackLayerMetadata) error {
	path := filepath.Join(bp.path + ".toml")
	if err := os.MkdirAll(filepath.Dir(path), 0777); err != nil {
		return err
	}
	fh, err := os.Create(path)
	if err != nil {
		return err
	}
	defer fh.Close()
	return toml.NewEncoder(fh).Encode(metadata.BuildpackLayerMetadataFile)
}

func (bp *bpLayer) hasLocalContents() bool {
	_, err := ioutil.ReadDir(bp.path)

	return !os.IsNotExist(err)
}

func (bp *bpLayer) writeSha(sha string) error {
	if err := ioutil.WriteFile(bp.path+".sha", []byte(sha), 0777); err != nil {
		return err
	}
	return nil
}

func (bp *bpLayer) name() string {
	return filepath.Base(bp.path)
}

type layer struct {
	path       string
	identifier string
}

func (l *layer) Identifier() string {
	return l.identifier
}

func (l *layer) Path() string {
	return l.path
}

type identifiableLayer interface {
	Identifier() string
	Path() string
}

func recursiveChown(path string, uid, gid int) error {
	fis, err := ioutil.ReadDir(path)
	if err != nil {
		return err
	}
	if err := os.Chown(path, uid, gid); err != nil {
		return err
	}
	for _, fi := range fis {
		filePath := filepath.Join(path, fi.Name())
		if fi.IsDir() {
			if err := recursiveChown(filePath, uid, gid); err != nil {
				return err
			}
		} else {
			if err := os.Lchown(filePath, uid, gid); err != nil {
				return err
			}
		}
	}
	return nil
}
