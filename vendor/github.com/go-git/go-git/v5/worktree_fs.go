package git

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/go-git/go-billy/v5"

	"github.com/go-git/go-git/v5/internal/pathutil"
)

// defaultProtectHFS returns the default value for core.protectHFS
// when not explicitly configured. Matches upstream Git's
// PROTECT_HFS_DEFAULT[1], which the Makefile sets to 1 on Darwin
// and leaves at 0 on every other platform.
//
// [1]: https://github.com/git/git/blob/v2.54.0/config.mak.uname#L146
func defaultProtectHFS() bool {
	return runtime.GOOS == "darwin"
}

// defaultProtectNTFS returns the default value for core.protectNTFS
// when not explicitly configured. Matches upstream Git's
// PROTECT_NTFS_DEFAULT, which has been 1 on every platform since
// 9102f958ee5 (CVE-2019-1353)[1]: WSL allows Linux processes to
// reach NTFS-mounted worktrees on Windows hosts, so the
// is_ntfs_dotgit guard cannot safely be gated on the runtime OS.
//
// [1]: https://github.com/git/git/commit/9102f958ee5
func defaultProtectNTFS() bool {
	return true
}

// worktreeFilesystem wraps a billy.Filesystem and validates every path passed
// to a mutating operation. This prevents writing to, or deleting from,
// dangerous locations (e.g. .git/*, ../) regardless of which worktree
// code path triggers the operation.
type worktreeFilesystem struct {
	billy.Filesystem
	protectNTFS bool
	protectHFS  bool
}

func newWorktreeFilesystem(fs billy.Filesystem, protectNTFS, protectHFS bool) *worktreeFilesystem {
	return &worktreeFilesystem{Filesystem: fs, protectNTFS: protectNTFS, protectHFS: protectHFS}
}

func (sfs *worktreeFilesystem) Create(filename string) (billy.File, error) {
	if err := sfs.validPath(filename); err != nil {
		return nil, fmt.Errorf("create: %w", err)
	}
	return sfs.Filesystem.Create(filename)
}

func (sfs *worktreeFilesystem) Open(filename string) (billy.File, error) {
	if err := sfs.validReadPath(filename); err != nil {
		return nil, fmt.Errorf("open: %w", err)
	}
	return sfs.Filesystem.Open(filename)
}

func (sfs *worktreeFilesystem) OpenFile(filename string, flag int, perm os.FileMode) (billy.File, error) {
	if err := sfs.validPath(filename); err != nil {
		return nil, fmt.Errorf("openfile: %w", err)
	}
	return sfs.Filesystem.OpenFile(filename, flag, perm)
}

func (sfs *worktreeFilesystem) Stat(filename string) (os.FileInfo, error) {
	if err := sfs.validReadPath(filename); err != nil {
		return nil, fmt.Errorf("stat: %w", err)
	}
	return sfs.Filesystem.Stat(filename)
}

func (sfs *worktreeFilesystem) Remove(filename string) error {
	if err := sfs.validPath(filename); err != nil {
		return fmt.Errorf("remove: %w", err)
	}
	return sfs.Filesystem.Remove(filename)
}

func (sfs *worktreeFilesystem) Rename(from, to string) error {
	if err := sfs.validPath(from, to); err != nil {
		return fmt.Errorf("rename: %w", err)
	}
	return sfs.Filesystem.Rename(from, to)
}

func (sfs *worktreeFilesystem) ReadDir(path string) ([]os.FileInfo, error) {
	if err := sfs.validReadPath(path); err != nil {
		return nil, fmt.Errorf("readdir: %w", err)
	}
	return sfs.Filesystem.ReadDir(path)
}

func (sfs *worktreeFilesystem) Lstat(filename string) (os.FileInfo, error) {
	if err := sfs.validReadPath(filename); err != nil {
		return nil, fmt.Errorf("lstat: %w", err)
	}
	return sfs.Filesystem.Lstat(filename)
}

func (sfs *worktreeFilesystem) Symlink(target, link string) error {
	if err := sfs.validPath(link); err != nil {
		return fmt.Errorf("symlink: %w", err)
	}
	if err := sfs.validSymlinkName(link); err != nil {
		return fmt.Errorf("symlink: %w", err)
	}
	return sfs.Filesystem.Symlink(target, link)
}

func (sfs *worktreeFilesystem) Readlink(link string) (string, error) {
	if err := sfs.validReadPath(link); err != nil {
		return "", fmt.Errorf("readlink: %w", err)
	}
	return sfs.Filesystem.Readlink(link)
}

func (sfs *worktreeFilesystem) MkdirAll(path string, perm os.FileMode) error {
	// MkdirAll on the worktree root is a no-op: the root always exists,
	// so there is nothing to materialise. Mirroring the tolerance that
	// validReadPath gives to read-side operations avoids breaking callers
	// that walk a directory tree and pass the relative-to-root prefix
	// ("") through to the worktree FS.
	if path == "" || path == "." || path == "/" {
		return nil
	}
	if err := sfs.validPath(path); err != nil {
		return fmt.Errorf("mkdirall: %w", err)
	}
	return sfs.Filesystem.MkdirAll(path, perm)
}

