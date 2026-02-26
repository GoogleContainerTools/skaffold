//go:build !windows

package dotgit

import (
	"github.com/go-git/go-billy/v5"
	"github.com/go-git/go-git/v5/utils/trace"
)

func fixPermissions(fs billy.Filesystem, path string) {
	if chmodFS, ok := fs.(billy.Chmod); ok {
		if err := chmodFS.Chmod(path, 0o444); err != nil {
			trace.General.Printf("failed to chmod %s: %v", path, err)
		}
	}
}

func isReadOnly(fs billy.Filesystem, path string) (bool, error) {
	fi, err := fs.Stat(path)
	if err != nil {
		return false, err
	}

	if fi.Mode().Perm() == 0o444 {
		return true, nil
	}

	return false, nil
}
