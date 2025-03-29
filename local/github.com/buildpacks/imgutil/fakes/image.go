package fakes

import (
	"archive/tar"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	registryName "github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/types"
	"github.com/pkg/errors"

	"github.com/buildpacks/imgutil"
)

var _ imgutil.Image = &Image{}

func NewImage(name, topLayerSha string, identifier imgutil.Identifier) *Image {
	return &Image{
		labels:           nil,
		env:              map[string]string{},
		topLayerSha:      topLayerSha,
		identifier:       identifier,
		name:             name,
		cmd:              []string{"initialCMD"},
		layersMap:        map[string]string{},
		prevLayersMap:    map[string]string{},
		createdAt:        time.Now(),
		savedNames:       map[string]bool{},
		os:               "linux",
		osVersion:        "",
		architecture:     "amd64",
		savedAnnotations: map[string]string{},
	}
}

type Image struct {
	deleted          bool
	layers           []string
	history          []v1.History
	layersMap        map[string]string
	prevLayersMap    map[string]string
	reusedLayers     []string
	labels           map[string]string
	env              map[string]string
	topLayerSha      string
	os               string
	osVersion        string
	architecture     string
	variant          string
	identifier       imgutil.Identifier
	name             string
	entryPoint       []string
	cmd              []string
	base             string
	createdAt        time.Time
	layerDir         string
	workingDir       string
	savedNames       map[string]bool
	manifestSize     int64
	refName          string
	savedAnnotations map[string]string
}

func (i *Image) CreatedAt() (time.Time, error) {
	return i.createdAt, nil
}

func (i *Image) History() ([]v1.History, error) {
	return i.history, nil
}

func (i *Image) Label(key string) (string, error) {
	return i.labels[key], nil
}

func (i *Image) Labels() (map[string]string, error) {
	copiedLabels := make(map[string]string)
	for i, l := range i.labels {
		copiedLabels[i] = l
	}
	return copiedLabels, nil
}

func (i *Image) OS() (string, error) {
	return i.os, nil
}

func (i *Image) OSVersion() (string, error) {
	return i.osVersion, nil
}

func (i *Image) Architecture() (string, error) {
	return i.architecture, nil
}

func (i *Image) Variant() (string, error) {
	return i.variant, nil
}

func (i *Image) Features() ([]string, error) {
	return nil, nil
}

func (i *Image) OSFeatures() ([]string, error) {
	return nil, nil
}

func (i *Image) Annotations() (map[string]string, error) {
	return nil, nil
}

func (i *Image) Rename(name string) {
	i.name = name
}

func (i *Image) Name() string {
	return i.name
}

func (i *Image) Identifier() (imgutil.Identifier, error) {
	return i.identifier, nil
}

func (i *Image) Digest() (v1.Hash, error) {
	return v1.Hash{}, nil
}

func (i *Image) MediaType() (types.MediaType, error) {
	return types.MediaType(""), nil
}

func (i *Image) Kind() string {
	return ""
}

func (i *Image) UnderlyingImage() v1.Image {
	return nil
}

func (i *Image) Rebase(_ string, newBase imgutil.Image) error {
	i.base = newBase.Name()
	return nil
}

func (i *Image) SetLabel(k string, v string) error {
	if i.labels == nil {
		i.labels = map[string]string{}
	}
	i.labels[k] = v
	return nil
}

func (i *Image) RemoveLabel(key string) error {
	delete(i.labels, key)
	return nil
}

func (i *Image) SetEnv(k string, v string) error {
	i.env[k] = v
	return nil
}

func (i *Image) SetHistory(history []v1.History) error {
	i.history = history
	return nil
}

func (i *Image) SetOS(o string) error {
	i.os = o
	return nil
}

func (i *Image) SetOSVersion(v string) error {
	i.osVersion = v
	return nil
}

func (i *Image) SetArchitecture(a string) error {
	i.architecture = a
	return nil
}

func (i *Image) SetVariant(a string) error {
	i.variant = a
	return nil
}

func (i *Image) SetFeatures(_ []string) error {
	return nil
}

func (i *Image) SetOSFeatures(_ []string) error {
	return nil
}

func (i *Image) SetAnnotations(_ map[string]string) error {
	return nil
}

func (i *Image) SetWorkingDir(dir string) error {
	i.workingDir = dir
	return nil
}

