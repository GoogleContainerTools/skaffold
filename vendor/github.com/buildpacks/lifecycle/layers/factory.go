package layers

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/buildpacks/lifecycle/archive"
)

type Factory struct {
	ArtifactsDir string // ArtifactsDir is the directory where layer files are written
	UID, GID     int    // UID and GID are used to normalize layer entries
	Logger       Logger

	tarHashes map[string]string // tarHases Stores hashes of layer tarballs for reuse between the export and cache steps.
}

type Layer struct {
	ID      string
	TarPath string
	Digest  string
}

type Logger interface {
	Debug(msg string)
	Debugf(fmt string, v ...interface{})

	Info(msg string)
	Infof(fmt string, v ...interface{})

	Warn(msg string)
	Warnf(fmt string, v ...interface{})

	Error(msg string)
	Errorf(fmt string, v ...interface{})
}

func (f *Factory) writeLayer(id string, addEntries func(tw *archive.NormalizingTarWriter) error) (layer Layer, err error) {
	tarPath := filepath.Join(f.ArtifactsDir, escape(id)+".tar")
	if f.tarHashes == nil {
		f.tarHashes = make(map[string]string)
	}
	if sha, ok := f.tarHashes[tarPath]; ok {
		f.Logger.Debugf("Reusing tarball for layer %q with SHA: %s\n", id, sha)
		return Layer{
			ID:      id,
			TarPath: tarPath,
			Digest:  sha,
		}, nil
	}
	lw, err := newFileLayerWriter(tarPath)
	if err != nil {
		return Layer{}, err
	}
	defer func() {
		if closeErr := lw.Close(); err == nil {
			err = closeErr
		}
	}()
	tw := tarWriter(lw)
	if err := addEntries(tw); err != nil {
		return Layer{}, err
	}

	if err := tw.Close(); err != nil {
		return Layer{}, err
	}
	digest := lw.Digest()
	f.tarHashes[tarPath] = digest
	return Layer{
		ID:      id,
		Digest:  digest,
		TarPath: tarPath,
	}, err
}

func escape(id string) string {
	return strings.ReplaceAll(id, "/", "_")
}

func parents(file string) ([]archive.PathInfo, error) {
	parent := filepath.Dir(file)
	if parent == filepath.VolumeName(file)+`\` || parent == "/" {
		return []archive.PathInfo{}, nil
	}
	fi, err := os.Stat(parent)
	if err != nil {
		return nil, err
	}
	parentDirs, err := parents(parent)
	if err != nil {
		return nil, err
	}
	return append(parentDirs, archive.PathInfo{
		Path: parent,
		Info: fi,
	}), nil
}
