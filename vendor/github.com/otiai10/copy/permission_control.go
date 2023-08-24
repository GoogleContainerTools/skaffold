package copy

import (
	"os"
)

const (
	// tmpPermissionForDirectory makes the destination directory writable,
	// so that stuff can be copied recursively even if any original directory is NOT writable.
	// See https://github.com/otiai10/copy/pull/9 for more information.
	tmpPermissionForDirectory = os.FileMode(0755)
)

type PermissionControlFunc func(srcinfo fileInfo, dest string) (chmodfunc func(*error), err error)

var (
	AddPermission = func(perm os.FileMode) PermissionControlFunc {
		return func(srcinfo fileInfo, dest string) (func(*error), error) {
			orig := srcinfo.Mode()
			if srcinfo.IsDir() {
				if err := os.MkdirAll(dest, tmpPermissionForDirectory); err != nil {
					return func(*error) {}, err
				}
			}
			return func(err *error) {
				chmod(dest, orig|perm, err)
			}, nil
		}
	}
	PerservePermission PermissionControlFunc = AddPermission(0)
	DoNothing          PermissionControlFunc = func(srcinfo fileInfo, dest string) (func(*error), error) {
		if srcinfo.IsDir() {
			if err := os.MkdirAll(dest, srcinfo.Mode()); err != nil {
				return func(*error) {}, err
			}
		}
		return func(*error) {}, nil
	}
)

// chmod ANYHOW changes file mode,
// with asiging error raised during Chmod,
// BUT respecting the error already reported.
func chmod(dir string, mode os.FileMode, reported *error) {
	if err := os.Chmod(dir, mode); *reported == nil {
		*reported = err
	}
}
