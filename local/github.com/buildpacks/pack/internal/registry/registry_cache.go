package registry

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/pkg/errors"
	"golang.org/x/mod/semver"

	"github.com/buildpacks/pack/internal/style"
	"github.com/buildpacks/pack/pkg/buildpack"
	"github.com/buildpacks/pack/pkg/logging"
)

const DefaultRegistryURL = "https://github.com/buildpacks/registry-index"
const DefaultRegistryName = "official"
const defaultRegistryDir = "registry"

// Cache is a RegistryCache
type Cache struct {
	logger      logging.Logger
	url         *url.URL
	Root        string
	RegistryDir string
}

const GithubIssueTitleTemplate = "{{ if .Yanked }}YANK{{ else }}ADD{{ end }} {{.Namespace}}/{{.Name}}@{{.Version}}"
const GithubIssueBodyTemplate = `
id = "{{.Namespace}}/{{.Name}}"
version = "{{.Version}}"
{{ if .Yanked }}{{ else if .Address }}addr = "{{.Address}}"{{ end }}
`
const GitCommitTemplate = `{{ if .Yanked }}YANK{{else}}ADD{{end}} {{.Namespace}}/{{.Name}}@{{.Version}}`

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
			err = r.CreateCache()
			if err != nil {
				return errors.Wrap(err, "creating registry cache")
			}
		}
	}

	if err := r.validateCache(); err != nil {
		err = os.RemoveAll(r.Root)
		if err != nil {
			return errors.Wrap(err, "resetting registry cache")
		}
		err = r.CreateCache()
		if err != nil {
			return errors.Wrap(err, "rebuilding registry cache")
		}
	}

	return nil
}

// CreateCache creates the cache on the filesystem
func (r *Cache) CreateCache() error {
	var repository *git.Repository
	r.logger.Debugf("Creating registry cache for %s/%s", r.url.Host, r.url.Path)

	registryDir, err := os.MkdirTemp(filepath.Dir(r.Root), "registry")
	if err != nil {
		return err
	}

	r.RegistryDir = registryDir

	if r.url.Host == "dev.azure.com" {
		err = exec.Command("git", "clone", r.url.String(), r.RegistryDir).Run()
		if err != nil {
			return errors.Wrap(err, "cloning remote registry with native git")
		}

		repository, err = git.PlainOpen(r.RegistryDir)
		if err != nil {
			return errors.Wrap(err, "opening remote registry clone")
		}
	} else {
		repository, err = git.PlainClone(r.RegistryDir, false, &git.CloneOptions{
			URL: r.url.String(),
		})
		if err != nil {
			return errors.Wrap(err, "cloning remote registry")
		}
	}

	w, err := repository.Worktree()
	if err != nil {
		return err
	}

	err = os.Rename(w.Filesystem.Root(), r.Root)
	if err != nil {
		if err == os.ErrExist {
			// If pack is run concurrently, this action might have already occurred
			return nil
		}
		return err
	}
	return nil
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

// Commit a Buildpack change
func (r *Cache) Commit(b Buildpack, username, msg string) error {
	r.logger.Debugf("Creating commit in registry cache")

	if msg == "" {
		return errors.New("invalid commit message")
	}

	repository, err := git.PlainOpen(r.Root)
	if err != nil {
		return errors.Wrap(err, "opening registry cache")
	}

	w, err := repository.Worktree()
	if err != nil {
		return errors.Wrapf(err, "reading %s", style.Symbol(r.Root))
	}

	index, err := r.writeEntry(b)
	if err != nil {
		return errors.Wrapf(err, "writing %s", style.Symbol(index))
	}

	relativeIndexFile, err := filepath.Rel(r.Root, index)
	if err != nil {
		return errors.Wrap(err, "resolving relative path")
	}

	if _, err := w.Add(relativeIndexFile); err != nil {
		return errors.Wrapf(err, "adding %s", style.Symbol(index))
	}

	if _, err := w.Commit(msg, &git.CommitOptions{
		Author: &object.Signature{
			Name:  username,
			Email: "",
			When:  time.Now(),
		},
	}); err != nil {
		return errors.Wrapf(err, "committing")
	}

	return nil
}

func (r *Cache) writeEntry(b Buildpack) (string, error) {
	var ns = b.Namespace
	var name = b.Name

	index, err := IndexPath(r.Root, ns, name)
	if err != nil {
		return "", err
	}

	if _, err := os.Stat(index); os.IsNotExist(err) {
		if err := os.MkdirAll(filepath.Dir(index), 0750); err != nil {
			return "", errors.Wrapf(err, "creating directory structure for: %s/%s", ns, name)
		}
	} else {
		if _, err := os.Stat(index); err == nil {
			entry, err := r.readEntry(ns, name)
			if err != nil {
				return "", errors.Wrapf(err, "reading existing buildpack entries")
			}

			availableBuildpacks := entry.Buildpacks

			if len(availableBuildpacks) != 0 {
				if availableBuildpacks[len(availableBuildpacks)-1].Version == b.Version {
					return "", errors.Wrapf(err, "same version exists, upgrade the version to add")
				}
			}
		}
	}

	f, err := os.OpenFile(filepath.Clean(index), os.O_APPEND|os.O_CREATE|os.O_RDWR, 0644)
	if err != nil {
		return "", errors.Wrapf(err, "creating buildpack file: %s/%s", ns, name)
	}
	defer f.Close()

	newline := "\n"
	if runtime.GOOS == "windows" {
		newline = "\r\n"
	}

	fileContents, err := json.Marshal(b)
	if err != nil {
		return "", errors.Wrapf(err, "converting buildpack file to json: %s/%s", ns, name)
	}

	fileContentsFormatted := string(fileContents) + newline
	if _, err := f.WriteString(fileContentsFormatted); err != nil {
		return "", errors.Wrapf(err, "writing buildpack to file: %s/%s", ns, name)
	}

	return index, nil
}

func (r *Cache) readEntry(ns, name string) (Entry, error) {
	index, err := IndexPath(r.Root, ns, name)
	if err != nil {
		return Entry{}, err
	}

	if _, err := os.Stat(index); err != nil {
		return Entry{}, errors.Wrapf(err, "finding buildpack: %s/%s", ns, name)
	}

	file, err := os.Open(filepath.Clean(index))
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
