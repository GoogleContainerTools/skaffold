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

	"github.com/pkg/errors"
	"golang.org/x/mod/semver"
	"gopkg.in/src-d/go-git.v4"

	"github.com/buildpacks/pack/internal/buildpack"
	"github.com/buildpacks/pack/logging"
)

const DefaultRegistryURL = "https://github.com/buildpacks/registry-index"
const defaultRegistryDir = "registry"

// Cache is a RegistryCache
type Cache struct {
	logger logging.Logger
	url    *url.URL
	Root   string
}

const GithubIssueTitleTemplate = "{{ if .Yanked }}YANK{{ else }}ADD{{ end }} {{.Namespace}}/{{.Name}}@{{.Version}}"
const GithubIssueBodyTemplate = `
id = "{{.Namespace}}/{{.Name}}"
version = "{{.Version}}"
{{ if .Yanked }}{{ else if .Address }}addr = "{{.Address}}"{{ end }}
`

// Entry is a list of buildpacks stored in a registry
type Entry struct {
	Buildpacks []Buildpack `json:"buildpacks"`
}

// NewDefaultRegistryCache creates a new registry cache with default options
func NewDefaultRegistryCache(logger logging.Logger, home string) (Cache, error) {
	return NewRegistryCache(logger, home, DefaultRegistryURL)
}

// NewRegistryCache creates a new registry cache
func NewRegistryCache(logger logging.Logger, home, registryURL string) (Cache, error) {
	if _, err := os.Stat(home); err != nil {
		return Cache{}, errors.Wrapf(err, "finding home %s", home)
	}

	normalizedURL, err := url.Parse(registryURL)
	if err != nil {
		return Cache{}, errors.Wrapf(err, "parsing registry url %s", registryURL)
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

// LocateBuildpack stored in registry
func (r *Cache) LocateBuildpack(bp string) (Buildpack, error) {
	err := r.Refresh()
	if err != nil {
		return Buildpack{}, errors.Wrap(err, "refreshing cache")
	}

	ns, name, version, err := buildpack.ParseRegistryID(bp)
	if err != nil {
		return Buildpack{}, errors.Wrap(err, "parsing buildpacks registry id")
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
			return highestVersion, Validate(highestVersion)
		}

		for _, bpIndex := range entry.Buildpacks {
			if bpIndex.Version == version {
				return bpIndex, Validate(bpIndex)
			}
		}
		return Buildpack{}, fmt.Errorf("could not find version for buildpack: %s", bp)
	}

	return Buildpack{}, fmt.Errorf("no entries for buildpack: %s", bp)
}

// Refresh local Registry Cache
func (r *Cache) Refresh() error {
	r.logger.Debugf("Refreshing registry cache for %s/%s", r.url.Host, r.url.Path)

	if err := r.Initialize(); err != nil {
		return errors.Wrapf(err, "initializing (%s)", r.Root)
	}

	repository, err := git.PlainOpen(r.Root)
	if err != nil {
		return errors.Wrapf(err, "opening (%s)", r.Root)
	}

	w, err := repository.Worktree()
	if err != nil {
		return errors.Wrapf(err, "reading (%s)", r.Root)
	}

	err = w.Pull(&git.PullOptions{RemoteName: "origin"})
	if err == git.NoErrAlreadyUpToDate {
		return nil
	}
	return err
}

// Initialize a local Registry Cache
func (r *Cache) Initialize() error {
	_, err := os.Stat(r.Root)
	if err != nil {
		if os.IsNotExist(err) {
			err = r.createCache()
			if err != nil {
				return errors.Wrap(err, "creating registry cache")
			}
		}
	}

	if err := r.validateCache(); err != nil {
		err = os.RemoveAll(r.Root)
		if err != nil {
			return errors.Wrap(err, "reseting registry cache")
		}
		err = r.createCache()
		if err != nil {
			return errors.Wrap(err, "rebuilding registry cache")
		}
	}

	return nil
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
		return errors.Wrap(err, "cloning remote registry")
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
		return errors.Wrap(err, "opening registry cache")
	}

	remotes, err := repository.Remotes()
	if err != nil {
		return errors.Wrap(err, "accessing registry cache")
	}

	for _, remote := range remotes {
		if remote.Config().Name == "origin" && remotes[0].Config().URLs[0] != r.url.String() {
			return nil
		}
	}
	return errors.New("invalid registry cache remote")
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
		return Entry{}, errors.Wrapf(err, "finding buildpack: %s/%s", ns, name)
	}

	file, err := os.Open(index)
	if err != nil {
		return Entry{}, errors.Wrapf(err, "opening index for buildpack: %s/%s", ns, name)
	}
	defer file.Close()

	entry := Entry{}
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		var bp Buildpack
		err = json.Unmarshal([]byte(scanner.Text()), &bp)
		if err != nil {
			return Entry{}, errors.Wrapf(err, "parsing index for buildpack: %s/%s", ns, name)
		}

		entry.Buildpacks = append(entry.Buildpacks, bp)
	}

	if err := scanner.Err(); err != nil {
		return entry, errors.Wrapf(err, "reading index for buildpack: %s/%s", ns, name)
	}

	return entry, nil
}
