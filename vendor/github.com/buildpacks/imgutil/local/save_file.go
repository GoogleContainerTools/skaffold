package local

import (
	"archive/tar"
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/docker/docker/api/types"
	registryName "github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/pkg/errors"
	"golang.org/x/sync/errgroup"
)

func (i *Image) SaveFile() (string, error) {
	f, err := os.CreateTemp("", "imgutil.local.image.export.*.tar")
	if err != nil {
		return "", errors.Wrap(err, "failed to create temporary file")
	}
	defer func() {
		f.Close()
		if err != nil {
			os.Remove(f.Name())
		}
	}()

	// All layers need to be present here. Missing layers are either due to utilization of: (1) WithPreviousImage(),
	// or (2) FromBaseImage(). The former is only relevant if ReuseLayers() has been called which takes care of
	// resolving them. The latter case needs to be handled explicitly.
	if err := i.downloadBaseLayersOnce(); err != nil {
		return "", errors.Wrap(err, "failed to fetch base layers")
	}

	errs, _ := errgroup.WithContext(context.Background())
	pr, pw := io.Pipe()

	// File writer
	errs.Go(func() error {
		defer pr.Close()
		_, err = f.ReadFrom(pr)
		return err
	})

	// Tar producer
	errs.Go(func() error {
		defer pw.Close()

		tw := tar.NewWriter(pw)
		defer tw.Close()

		config, err := i.newConfigFile()
		if err != nil {
			return errors.Wrap(err, "failed to generate config file")
		}

		configName := fmt.Sprintf("/%x.json", sha256.Sum256(config))
		if err := addTextToTar(tw, configName, config); err != nil {
			return errors.Wrap(err, "failed to add config file to tar archive")
		}

		for _, path := range i.layerPaths {
			err := func() error {
				f, err := os.Open(filepath.Clean(path))
				if err != nil {
					return errors.Wrapf(err, "failed to open layer path: %s", path)
				}
				defer f.Close()

				layerName := fmt.Sprintf("/%x.tar", sha256.Sum256([]byte(path)))
				if err := addFileToTar(tw, layerName, f); err != nil {
					return errors.Wrapf(err, "failed to add layer to tar archive from path: %s", path)
				}

				return nil
			}()

			if err != nil {
				return err
			}
		}

		t, err := registryName.NewTag(i.repoName, registryName.WeakValidation)
		if err != nil {
			return errors.Wrap(err, "failed to create tag")
		}

		layers := make([]string, 0, len(i.layerPaths))
		for _, path := range i.layerPaths {
			layers = append(layers, fmt.Sprintf("/%x.tar", sha256.Sum256([]byte(path))))
		}

		manifest, err := json.Marshal([]map[string]interface{}{
			{
				"Config":   configName,
				"RepoTags": []string{t.Name()},
				"Layers":   layers,
			},
		})
		if err != nil {
			return errors.Wrap(err, "failed to create manifest")
		}

		if err := addTextToTar(tw, "/manifest.json", manifest); err != nil {
			return errors.Wrap(err, "failed to add manifest to tar archive")
		}

		return nil
	})

	err = errs.Wait()
	if err != nil {
		return "", err
	}

	return f.Name(), nil
}

func (i *Image) newConfigFile() ([]byte, error) {
	if !i.withHistory {
		// zero history
		i.history = make([]v1.History, len(i.inspect.RootFS.Layers))
	}
	cfg, err := v1Config(i.inspect, i.createdAt, i.history)
	if err != nil {
		return nil, err
	}
	return json.Marshal(cfg)
}

// helpers

func addFileToTar(tw *tar.Writer, name string, contents *os.File) error {
	fi, err := contents.Stat()
	if err != nil {
		return err
	}
	hdr := &tar.Header{Name: name, Mode: 0644, Size: fi.Size()}
	if err := tw.WriteHeader(hdr); err != nil {
		return err
	}
	_, err = io.Copy(tw, contents)
	return err
}

func addTextToTar(tw *tar.Writer, name string, contents []byte) error {
	hdr := &tar.Header{Name: name, Mode: 0644, Size: int64(len(contents))}
	if err := tw.WriteHeader(hdr); err != nil {
		return err
	}
	_, err := tw.Write(contents)
	return err
}

func cleanPath(dest, header string) (string, error) {
	joined := filepath.Join(dest, header)
	if strings.HasPrefix(joined, filepath.Clean(dest)) {
		return joined, nil
	}
	return "", fmt.Errorf("bad filepath: %s", header)
}