func (i *Image) SetEntrypoint(v ...string) error {
	i.entryPoint = v
	return nil
}

func (i *Image) SetCmd(v ...string) error {
	i.cmd = v
	return nil
}

func (i *Image) SetCreatedAt(t time.Time) error {
	i.createdAt = t
	return nil
}

func (i *Image) Env(k string) (string, error) {
	return i.env[k], nil
}

func (i *Image) TopLayer() (string, error) {
	return i.topLayerSha, nil
}

func (i *Image) AddLayer(path string) error {
	sha, err := shaForFile(path)
	if err != nil {
		return err
	}

	i.layersMap["sha256:"+sha] = path
	i.layers = append(i.layers, path)
	i.history = append(i.history, v1.History{})
	return nil
}

func (i *Image) AddLayerWithDiffID(path string, diffID string) error {
	i.layersMap[diffID] = path
	i.layers = append(i.layers, path)
	i.history = append(i.history, v1.History{})
	return nil
}

func (i *Image) AddLayerWithDiffIDAndHistory(path, diffID string, history v1.History) error {
	i.layersMap[diffID] = path
	i.layers = append(i.layers, path)
	i.history = append(i.history, history)
	return nil
}

func shaForFile(path string) (string, error) {
	rc, err := os.Open(filepath.Clean(path))
	if err != nil {
		return "", errors.Wrapf(err, "failed to open file")
	}
	defer rc.Close()

	hasher := sha256.New()
	if _, err := io.Copy(hasher, rc); err != nil {
		return "", errors.Wrapf(err, "failed to copy rc to hasher")
	}

	return hex.EncodeToString(hasher.Sum(make([]byte, 0, hasher.Size()))), nil
}

func (i *Image) AddOrReuseLayerWithHistory(_, _ string, _ v1.History) error {
	panic("implement me")
}

func (i *Image) GetLayer(sha string) (io.ReadCloser, error) {
	path, ok := i.layersMap[sha]
	if !ok {
		return nil, fmt.Errorf("failed to get layer with sha '%s'", sha)
	}

	return os.Open(filepath.Clean(path))
}

func (i *Image) ReuseLayer(sha string) error {
	prevLayer, ok := i.prevLayersMap[sha]
	if !ok {
		return fmt.Errorf("image does not have previous layer with sha '%s'", sha)
	}
	i.reusedLayers = append(i.reusedLayers, sha)
	i.layersMap[sha] = prevLayer
	return nil
}

func (i *Image) ReuseLayerWithHistory(sha string, history v1.History) error {
	if err := i.ReuseLayer(sha); err != nil {
		return err
	}
	i.history = append(i.history, history)
	return nil
}

func (i *Image) Save(additionalNames ...string) error {
	return i.SaveAs(i.Name(), additionalNames...)
}

func (i *Image) SaveAs(name string, additionalNames ...string) error {
	var err error
	i.layerDir, err = os.MkdirTemp("", "fake-image")
	if err != nil {
		return err
	}

	for sha, path := range i.layersMap {
		newPath := filepath.Join(i.layerDir, filepath.Base(path))
		i.copyLayer(path, newPath) // errcheck ignore
		i.layersMap[sha] = newPath
	}

	for l := range i.layers {
		layerPath := i.layers[l]
		i.layers[l] = filepath.Join(i.layerDir, filepath.Base(layerPath))
	}

	allNames := append([]string{name}, additionalNames...)
	if i.refName != "" {
		i.savedAnnotations["org.opencontainers.image.ref.name"] = i.refName
	}

	var errs []imgutil.SaveDiagnostic
	for _, n := range allNames {
		_, err := registryName.ParseReference(n, registryName.WeakValidation)
		if err != nil {
			errs = append(errs, imgutil.SaveDiagnostic{ImageName: n, Cause: err})
		} else {
			i.savedNames[n] = true
		}
	}

	if len(errs) > 0 {
		return imgutil.SaveError{Errors: errs}
	}

	return nil
}

func (i *Image) SaveFile() (string, error) {
	return "", errors.New("not yet implemented")
}

func (i *Image) copyLayer(path, newPath string) error {
	src, err := os.Open(filepath.Clean(path))
	if err != nil {
		return errors.Wrap(err, "opening layer during copy")
	}
	defer src.Close()

	dst, err := os.Create(newPath)
	if err != nil {
		return errors.Wrap(err, "creating new layer during copy")
	}
	defer dst.Close()

	if _, err := io.Copy(dst, src); err != nil {
		return errors.Wrap(err, "copying layers")
	}

	return nil
}

