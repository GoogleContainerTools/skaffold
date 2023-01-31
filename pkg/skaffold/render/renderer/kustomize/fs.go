package kustomize

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

var NewTmpFS = newTmpFS

type TmpFSReal struct {
	root string
}

func (f TmpFSReal) WriteTo(path string, content []byte) error {

	dst, err := f.GetPath(path)
	if err != nil {
		return err
	}
	dir := filepath.Dir(dst)

	if err := os.MkdirAll(dir, os.ModePerm); err != nil {
		return err
	}

	return os.WriteFile(dst, content, os.ModePerm)
}

func newTmpFS(rootPath string) TmpFS {
	return TmpFSReal{
		root: filepath.Clean(rootPath),
	}
}

func (f TmpFSReal) Cleanup() {
	os.RemoveAll(f.root)
}

func (f TmpFSReal) GetPath(path string) (string, error) {
	res := filepath.Join(f.root, path)
	if err := f.validate(res); err != nil {
		return "", err
	}
	return res, nil
}

func (f TmpFSReal) validate(path string) error {
	if path != f.root && !strings.HasPrefix(path, f.root+"/") {
		return fmt.Errorf("temporary file system operation is out of boundary, root: %s, trying to access %s", f.root, path)
	}
	return nil
}

type TmpFS interface {
	WriteTo(path string, content []byte) error
	Cleanup()
	GetPath(path string) (string, error)
}