func untar(r io.Reader, dest string) error {
	tr := tar.NewReader(r)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			// end of tar archive
			return nil
		}
		if err != nil {
			return err
		}

		path, err := cleanPath(dest, hdr.Name)
		if err != nil {
			return err
		}

		switch hdr.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(path, hdr.FileInfo().Mode()); err != nil {
				return err
			}
		case tar.TypeReg:
			_, err := os.Stat(filepath.Dir(path))
			if os.IsNotExist(err) {
				if err := os.MkdirAll(filepath.Dir(path), 0750); err != nil {
					return err
				}
			}

			fh, err := os.OpenFile(filepath.Clean(path), os.O_CREATE|os.O_WRONLY, hdr.FileInfo().Mode())
			if err != nil {
				return err
			}
			if _, err := io.Copy(fh, tr); err != nil {
				fh.Close()
				return err
			} // #nosec G110
			fh.Close()
		case tar.TypeSymlink:
			_, err := os.Stat(filepath.Dir(path))
			if os.IsNotExist(err) {
				if err := os.MkdirAll(filepath.Dir(path), 0750); err != nil {
					return err
				}
			}

			if err := os.Symlink(hdr.Linkname, path); err != nil {
				return err
			}
		default:
			return fmt.Errorf("unknown file type in tar %d", hdr.Typeflag)
		}
	}
}

func v1Config(inspect types.ImageInspect, createdAt time.Time, history []v1.History) (v1.ConfigFile, error) {
	if len(history) != len(inspect.RootFS.Layers) {
		history = make([]v1.History, len(inspect.RootFS.Layers))
	}
	for i := range history {
		// zero history
		history[i].Created = v1.Time{Time: createdAt}
	}
	diffIDs := make([]v1.Hash, len(inspect.RootFS.Layers))
	for i, layer := range inspect.RootFS.Layers {
		hash, err := v1.NewHash(layer)
		if err != nil {
			return v1.ConfigFile{}, err
		}
		diffIDs[i] = hash
	}
	exposedPorts := make(map[string]struct{}, len(inspect.Config.ExposedPorts))
	for key, val := range inspect.Config.ExposedPorts {
		exposedPorts[string(key)] = val
	}
	var config v1.Config
	if inspect.Config != nil {
		var healthcheck *v1.HealthConfig
		if inspect.Config.Healthcheck != nil {
			healthcheck = &v1.HealthConfig{
				Test:        inspect.Config.Healthcheck.Test,
				Interval:    inspect.Config.Healthcheck.Interval,
				Timeout:     inspect.Config.Healthcheck.Timeout,
				StartPeriod: inspect.Config.Healthcheck.StartPeriod,
				Retries:     inspect.Config.Healthcheck.Retries,
			}
		}
		config = v1.Config{
			AttachStderr:    inspect.Config.AttachStderr,
			AttachStdin:     inspect.Config.AttachStdin,
			AttachStdout:    inspect.Config.AttachStdout,
			Cmd:             inspect.Config.Cmd,
			Healthcheck:     healthcheck,
			Domainname:      inspect.Config.Domainname,
			Entrypoint:      inspect.Config.Entrypoint,
			Env:             inspect.Config.Env,
			Hostname:        inspect.Config.Hostname,
			Image:           inspect.Config.Image,
			Labels:          inspect.Config.Labels,
			OnBuild:         inspect.Config.OnBuild,
			OpenStdin:       inspect.Config.OpenStdin,
			StdinOnce:       inspect.Config.StdinOnce,
			Tty:             inspect.Config.Tty,
			User:            inspect.Config.User,
			Volumes:         inspect.Config.Volumes,
			WorkingDir:      inspect.Config.WorkingDir,
			ExposedPorts:    exposedPorts,
			ArgsEscaped:     inspect.Config.ArgsEscaped,
			NetworkDisabled: inspect.Config.NetworkDisabled,
			MacAddress:      inspect.Config.MacAddress,
			StopSignal:      inspect.Config.StopSignal,
			Shell:           inspect.Config.Shell,
		}
	}
	return v1.ConfigFile{
		Architecture: inspect.Architecture,
		Created:      v1.Time{Time: createdAt},
		History:      history,
		OS:           inspect.Os,
		OSVersion:    inspect.OsVersion,
		RootFS: v1.RootFS{
			Type:    "layers",
			DiffIDs: diffIDs,
		},
		Config: config,
	}, nil
}