func (sfs *worktreeFilesystem) TempFile(_, _ string) (billy.File, error) {
	return nil, fmt.Errorf("tempfile: %w", errUnsupportedOperation)
}

func (sfs *worktreeFilesystem) Chroot(path string) (billy.Filesystem, error) {
	if err := sfs.validReadPath(path); err != nil {
		return nil, fmt.Errorf("chroot: %w", err)
	}
	return sfs.Filesystem.Chroot(path)
}

// validReadPath is like validPath but treats the empty string and "." as
// valid references to the worktree root. Read-side operations on the root
// (e.g. ReadDir(""), Lstat(".")) are legitimate; mutating the root itself
// is not, so write-side operations continue to use validPath directly.
func (sfs *worktreeFilesystem) validReadPath(p string) error {
	if p == "" || p == "." || p == "/" {
		return nil
	}
	return sfs.validPath(p)
}

var errUnsupportedOperation = errors.New("unsupported operation")

// isDotGitVariant reports whether part is .git, git~1, or an HFS+
// equivalent of .git (when protectHFS is true). NTFS variants of .git
// (e.g. ".git " with trailing space, ".git::$INDEX_ALLOCATION") are
// detected separately by pathutil.WindowsValidPath, which applies
// regardless of position in the path. Both validators reuse this
// helper.
func isDotGitVariant(part string, protectHFS bool) bool {
	if pathutil.IsDotGitName(part) {
		return true
	}
	if protectHFS && pathutil.IsHFSDotGit(part) {
		return true
	}
	return false
}

// validPath checks whether paths are valid for the worktree
// filesystem abstraction. It is intentionally tolerant of .git as
// the final path component of a multi-component path
// (e.g. "submodule/.git"), so that legitimate gitlink pointer files
// can still be Stat'd, Read, and Removed via the wrapper during
// submodule cleanup. Attacker-controlled tree-entry paths are
// validated separately by pathutil.ValidTreePath at the boundaries
// where data leaves the trusted store (Tree.FindEntry, the explicit
// callers in CherryPick and Submodule.Repository).
//
// For upstream rules:
// https://github.com/git/git/blob/v2.54.0/read-cache.c#L987
// https://github.com/git/git/blob/v2.54.0/path.c#L1419
func (sfs *worktreeFilesystem) validPath(paths ...string) error {
	for _, p := range paths {
		for i := 0; i < len(p); i++ {
			if p[i] < 0x20 || p[i] == 0x7f {
				return fmt.Errorf("invalid path %q: contains control character", p)
			}
		}

		parts := strings.FieldsFunc(p, func(r rune) bool { return (r == '\\' || r == '/') })
		if len(parts) == 0 {
			return fmt.Errorf("invalid path: %q", p)
		}

		if sfs.protectNTFS {
			// Volume names are not supported, in both formats: \\ and <DRIVE_LETTER>:.
			if vol := filepath.VolumeName(p); vol != "" {
				return fmt.Errorf("invalid path: %q", p)
			}
		}

		for i, part := range parts {
			if part == "." || part == ".." {
				return fmt.Errorf("invalid path %q: cannot use %q", p, part)
			}

			// Reject .git (and equivalents) as a path component when it is
			// either the first component (root-level .git) or a non-final
			// component (traversal into a .git directory, e.g. "a/.git/config").
			// A final non-first .git component (e.g. "submodule/.git") is
			// allowed because submodule worktrees contain a .git pointer file.
			if isDotGitVariant(part, sfs.protectHFS) && (i == 0 || i < len(parts)-1) {
				return fmt.Errorf("invalid path component: %q", p)
			}

			if sfs.protectNTFS && !pathutil.WindowsValidPath(part) {
				return fmt.Errorf("invalid path: %q", p)
			}
		}
	}
	return nil
}

// validSymlinkName checks the per-component name of a symlink for
// dotfile names that attackers can use to trick a checkout into
// writing a dangerous symlink. Each path component is compared
// against .gitmodules case-insensitively, against its NTFS variants
// (e.g. ".gitmodules .", ".gitmodules::$INDEX_ALLOCATION", or 8.3
// short-name forms) when protectNTFS is on, and against its HFS+
// variants (Unicode ignored code points folded into ".gitmodules")
// when protectHFS is on.
//
// Reference: upstream Git verify_path_internal at read-cache.c#L1004-L1024
// in tag v2.54.0[1].
//
// [1]: https://github.com/git/git/blob/v2.54.0/read-cache.c#L1004-L1024
func (sfs *worktreeFilesystem) validSymlinkName(name string) error {
	parts := strings.FieldsFunc(name, func(r rune) bool {
		return r == '/' || r == '\\'
	})
	for _, part := range parts {
		if strings.EqualFold(part, gitmodulesFile) {
			return ErrGitModulesSymlink
		}
		if sfs.protectNTFS && pathutil.IsNTFSDotGitmodules(part) {
			return ErrGitModulesSymlink
		}
		if sfs.protectHFS && pathutil.IsHFSDotGitmodules(part) {
			return ErrGitModulesSymlink
		}
	}
	return nil
}
