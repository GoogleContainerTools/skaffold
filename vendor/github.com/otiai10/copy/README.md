# copy

[![Go Reference](https://pkg.go.dev/badge/github.com/otiai10/copy.svg)](https://pkg.go.dev/github.com/otiai10/copy)
[![Actions Status](https://github.com/otiai10/copy/workflows/Go/badge.svg)](https://github.com/otiai10/copy/actions)
[![codecov](https://codecov.io/gh/otiai10/copy/branch/main/graph/badge.svg)](https://codecov.io/gh/otiai10/copy)
[![License: MIT](https://img.shields.io/badge/License-MIT-green.svg)](https://github.com/otiai10/copy/blob/main/LICENSE)
[![FOSSA Status](https://app.fossa.com/api/projects/git%2Bgithub.com%2Fotiai10%2Fcopy.svg?type=shield)](https://app.fossa.com/projects/git%2Bgithub.com%2Fotiai10%2Fcopy?ref=badge_shield)
[![CodeQL](https://github.com/otiai10/copy/actions/workflows/codeql-analysis.yml/badge.svg)](https://github.com/otiai10/copy/actions/workflows/codeql-analysis.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/otiai10/copy)](https://goreportcard.com/report/github.com/otiai10/copy)
[![GitHub tag (latest SemVer)](https://img.shields.io/github/v/tag/otiai10/copy?sort=semver)](https://pkg.go.dev/github.com/otiai10/copy)
[![Docker Test](https://github.com/otiai10/copy/actions/workflows/docker-test.yml/badge.svg)](https://github.com/otiai10/copy/actions/workflows/docker-test.yml)
[![Vagrant Test](https://github.com/otiai10/copy/actions/workflows/vagrant-test.yml/badge.svg)](https://github.com/otiai10/copy/actions/workflows/vagrant-test.yml)

`copy` copies directories recursively.

# Example Usage

```go
package main

import (
	"fmt"
	cp "github.com/otiai10/copy"
)

func main() {
	err := cp.Copy("your/src", "your/dest")
	fmt.Println(err) // nil
}
```

# Advanced Usage

```go
// Options specifies optional actions on copying.
type Options struct {

	// OnSymlink can specify what to do on symlink
	OnSymlink func(src string) SymlinkAction

	// OnDirExists can specify what to do when there is a directory already existing in destination.
	OnDirExists func(src, dest string) DirExistsAction

	// OnError can let users decide how to handle errors (e.g., you can suppress specific error).
	OnError func(src, dest, string, err error) error

	// Skip can specify which files should be skipped
	Skip func(srcinfo os.FileInfo, src, dest string) (bool, error)

	// PermissionControl can control permission of
	// every entry.
	// When you want to add permission 0222, do like
	//
	//		PermissionControl = AddPermission(0222)
	//
	// or if you even don't want to touch permission,
	//
	//		PermissionControl = DoNothing
	//
	// By default, PermissionControl = PreservePermission
	PermissionControl PermissionControlFunc

	// Sync file after copy.
	// Useful in case when file must be on the disk
	// (in case crash happens, for example),
	// at the expense of some performance penalty
	Sync bool

	// Preserve the atime and the mtime of the entries
	// On linux we can preserve only up to 1 millisecond accuracy
	PreserveTimes bool

	// Preserve the uid and the gid of all entries.
	PreserveOwner bool

	// The byte size of the buffer to use for copying files.
	// If zero, the internal default buffer of 32KB is used.
	// See https://golang.org/pkg/io/#CopyBuffer for more information.
	CopyBufferSize uint

	// If you want to add some limitation on reading src file,
	// you can wrap the src and provide new reader,
	// such as `RateLimitReader` in the test case.
	WrapReader func(src io.Reader) io.Reader

	// If given, copy.Copy refers to this fs.FS instead of the OS filesystem.
	// e.g., You can use embed.FS to copy files from embedded filesystem.
	FS fs.FS
}
```

```go
// For example...
opt := Options{
	Skip: func(info os.FileInfo, src, dest string) (bool, error) {
		return strings.HasSuffix(src, ".git"), nil
	},
}
err := Copy("your/directory", "your/directory.copy", opt)
```

# Issues

- https://github.com/otiai10/copy/issues


## License
[![FOSSA Status](https://app.fossa.com/api/projects/git%2Bgithub.com%2Fotiai10%2Fcopy.svg?type=large)](https://app.fossa.com/projects/git%2Bgithub.com%2Fotiai10%2Fcopy?ref=badge_large)