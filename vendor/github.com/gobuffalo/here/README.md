# Here

[![](https://github.com/gobuffalo/here/workflows/Tests/badge.svg)](https://github.com/gobuffalo/here/actions)
[![GoDoc](https://godoc.org/github.com/gobuffalo/here?status.svg)](https://godoc.org/github.com/gobuffalo/here)

Here will get you **accurate** Go information about the directory of package requested.

### Requirements

* Go 1.13+
* Go Modules

## CLI

While you can use the tool via its API, you can also use the CLI to get a JSON version of the data.

### Installation

```bash
$ go get github.com/gobuffalo/here/cmd/here
```

### Usage

#### Default

```bash
$ here

{
  "Dir": "$GOPATH/src/github.com/gobuffalo/here",
  "ImportPath": "github.com/gobuffalo/here",
  "Name": "here",
  "Doc": "",
  "Target": "$GOPATH/pkg/darwin_amd64/github.com/gobuffalo/here.a",
  "Root": "$GOPATH",
  "Match": [
    "."
  ],
  "Stale": true,
  "StaleReason": "not installed but available in build cache",
  "GoFiles": [
    "current.go",
    "dir.go",
    "here.go",
    "info.go",
    "info_map.go",
    "module.go",
    "pkg.go",
    "version.go"
  ],
  "Imports": [
    "bytes",
    "encoding/json",
    "fmt",
    "os",
    "os/exec",
    "path/filepath",
    "regexp",
    "sync"
  ],
  "Deps": [
    "bytes",
    "context",
    "encoding",
    "encoding/base64",
    "encoding/binary",
    "encoding/json",
    "errors",
    "fmt",
    "internal/bytealg",
    "internal/cpu",
    "internal/fmtsort",
    "internal/oserror",
    "internal/poll",
    "internal/race",
    "internal/reflectlite",
    "internal/syscall/unix",
    "internal/testlog",
    "io",
    "math",
    "math/bits",
    "os",
    "os/exec",
    "path/filepath",
    "reflect",
    "regexp",
    "regexp/syntax",
    "runtime",
    "runtime/internal/atomic",
    "runtime/internal/math",
    "runtime/internal/sys",
    "sort",
    "strconv",
    "strings",
    "sync",
    "sync/atomic",
    "syscall",
    "time",
    "unicode",
    "unicode/utf16",
    "unicode/utf8",
    "unsafe"
  ],
  "TestGoFiles": [
    "current_test.go",
    "dir_test.go",
    "here_test.go",
    "info_test.go",
    "module_test.go",
    "pkg_test.go"
  ],
  "TestImports": [
    "github.com/stretchr/testify/require",
    "os",
    "path/filepath",
    "testing"
  ],
  "Module": {
    "Path": "github.com/gobuffalo/here",
    "Main": true,
    "Dir": "$GOPATH/src/github.com/gobuffalo/here",
    "GoMod": "$GOPATH/src/github.com/gobuffalo/here/go.mod",
    "GoVersion": "1.13"
  }
}
```

#### By Directory

```bash
$ here cmd/here

{
  "Dir": "$GOPATH/src/github.com/gobuffalo/here/cmd/here",
  "ImportPath": "github.com/gobuffalo/here/cmd/here",
  "Name": "main",
  "Doc": "",
  "Target": "$GOPATH/bin/here",
  "Root": "$GOPATH",
  "Match": [
    "."
  ],
  "Stale": false,
  "StaleReason": "",
  "GoFiles": [
    "main.go"
  ],
  "Imports": [
    "fmt",
    "github.com/gobuffalo/here",
    "log",
    "os",
    "os/exec"
  ],
  "Deps": [
    "bytes",
    "context",
    "encoding",
    "encoding/base64",
    "encoding/binary",
    "encoding/json",
    "errors",
    "fmt",
    "github.com/gobuffalo/here",
    "internal/bytealg",
    "internal/cpu",
    "internal/fmtsort",
    "internal/oserror",
    "internal/poll",
    "internal/race",
    "internal/reflectlite",
    "internal/syscall/unix",
    "internal/testlog",
    "io",
    "log",
    "math",
    "math/bits",
    "os",
    "os/exec",
    "path/filepath",
    "reflect",
    "regexp",
    "regexp/syntax",
    "runtime",
    "runtime/internal/atomic",
    "runtime/internal/math",
    "runtime/internal/sys",
    "sort",
    "strconv",
    "strings",
    "sync",
    "sync/atomic",
    "syscall",
    "time",
    "unicode",
    "unicode/utf16",
    "unicode/utf8",
    "unsafe"
  ],
  "TestGoFiles": null,
  "TestImports": null,
  "Module": {
    "Path": "github.com/gobuffalo/here",
    "Main": true,
    "Dir": "$GOPATH/src/github.com/gobuffalo/here",
    "GoMod": "$GOPATH/src/github.com/gobuffalo/here/go.mod",
    "GoVersion": "1.13"
  }
}
```

#### By Package

```bash
$ here pkg github.com/gobuffalo/genny

{
  "Dir": "$GOPATH/pkg/mod/github.com/gobuffalo/genny@v0.4.1",
  "ImportPath": "github.com/gobuffalo/genny",
  "Name": "genny",
  "Doc": "Package genny is a _framework_ for writing modular generators, it however, doesn't actually generate anything.",
  "Target": "",
  "Root": "$GOPATH/pkg/mod/github.com/gobuffalo/genny@v0.4.1",
  "Match": [
    "github.com/gobuffalo/genny"
  ],
  "Stale": true,
  "StaleReason": "build ID mismatch",
  "GoFiles": [
    "confirm.go",
    "dir.go",
    "disk.go",
    "dry_runner.go",
    "events.go",
    "file.go",
    "force.go",
    "generator.go",
    "genny.go",
    "group.go",
    "helpers.go",
    "logger.go",
    "replacer.go",
    "results.go",
    "runner.go",
    "step.go",
    "transformer.go",
    "version.go",
    "wet_runner.go"
  ],
  "Imports": null,
  "Deps": null,
  "TestGoFiles": [
    "dry_runner_test.go",
    "file_test.go",
    "force_test.go",
    "generator_test.go",
    "genny_test.go",
    "group_test.go",
    "helpers_test.go",
    "replacer_test.go",
    "results_test.go",
    "runner_test.go",
    "step_test.go",
    "transformer_test.go",
    "wet_runner_test.go"
  ],
  "TestImports": null,
  "Module": {
    "Path": "github.com/gobuffalo/genny",
    "Main": false,
    "Dir": "$GOPATH/pkg/mod/github.com/gobuffalo/genny@v0.4.1",
    "GoMod": "$GOPATH/pkg/mod/cache/download/github.com/gobuffalo/genny/@v/v0.4.1.mod",
    "GoVersion": "1.13"
  }
}
```
