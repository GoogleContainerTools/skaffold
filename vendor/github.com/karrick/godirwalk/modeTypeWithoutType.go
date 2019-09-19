// +build aix js nacl solaris

package godirwalk

import (
	"os"
	"path/filepath"
	"syscall"
)

// modeType converts a syscall defined constant, which is in purview of OS, to a
// constant defined by Go, assumed by this project to be stable.
//
// Because some operating system syscall.Dirent structure does not include a
// Type field, fall back on Stat of the file system.
func modeType(_ *syscall.Dirent, osDirname, osChildname string) (os.FileMode, error) {
	fi, err := os.Lstat(filepath.Join(osDirname, osChildname))
	if err != nil {
		return 0, err
	}
	// Even though the stat provided all file mode bits, we want to
	// ensure same values returned to caller regardless of whether
	// we obtained file mode bits from syscall or stat call.
	// Therefore mask out the additional file mode bits that are
	// provided by stat but not by the syscall, so users can rely on
	// their values.
	return fi.Mode() & os.ModeType, nil
}
