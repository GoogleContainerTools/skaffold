package index

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/buildpacks/imgutil"
	"github.com/buildpacks/imgutil/layout"
	"github.com/buildpacks/imgutil/remote"
	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/pkg/errors"
)

type IndexFactory struct {
	keychain authn.Keychain
	path     string
}

func NewIndexFactory(keychain authn.Keychain, path string) *IndexFactory {
	return &IndexFactory{
		keychain: keychain,
		path:     path,
	}
}

func (f *IndexFactory) Exists(repoName string) bool {
	return layoutImageExists(f.localPath(repoName))
}

func (f *IndexFactory) LoadIndex(repoName string, opts ...imgutil.IndexOption) (index imgutil.ImageIndex, err error) {
	if !f.Exists(repoName) {
		return nil, errors.New(fmt.Sprintf("Image: '%s' not found", repoName))
	}
	opts = appendOption(opts, imgutil.FromBaseIndex(f.localPath(repoName)))
	return layout.NewIndex(repoName, appendDefaultOptions(opts, f.keychain, f.path)...)
}

func (f *IndexFactory) FetchIndex(name string, opts ...imgutil.IndexOption) (idx imgutil.ImageIndex, err error) {
	return remote.NewIndex(name, appendDefaultOptions(opts, f.keychain, f.path)...)
}

func (f *IndexFactory) FindIndex(repoName string, opts ...imgutil.IndexOption) (idx imgutil.ImageIndex, err error) {
	if f.Exists(repoName) {
		return f.LoadIndex(repoName, opts...)
	}
	return f.FetchIndex(repoName, opts...)
}

func (f *IndexFactory) CreateIndex(repoName string, opts ...imgutil.IndexOption) (idx imgutil.ImageIndex, err error) {
	return layout.NewIndex(repoName, appendDefaultOptions(opts, f.keychain, f.path)...)
}

func (f *IndexFactory) localPath(repoName string) string {
	return filepath.Join(f.path, imgutil.MakeFileSafeName(repoName))
}

func layoutImageExists(path string) bool {
	if !pathExists(path) {
		return false
	}
	index := filepath.Join(path, "index.json")
	if _, err := os.Stat(index); os.IsNotExist(err) {
		return false
	}
	return true
}

func pathExists(path string) bool {
	if path != "" {
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			return true
		}
	}
	return false
}

func appendOption(ops []imgutil.IndexOption, op imgutil.IndexOption) []imgutil.IndexOption {
	return append(ops, op)
}

func appendDefaultOptions(ops []imgutil.IndexOption, keychain authn.Keychain, path string) []imgutil.IndexOption {
	return append(ops, imgutil.WithKeychain(keychain), imgutil.WithXDGRuntimePath(path))
}