func (i *Image) Delete() error {
	i.deleted = true
	return nil
}

func (i *Image) Found() bool {
	return !i.deleted
}

func (i *Image) Valid() bool {
	return !i.deleted
}

func (i *Image) AnnotateRefName(refName string) error {
	i.refName = refName
	return nil
}

func (i *Image) GetAnnotateRefName() (string, error) {
	return i.refName, nil
}

// test methods

func (i *Image) SetIdentifier(identifier imgutil.Identifier) {
	i.identifier = identifier
}

func (i *Image) Cleanup() error {
	return os.RemoveAll(i.layerDir)
}

func (i *Image) AppLayerPath() string {
	return i.layers[0]
}

func (i *Image) Entrypoint() ([]string, error) {
	return i.entryPoint, nil
}

func (i *Image) Cmd() ([]string, error) {
	return i.cmd, nil
}

func (i *Image) ConfigLayerPath() string {
	return i.layers[1]
}

func (i *Image) ReusedLayers() []string {
	return i.reusedLayers
}

func (i *Image) WorkingDir() (string, error) {
	return i.workingDir, nil
}

func (i *Image) AddPreviousLayer(sha, path string) {
	i.prevLayersMap[sha] = path
}

func (i *Image) FindLayerWithPath(path string) (string, error) {
	// we iterate backwards over the layer array b/c later layers could replace a file with a given path
	for idx := len(i.layers) - 1; idx >= 0; idx-- {
		tarPath := i.layers[idx]
		rc, err := os.Open(filepath.Clean(tarPath))
		if err != nil {
			return "", errors.Wrapf(err, "opening layer file '%s'", tarPath)
		}
		defer rc.Close()

		tr := tar.NewReader(rc)
		for {
			header, err := tr.Next()
			if err == io.EOF {
				break
			} else if err != nil {
				return "", errors.Wrap(err, "finding next header in layer")
			}

			if header.Name == path {
				return tarPath, nil
			}
		}
	}
	return "", fmt.Errorf("could not find '%s' in any layer.\n\n%s", path, i.tarContents())
}

func (i *Image) tarContents() string {
	var strBuilder = &strings.Builder{}
	strBuilder.WriteString("Layers\n-------\n")
	for idx, tarPath := range i.layers {
		i.writeLayerContents(strBuilder, tarPath)
		if idx < len(i.layers)-1 {
			strBuilder.WriteString("\n")
		}
	}
	return strBuilder.String()
}

func (i *Image) writeLayerContents(strBuilder *strings.Builder, tarPath string) {
	strBuilder.WriteString(fmt.Sprintf("%s\n", filepath.Base(tarPath)))

	rc, err := os.Open(filepath.Clean(tarPath))
	if err != nil {
		strBuilder.WriteString(fmt.Sprintf("Error reading layer files: %s\n", err))
		return
	}
	defer rc.Close()

	tr := tar.NewReader(rc)

	hasFiles := false
	for {
		header, err := tr.Next()
		if err == io.EOF {
			if !hasFiles {
				strBuilder.WriteString("  (empty)\n")
			}
			break
		}

		var typ = "F"
		var extra = ""
		switch header.Typeflag {
		case tar.TypeDir:
			typ = "D"
		case tar.TypeSymlink:
			typ = "S"
			extra = fmt.Sprintf(" -> %s", header.Linkname)
		}

		strBuilder.WriteString(fmt.Sprintf("  - [%s] %s%s\n", typ, header.Name, extra))
		hasFiles = true
	}
}

func (i *Image) NumberOfAddedLayers() int {
	return len(i.layers)
}

func (i *Image) IsSaved() bool {
	return len(i.savedNames) > 0
}

func (i *Image) Base() string {
	return i.base
}

func (i *Image) SavedNames() []string {
	var names []string
	for k := range i.savedNames {
		names = append(names, k)
	}

	return names
}

func (i *Image) SetManifestSize(size int64) {
	i.manifestSize = size
}

func (i *Image) ManifestSize() (int64, error) {
	return i.manifestSize, nil
}

func (i *Image) SavedAnnotations() map[string]string {
	return i.savedAnnotations
}
