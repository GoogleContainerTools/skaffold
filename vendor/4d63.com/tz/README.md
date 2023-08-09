# tz
[![Tests](https://github.com/leighmcculloch/go-tz/workflows/tests/badge.svg)](https://github.com/leighmcculloch/go-tz/actions)
[![Go Report Card](https://goreportcard.com/badge/github.com/leighmcculloch/go-tz)](https://goreportcard.com/report/github.com/leighmcculloch/go-tz)
[![Go docs](https://img.shields.io/badge/godoc-reference-blue.svg)](https://godoc.org/4d63.com/tz)

**Deprecated: Use [time/tzdata](https://golang.org/pkg/time/tzdata/) available in Go 1.15. time/tzdata does not work exactly the same as it defaults to local tzdata when available where as this package always uses embedded data.**
**This package will be maintained until Go 1.16 is released.**

Predictably load `time.Location`s regardless of operating system.

```
import "4d63.com/tz"
```

```
loc, err := tz.LoadLocation("Australia/Sydney")
```

Docs and examples at https://godoc.org/4d63.com/tz.

This package exists because of https://github.com/golang/go/issues/21881.
