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

// worktreeFilesystem wraps a billy.Filesystem and validates every path it
// is handed, so worktree operations cannot use dangerous paths at the
// boundary. Two layers apply:
//
//   - validPath rejects dangerous path *strings*: .git and its HFS+/NTFS
//     variants, "..", control characters, volume names.
//   - validNoLeadingSymlink rejects paths whose leading directories
//     already exist on disk as symlinks, so a write or delete cannot
//     follow a planted link out of the tree.
//
// Both layers run on every mutating operation (validWritePath) and every
// read (validReadPath). Chroot additionally refuses a symlink as the final
// component, so a sub-filesystem such as a submodule worktree cannot be
// scoped to a redirected target.
//
// The wrapper intentionally stops at leading-component traversal. Callers
// that need final-component no-follow semantics for materialisation
// (checkoutFile) enforce that directly by removing the blocking symlink
// before opening the destination path.
type worktreeFilesystem struct {
	billy.Filesystem
	protectNTFS bool
	protectHFS  bool
}

func newWorktreeFilesystem(fs billy.Filesystem, protectNTFS, protectHFS bool) *worktreeFilesystem {
	return &worktreeFilesystem{Filesystem: fs, protectNTFS: protectNTFS, protectHFS: protectHFS}
}

func (sfs *worktreeFilesystem) Create(filename string) (billy.File, error) {
	if err := sfs.validWritePath(filename); err != nil {
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
	if err := sfs.validWritePath(filename); err != nil {
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
	if err := sfs.validWritePath(filename); err != nil {
		return fmt.Errorf("remove: %w", err)
	}
	return sfs.Filesystem.Remove(filename)
}

func (sfs *worktreeFilesystem) Rename(from, to string) error {
	if err := sfs.validWritePath(from, to); err != nil {
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
	if err := sfs.validWritePath(link); err != nil {
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
	if err := sfs.validWritePath(path); err != nil {
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
	// Chroot scopes a sub-filesystem to path, so the final component must
	// be a real directory too: a symlink there would silently redirect the
	// scope (e.g. a submodule worktree) to a target outside the tree. This
	// is the "valid path, wrong target" case that validNoLeadingSymlink,
	// which only inspects leading components, does not cover.
	//
	// A non-existent target is fine: Chroot creates it as a real
	// directory. Any other Lstat error means we cannot prove the target
	// is not a symlink, so fail closed rather than scope through it.
	if fi, err := sfs.Filesystem.Lstat(path); err != nil {
		if !os.IsNotExist(err) {
			return nil, fmt.Errorf("chroot: cannot stat %q: %w", path, err)
		}
	} else if fi.Mode()&os.ModeSymlink != 0 {
		return nil, fmt.Errorf("chroot: invalid path %q: is a symlink", path)
	}
	return sfs.Filesystem.Chroot(path)
}

// validReadPath is like validWritePath but treats the empty string and "."
// as valid references to the worktree root. Read-side operations on the
// root (e.g. ReadDir(""), Lstat(".")) are legitimate. Mutating the root
// itself is not, so write-side operations reject it via validPath. Reads
// are still refused through a leading symlink, so the wrapper never
// follows a planted link even on the read surface.
func (sfs *worktreeFilesystem) validReadPath(p string) error {
	if p == "" || p == "." || p == "/" {
		return nil
	}
	if err := sfs.validPath(p); err != nil {
		return err
	}
	return sfs.validNoLeadingSymlink(p)
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

// validWritePath validates paths for mutating operations. It layers the
// filesystem-state check validNoLeadingSymlink on top of the string-only
// checks in validPath, so a write can neither name a dangerous path nor
// reach one by traversing an existing symlink. Every mutating method on
// the wrapper funnels through here, so the leading-symlink invariant holds
// for all worktree writers without each call site having to remember it.
func (sfs *worktreeFilesystem) validWritePath(paths ...string) error {
	if err := sfs.validPath(paths...); err != nil {
		return err
	}
	return sfs.validNoLeadingSymlink(paths...)
}

// validNoLeadingSymlink rejects paths whose leading directory components
// resolve through a symlink that already exists on the underlying
// filesystem. validPath guards the path string. This guards the on-disk
// state, so a write or delete cannot reach outside the worktree by
// traversing a symlink that a tree or an earlier step left in place.
//
// This is the fail-closed backstop for the whole class. Callers that want
// upstream's replace-and-continue behaviour (checkout) remove the blocking
// symlink first via clearBlockingSymlinks, so no symlink remains when the
// write reaches the wrapper. Callers that do not get a safe error,
// matching upstream Git refusing rather than following the link. See
// has_symlink_leading_path (symlinks.c) and the check_leading_path guard
// in unlink_entry (entry.c).
func (sfs *worktreeFilesystem) validNoLeadingSymlink(paths ...string) error {
	for _, p := range paths {
		for dir := filepath.Dir(p); dir != "." && dir != "" && dir != string(filepath.Separator); dir = filepath.Dir(dir) {
			fi, err := sfs.Filesystem.Lstat(dir)
			if err != nil {
				// A missing ancestor is materialised as a real directory,
				// so it cannot be a symlink and is safe to skip. Any other
				// error (permission, I/O) means we cannot prove the
				// component is not a symlink, so fail closed rather than
				// let the operation traverse an unverified component.
				if os.IsNotExist(err) {
					continue
				}
				return fmt.Errorf("invalid path %q: cannot stat leading component %q: %w", p, dir, err)
			}
			if fi.Mode()&os.ModeSymlink != 0 {
				return fmt.Errorf("invalid path %q: leading component %q is a symlink", p, dir)
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
