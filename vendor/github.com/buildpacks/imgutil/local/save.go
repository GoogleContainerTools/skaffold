package local

import (
	"archive/tar"
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/jsonmessage"
	registryName "github.com/google/go-containerregistry/pkg/name"
	"github.com/pkg/errors"

	"github.com/buildpacks/imgutil"
)

func (i *Image) Save(additionalNames ...string) error {
	return i.SaveAs(i.Name(), additionalNames...)
}

func (i *Image) SaveAs(name string, additionalNames ...string) error {
	// during the first save attempt some layers may be excluded. The docker daemon allows this if the given set
	// of layers already exists in the daemon in the given order
	inspect, err := i.doSaveAs(name)
	if err != nil {
		// populate all layer paths and try again without the above performance optimization.
		if err := i.downloadBaseLayersOnce(); err != nil {
			return err
		}

		inspect, err = i.doSaveAs(name)
		if err != nil {
			saveErr := imgutil.SaveError{}
			for _, n := range append([]string{name}, additionalNames...) {
				saveErr.Errors = append(saveErr.Errors, imgutil.SaveDiagnostic{ImageName: n, Cause: err})
			}
			return saveErr
		}
	}
	i.inspect = inspect

	var errs []imgutil.SaveDiagnostic
	for _, n := range append([]string{name}, additionalNames...) {
		if err := i.docker.ImageTag(context.Background(), i.inspect.ID, n); err != nil {
			errs = append(errs, imgutil.SaveDiagnostic{ImageName: n, Cause: err})
		}
	}

	if len(errs) > 0 {
		return imgutil.SaveError{Errors: errs}
	}

	return nil
}

func (i *Image) doSaveAs(name string) (types.ImageInspect, error) {
	ctx := context.Background()
	done := make(chan error)

	t, err := registryName.NewTag(name, registryName.WeakValidation)
	if err != nil {
		return types.ImageInspect{}, err
	}

	// returns valid 'name:tag' appending 'latest', if missing tag
	repoName := t.Name()

	pr, pw := io.Pipe()
	defer pw.Close()
	go func() {
		res, err := i.docker.ImageLoad(ctx, pr, true)
		if err != nil {
			done <- err
			return
		}

		// only return response error after response is drained and closed
		responseErr := checkResponseError(res.Body)
		drainCloseErr := ensureReaderClosed(res.Body)
		if responseErr != nil {
			done <- responseErr
			return
		}
		if drainCloseErr != nil {
			done <- drainCloseErr
		}

		done <- nil
	}()

	tw := tar.NewWriter(pw)
	defer tw.Close()

	configFile, err := i.newConfigFile()
	if err != nil {
		return types.ImageInspect{}, errors.Wrap(err, "generating config file")
	}

	id := fmt.Sprintf("%x", sha256.Sum256(configFile))
	if err := addTextToTar(tw, id+".json", configFile); err != nil {
		return types.ImageInspect{}, err
	}

	var blankIdx int
	var layerPaths []string
	for _, path := range i.layerPaths {
		if path == "" {
			layerName := fmt.Sprintf("blank_%d", blankIdx)
			blankIdx++
			hdr := &tar.Header{Name: layerName, Mode: 0644, Size: 0}
			if err := tw.WriteHeader(hdr); err != nil {
				return types.ImageInspect{}, err
			}
			layerPaths = append(layerPaths, layerName)
		} else {
			layerName := fmt.Sprintf("/%x.tar", sha256.Sum256([]byte(path)))
			f, err := os.Open(filepath.Clean(path))
			if err != nil {
				return types.ImageInspect{}, err
			}
			defer f.Close()
			if err := addFileToTar(tw, layerName, f); err != nil {
				return types.ImageInspect{}, err
			}
			f.Close()
			layerPaths = append(layerPaths, layerName)
		}
	}

	manifest, err := json.Marshal([]map[string]interface{}{
		{
			"Config":   id + ".json",
			"RepoTags": []string{repoName},
			"Layers":   layerPaths,
		},
	})
	if err != nil {
		return types.ImageInspect{}, err
	}

	if err := addTextToTar(tw, "manifest.json", manifest); err != nil {
		return types.ImageInspect{}, err
	}

	tw.Close()
	pw.Close()
	err = <-done
	if err != nil {
		return types.ImageInspect{}, errors.Wrapf(err, "loading image %q. first error", i.repoName)
	}

	inspect, _, err := i.docker.ImageInspectWithRaw(context.Background(), id)
	if err != nil {
		if client.IsErrNotFound(err) {
			return types.ImageInspect{}, errors.Wrapf(err, "saving image %q", i.repoName)
		}
		return types.ImageInspect{}, err
	}

	return inspect, nil
}

// downloadBaseLayersOnce exports the base image from the daemon and populates layerPaths the first time it is called.
// subsequent calls do nothing.
func (i *Image) downloadBaseLayersOnce() error {
	var err error
	if !i.Found() {
		return nil
	}
	i.downloadBaseOnce.Do(func() {
		err = i.downloadBaseLayers()
	})
	if err != nil {
		return errors.Wrap(err, "fetching base layers")
	}
	return err
}

func (i *Image) downloadBaseLayers() error {
	ctx := context.Background()

	imageReader, err := i.docker.ImageSave(ctx, []string{i.inspect.ID})
	if err != nil {
		return errors.Wrapf(err, "saving base image with ID %q from the docker daemon", i.inspect.ID)
	}
	defer ensureReaderClosed(imageReader)

	tmpDir, err := ioutil.TempDir("", "imgutil.local.image.")
	if err != nil {
		return errors.Wrap(err, "failed to create temp dir")
	}

	err = untar(imageReader, tmpDir)
	if err != nil {
		return err
	}

	mf, err := os.Open(filepath.Clean(filepath.Join(tmpDir, "manifest.json")))
	if err != nil {
		return err
	}
	defer mf.Close()

	var manifest []struct {
		Config string
		Layers []string
	}
	if err := json.NewDecoder(mf).Decode(&manifest); err != nil {
		return err
	}

	if len(manifest) != 1 {
		return fmt.Errorf("manifest.json had unexpected number of entries: %d", len(manifest))
	}

	df, err := os.Open(filepath.Clean(filepath.Join(tmpDir, manifest[0].Config)))
	if err != nil {
		return err
	}
	defer df.Close()

	var details struct {
		RootFS struct {
			DiffIDs []string `json:"diff_ids"`
		} `json:"rootfs"`
	}

	if err = json.NewDecoder(df).Decode(&details); err != nil {
		return err
	}

	for l := range details.RootFS.DiffIDs {
		i.layerPaths[l] = filepath.Join(tmpDir, manifest[0].Layers[l])
	}

	for l := range i.layerPaths {
		if i.layerPaths[l] == "" {
			return errors.New("failed to download all base layers from daemon")
		}
	}

	return nil
}

// helpers

func checkResponseError(r io.Reader) error {
	decoder := json.NewDecoder(r)
	var jsonMessage jsonmessage.JSONMessage
	if err := decoder.Decode(&jsonMessage); err != nil {
		return errors.Wrapf(err, "parsing daemon response")
	}

	if jsonMessage.Error != nil {
		return errors.Wrap(jsonMessage.Error, "embedded daemon response")
	}
	return nil
}

// ensureReaderClosed drains and closes and reader, returning the first error
func ensureReaderClosed(r io.ReadCloser) error {
	_, err := io.Copy(ioutil.Discard, r)
	if closeErr := r.Close(); closeErr != nil && err == nil {
		err = closeErr
	}
	return err
}
