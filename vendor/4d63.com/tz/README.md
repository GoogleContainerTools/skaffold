# tz
[![Build Status](https://img.shields.io/travis/leighmcculloch/go-tz.svg)](https://travis-ci.org/leighmcculloch/go-tz)
[![Codecov](https://img.shields.io/codecov/c/github/leighmcculloch/go-tz.svg)](https://codecov.io/gh/leighmcculloch/go-tz)
[![Go Report Card](https://goreportcard.com/badge/github.com/leighmcculloch/go-tz)](https://goreportcard.com/report/github.com/leighmcculloch/go-tz)
[![Go docs](https://img.shields.io/badge/godoc-reference-blue.svg)](https://godoc.org/4d63.com/tz)

Predictably load `time.Location`s regardless of operating system.

```
import "4d63.com/tz"
```

```
loc, err := tz.LoadLocation("Australia/Sydney")
```

Docs and examples at https://godoc.org/4d63.com/tz.

This package exists because of https://github.com/golang/go/issues/21881.
