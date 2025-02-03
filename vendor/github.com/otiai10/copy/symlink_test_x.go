//go:build windows || plan9 || netbsd || aix || illumos || solaris || js

package copy

import (
	"os"
	"testing"

	. "github.com/otiai10/mint"
)

func TestOptions_OnSymlink(t *testing.T) {
	opt := Options{OnSymlink: func(string) SymlinkAction { return Deep }}
	err := Copy("test/data/case03", "test/data.copy/case03.deep", opt)
	Expect(t, err).ToBe(nil)
	info, err := os.Lstat("test/data.copy/case03.deep/case01")
	Expect(t, err).ToBe(nil)
	Expect(t, info.Mode()&os.ModeSymlink).ToBe(os.FileMode(0))

	opt = Options{OnSymlink: func(string) SymlinkAction { return Shallow }}
	err = Copy("test/data/case03", "test/data.copy/case03.shallow", opt)
	Expect(t, err).ToBe(nil)
	info, err = os.Lstat("test/data.copy/case03.shallow/case01")
	Expect(t, err).ToBe(nil)
	Expect(t, info.Mode()&os.ModeSymlink).Not().ToBe(os.FileMode(0))

	opt = Options{OnSymlink: func(string) SymlinkAction { return Skip }}
	err = Copy("test/data/case03", "test/data.copy/case03.skip", opt)
	Expect(t, err).ToBe(nil)
	_, err = os.Stat("test/data.copy/case03.skip/case01")
	Expect(t, os.IsNotExist(err)).ToBe(true)

	err = Copy("test/data/case03", "test/data.copy/case03.default")
	Expect(t, err).ToBe(nil)
	info, err = os.Lstat("test/data.copy/case03.default/case01")
	Expect(t, err).ToBe(nil)
	Expect(t, info.Mode()&os.ModeSymlink).Not().ToBe(os.FileMode(0))

	opt = Options{OnSymlink: nil}
	err = Copy("test/data/case03", "test/data.copy/case03.not-specified", opt)
	Expect(t, err).ToBe(nil)
	info, err = os.Lstat("test/data.copy/case03.not-specified/case01")
	Expect(t, err).ToBe(nil)
	Expect(t, info.Mode()&os.ModeSymlink).Not().ToBe(os.FileMode(0))
}
