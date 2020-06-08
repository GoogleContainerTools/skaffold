package registry

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/url"
	"os"
	"path/filepath"

	ggcrname "github.com/google/go-containerregistry/pkg/name"
	"github.com/pkg/errors"
	"golang.org/x/mod/semver"
	"gopkg.in/src-d/go-git.v4"

	"github.com/buildpacks/pack/internal/buildpack"
	"github.com/buildpacks/pack/logging"
)

const defaultRegistryURL = "https://github.com/buildpacks/registry-index"

const defaultRegistryDir = "registry"

type Buildpack struct {
	Namespace string `json:"ns"`
	Name      string `json:"name"`
	Version   string `json:"version"`
	Yanked    bool   `json:"yanked"`
	Address   string `json:"addr"`
}

type Entry struct {
	Buildpacks []Buildpack `json:"buildpacks"`
}

type Cache struct {
	logger logging.Logger
	url    *url.URL
	Root   string
}

func NewRegistryCache(logger logging.Logger, home, registryURL string) (Cache, error) {
	if _, err := os.Stat(home); err != nil {
		return Cache{}, err
	}

	normalizedURL, err := url.Parse(registryURL)
	if err != nil {
		return Cache{}, err
	}

	key := sha256.New()
	key.Write([]byte(normalizedURL.String()))
	cacheDir := fmt.Sprintf("%s-%s", defaultRegistryDir, hex.EncodeToString(key.Sum(nil)))

	return Cache{
		url:    normalizedURL,
		logger: logger,
		Root:   filepath.Join(home, cacheDir),
	}, nil
}

func NewDefaultRegistryCache(logger logging.Logger, home string) (Cache, error) {
	return NewRegistryCache(logger, home, defaultRegistryURL)
}

func (r *Cache) createCache() error {
	r.logger.Debugf("Creating registry cache for %s/%s", r.url.Host, r.url.Path)

	root, err := ioutil.TempDir("", "registry")
	if err != nil {
		return err
	}

	repository, err := git.PlainClone(root, false, &git.CloneOptions{
		URL: r.url.String(),
	})
	if err != nil {
		return errors.Wrap(err, "could not clone remote registry")
	}

	w, err := repository.Worktree()
	if err != nil {
		return err
	}

	return os.Rename(w.Filesystem.Root(), r.Root)
}

func (r *Cache) validateCache() error {
	r.logger.Debugf("Validating registry cache for %s/%s", r.url.Host, r.url.Path)

	repository, err := git.PlainOpen(r.Root)
	if err != nil {
		return errors.Wrap(err, "could not open registry cache")
	}

	remotes, err := repository.Remotes()
	if err != nil {
		return errors.Wrap(err, "could not access registry cache")
	}

	for _, remote := range remotes {
		if remote.Config().Name == "origin" && remotes[0].Config().URLs[0] != r.url.String() {
			return nil
		}
	}
	return errors.New("invalid registry cache remote")
}

func (r *Cache) Initialize() error {
	_, err := os.Stat(r.Root)
	if err != nil {
		if os.IsNotExist(err) {
			err = r.createCache()
			if err != nil {
				return errors.Wrap(err, "could not create registry cache")
			}
		}
	}

	if err := r.validateCache(); err != nil {
		err = os.RemoveAll(r.Root)
		if err != nil {
			return errors.Wrap(err, "could not reset registry cache")
		}
		err = r.createCache()
		if err != nil {
			return errors.Wrap(err, "could not rebuild registry cache")
		}
	}

	return nil
}

func (r *Cache) Refresh() error {
	r.logger.Debugf("Refreshing registry cache for %s/%s", r.url.Host, r.url.Path)

	if err := r.Initialize(); err != nil {
		return err
	}

	repository, err := git.PlainOpen(r.Root)
	if err != nil {
		return errors.Wrapf(err, "could not open (%s)", r.Root)
	}

	w, err := repository.Worktree()
	if err != nil {
		return errors.Wrapf(err, "could not read (%s)", r.Root)
	}

	err = w.Pull(&git.PullOptions{RemoteName: "origin"})
	if err == git.NoErrAlreadyUpToDate {
		return nil
	}
	return err
}

func (r *Cache) readEntry(ns, name string) (Entry, error) {
	var indexDir string
	switch {
	case len(name) == 0:
		return Entry{}, errors.New("empty buildpack name")
	case len(name) == 1:
		indexDir = "1"
	case len(name) == 2:
		indexDir = "2"
	case len(name) == 3:
		indexDir = "3"
	default:
		indexDir = filepath.Join(name[:2], name[2:4])
	}

	index := filepath.Join(r.Root, indexDir, fmt.Sprintf("%s_%s", ns, name))

	if _, err := os.Stat(index); err != nil {
		return Entry{}, errors.Wrapf(err, "could not find buildpack: %s/%s", ns, name)
	}

	file, err := os.Open(index)
	if err != nil {
		return Entry{}, errors.Wrapf(err, "could not open index for buildpack: %s/%s", ns, name)
	}
	defer file.Close()

	entry := Entry{}
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		var bp Buildpack
		err = json.Unmarshal([]byte(scanner.Text()), &bp)
		if err != nil {
			return Entry{}, errors.Wrapf(err, "could not parse index for buildpack: %s/%s", ns, name)
		}

		entry.Buildpacks = append(entry.Buildpacks, bp)
	}

	if err := scanner.Err(); err != nil {
		return entry, errors.Wrapf(err, "could not read index for buildpack: %s/%s", ns, name)
	}

	return entry, nil
}

func (r *Cache) LocateBuildpack(bp string) (Buildpack, error) {
	err := r.Refresh()
	if err != nil {
		return Buildpack{}, errors.Wrap(err, "refreshing cache")
	}

	ns, name, version, err := buildpack.ParseRegistryID(bp)
	if err != nil {
		return Buildpack{}, err
	}

	entry, err := r.readEntry(ns, name)
	if err != nil {
		return Buildpack{}, errors.Wrap(err, "reading entry")
	}

	if len(entry.Buildpacks) > 0 {
		if version == "" {
			highestVersion := entry.Buildpacks[0]
			if len(entry.Buildpacks) > 1 {
				for _, bp := range entry.Buildpacks[1:] {
					if semver.Compare(fmt.Sprintf("v%s", bp.Version), fmt.Sprintf("v%s", highestVersion.Version)) > 0 {
						highestVersion = bp
					}
				}
			}
			return highestVersion, highestVersion.Validate()
		}

		for _, bpIndex := range entry.Buildpacks {
			if bpIndex.Version == version {
				return bpIndex, bpIndex.Validate()
			}
		}
		return Buildpack{}, fmt.Errorf("could not find version for buildpack: %s", bp)
	}

	return Buildpack{}, fmt.Errorf("no entries for buildpack: %s", bp)
}

func (b *Buildpack) Validate() error {
	if b.Address == "" {
		return errors.New("invalid entry: address is a required field")
	}
	_, err := ggcrname.NewDigest(b.Address)
	if err != nil {
		return fmt.Errorf("invalid entry: '%s' is not a digest reference", b.Address)
	}

	return nil
}
