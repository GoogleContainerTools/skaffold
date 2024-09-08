package cache

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/pkg/errors"

	"github.com/buildpacks/lifecycle/log"

	"github.com/buildpacks/lifecycle/internal/fsutil"
	"github.com/buildpacks/lifecycle/platform"
)

type VolumeCache struct {
	committed    bool
	dir          string
	backupDir    string
	stagingDir   string
	committedDir string
	logger       log.Logger
}

// NewVolumeCache creates a new VolumeCache
func NewVolumeCache(dir string, logger log.Logger) (*VolumeCache, error) {
	if _, err := os.Stat(dir); err != nil {
		return nil, err
	}

	c := &VolumeCache{
		dir:          dir,
		backupDir:    filepath.Join(dir, "committed-backup"),
		stagingDir:   filepath.Join(dir, "staging"),
		committedDir: filepath.Join(dir, "committed"),
		logger:       logger,
	}

	if err := c.setupStagingDir(); err != nil {
		return nil, errors.Wrapf(err, "initializing staging directory '%s'", c.stagingDir)
	}

	if err := os.RemoveAll(c.backupDir); err != nil {
		return nil, errors.Wrapf(err, "removing backup directory '%s'", c.backupDir)
	}

	if err := os.MkdirAll(c.committedDir, 0777); err != nil {
		return nil, errors.Wrapf(err, "creating committed directory '%s'", c.committedDir)
	}

	return c, nil
}

func (c *VolumeCache) Exists() bool {
	if _, err := os.Stat(c.committedDir); err != nil {
		return false
	}
	return true
}

func (c *VolumeCache) Name() string {
	return c.dir
}

func (c *VolumeCache) SetMetadata(metadata platform.CacheMetadata) error {
	if c.committed {
		return errCacheCommitted
	}
	metadataPath := filepath.Join(c.stagingDir, MetadataLabel)
	file, err := os.Create(metadataPath)
	if err != nil {
		return errors.Wrapf(err, "creating metadata file '%s'", metadataPath)
	}
	defer file.Close()

	if err := json.NewEncoder(file).Encode(metadata); err != nil {
		return errors.Wrap(err, "marshalling metadata")
	}

	return nil
}

func (c *VolumeCache) RetrieveMetadata() (platform.CacheMetadata, error) {
	metadataPath := filepath.Join(c.committedDir, MetadataLabel)
	file, err := os.Open(metadataPath)
	if err != nil {
		if os.IsNotExist(err) {
			return platform.CacheMetadata{}, nil
		}
		return platform.CacheMetadata{}, errors.Wrapf(err, "opening metadata file '%s'", metadataPath)
	}
	defer file.Close()

	metadata := platform.CacheMetadata{}
	if json.NewDecoder(file).Decode(&metadata) != nil {
		return platform.CacheMetadata{}, nil
	}
	return metadata, nil
}

func (c *VolumeCache) AddLayerFile(tarPath string, diffID string) error {
	if c.committed {
		return errCacheCommitted
	}
	layerTar := diffIDPath(c.stagingDir, diffID)
	if _, err := os.Stat(layerTar); err == nil {
		// don't waste time rewriting an identical layer
		return nil
	}

	if err := fsutil.Copy(tarPath, layerTar); err != nil {
		return errors.Wrapf(err, "caching layer (%s)", diffID)
	}
	return nil
}

func (c *VolumeCache) AddLayer(rc io.ReadCloser, diffID string) error {
	if c.committed {
		return errCacheCommitted
	}

	fh, err := os.Create(diffIDPath(c.stagingDir, diffID))
	if err != nil {
		return errors.Wrapf(err, "create layer file in cache")
	}
	defer fh.Close()

	if _, err := io.Copy(fh, rc); err != nil {
		return errors.Wrap(err, "copying layer to tar file")
	}
	return nil
}

func (c *VolumeCache) ReuseLayer(diffID string) error {
	if c.committed {
		return errCacheCommitted
	}
	committedPath := diffIDPath(c.committedDir, diffID)
	stagingPath := diffIDPath(c.stagingDir, diffID)

	if _, err := os.Stat(committedPath); err != nil {
		if os.IsNotExist(err) {
			return NewReadErr(fmt.Sprintf("failed to find cache layer with SHA '%s'", diffID))
		}
		if os.IsPermission(err) {
			return NewReadErr(fmt.Sprintf("failed to read cache layer with SHA '%s' due to insufficient permissions", diffID))
		}
		return fmt.Errorf("failed to re-use cache layer with SHA '%s': %w", diffID, err)
	}

	if err := os.Link(committedPath, stagingPath); err != nil && !os.IsExist(err) {
		return errors.Wrapf(err, "reusing layer (%s)", diffID)
	}
	return nil
}

func (c *VolumeCache) RetrieveLayer(diffID string) (io.ReadCloser, error) {
	path, err := c.RetrieveLayerFile(diffID)
	if err != nil {
		return nil, err
	}
	file, err := os.Open(path)
	if err != nil {
		if os.IsPermission(err) {
			return nil, NewReadErr(fmt.Sprintf("failed to read cache layer with SHA '%s' due to insufficient permissions", diffID))
		}
		if os.IsNotExist(err) {
			return nil, NewReadErr(fmt.Sprintf("failed to find cache layer with SHA '%s'", diffID))
		}
		return nil, fmt.Errorf("failed to get cache layer with SHA '%s'", diffID)
	}
	return file, nil
}

func (c *VolumeCache) HasLayer(diffID string) (bool, error) {
	if _, err := os.Stat(diffIDPath(c.committedDir, diffID)); err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, errors.Wrapf(err, "retrieving layer with SHA '%s'", diffID)
	}
	return true, nil
}

func (c *VolumeCache) RetrieveLayerFile(diffID string) (string, error) {
	path := diffIDPath(c.committedDir, diffID)
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return "", NewReadErr(fmt.Sprintf("failed to find cache layer with SHA '%s'", diffID))
		}
		return "", errors.Wrapf(err, "retrieving layer with SHA '%s'", diffID)
	}
	return path, nil
}

func (c *VolumeCache) Commit() error {
	if c.committed {
		return errCacheCommitted
	}
	c.committed = true
	if err := fsutil.RenameWithWindowsFallback(c.committedDir, c.backupDir); err != nil {
		return errors.Wrap(err, "backing up cache")
	}
	defer os.RemoveAll(c.backupDir)

	if err1 := fsutil.RenameWithWindowsFallback(c.stagingDir, c.committedDir); err1 != nil {
		if err2 := fsutil.RenameWithWindowsFallback(c.backupDir, c.committedDir); err2 != nil {
			return errors.Wrap(err2, "rolling back cache")
		}
		return errors.Wrap(err1, "committing cache")
	}

	return nil
}

func diffIDPath(basePath, diffID string) string {
	if runtime.GOOS == "windows" {
		// Avoid colons in Windows file paths
		diffID = strings.TrimPrefix(diffID, "sha256:")
	}
	return filepath.Join(basePath, diffID+".tar")
}

func (c *VolumeCache) setupStagingDir() error {
	if err := os.RemoveAll(c.stagingDir); err != nil {
		return err
	}
	return os.MkdirAll(c.stagingDir, 0777)
}
