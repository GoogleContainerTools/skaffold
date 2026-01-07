# jsonpointer

<!-- Badges: status  -->
[![Tests][test-badge]][test-url] [![Coverage][cov-badge]][cov-url] [![CI vuln scan][vuln-scan-badge]][vuln-scan-url] [![CodeQL][codeql-badge]][codeql-url]
<!-- Badges: release & docker images  -->
[![Release][release-badge]][release-url]
<!-- Badges: code quality  -->
[![Go Report Card][gocard-badge]][gocard-url] [![CodeFactor Grade][codefactor-badge]][codefactor-url]
<!-- Badges: license & compliance -->
[![License][license-badge]][license-url]
<!-- Badges: documentation & support -->
<!-- Badges: others & stats -->
[![GoDoc][godoc-badge]][godoc-url] [![Slack Channel][slack-badge]][slack-url] [![go version][goversion-badge]][goversion-url] ![Top language][top-badge] ![Commits since latest release][commits-badge]

---

An implementation of JSON Pointer for golang, which supports go `struct`.

## Status

API is stable.

## Import this library in your project

```cmd
go get github.com/go-openapi/jsonpointer
```

## Basic usage

See [examples](./examples_test.go)

```go
  import (
    "github.com/go-openapi/jsonpointer"
  )

  ...

	pointer, err := jsonpointer.New("/foo/1")
	if err != nil {
		... // error: e.g. invalid JSON pointer specification
	}

	value, kind, err := pointer.Get(doc)
	if err != nil {
		... // error: e.g. key not found, index out of bounds, etc.
	}

  ...
```

## Change log

See <https://github.com/go-openapi/jsonpointer/releases>

## References

<https://tools.ietf.org/html/draft-ietf-appsawg-json-pointer-07>

also known as [RFC6901](https://www.rfc-editor.org/rfc/rfc6901)

## Licensing

This library ships under the [SPDX-License-Identifier: Apache-2.0](./LICENSE).

See the license [NOTICE](./NOTICE), which recalls the licensing terms of all the pieces of software
on top of which it has been built.

## Limitations

The 4.Evaluation part of the previous reference, starting with 'If the currently referenced value is a JSON array,
the reference token MUST contain either...' is not implemented.

That is because our implementation of the JSON pointer only supports explicit references to array elements:
the provision in the spec to resolve non-existent members as "the last element in the array",
using the special trailing character "-" is not implemented.

<!-- Badges: status  -->
[test-badge]: https://github.com/go-openapi/jsonpointer/actions/workflows/go-test.yml/badge.svg
[test-url]: https://github.com/go-openapi/jsonpointer/actions/workflows/go-test.yml
[cov-badge]: https://codecov.io/gh/go-openapi/jsonpointer/branch/master/graph/badge.svg
[cov-url]: https://codecov.io/gh/go-openapi/jsonpointer
[vuln-scan-badge]: https://github.com/go-openapi/jsonpointer/actions/workflows/scanner.yml/badge.svg
[vuln-scan-url]: https://github.com/go-openapi/jsonpointer/actions/workflows/scanner.yml
[codeql-badge]: https://github.com/go-openapi/jsonpointer/actions/workflows/codeql.yml/badge.svg
[codeql-url]: https://github.com/go-openapi/jsonpointer/actions/workflows/codeql.yml
<!-- Badges: release & docker images  -->
[release-badge]: https://badge.fury.io/gh/go-openapi%2Fjsonpointer.svg
[release-url]: https://badge.fury.io/gh/go-openapi%2Fjsonpointer
<!-- Badges: code quality  -->
[gocard-badge]: https://goreportcard.com/badge/github.com/go-openapi/jsonpointer
[gocard-url]: https://goreportcard.com/report/github.com/go-openapi/jsonpointer
[codefactor-badge]: https://img.shields.io/codefactor/grade/github/go-openapi/jsonpointer
[codefactor-url]: https://www.codefactor.io/repository/github/go-openapi/jsonpointer
<!-- Badges: documentation & support -->
[doc-badge]: https://img.shields.io/badge/doc-site-blue?link=https%3A%2F%2Fgoswagger.io%2Fgo-openapi%2F
[doc-url]: https://goswagger.io/go-openapi
[godoc-badge]: https://pkg.go.dev/github.com/go-openapi/jsonpointer?status.svg
[godoc-url]: http://pkg.go.dev/github.com/go-openapi/jsonpointer
[slack-badge]: https://slackin.goswagger.io/badge.svg
[slack-url]: https://slackin.goswagger.io
<!-- Badges: license & compliance -->
[license-badge]: http://img.shields.io/badge/license-Apache%20v2-orange.svg
[license-url]: https://github.com/go-openapi/jsonpointer/?tab=Apache-2.0-1-ov-file#readme
<!-- Badges: others & stats -->
[goversion-badge]: https://img.shields.io/github/go-mod/go-version/go-openapi/jsonpointer
[goversion-url]: https://github.com/go-openapi/jsonpointer/blob/master/go.mod
[top-badge]: https://img.shields.io/github/languages/top/go-openapi/jsonpointer
[commits-badge]: https://img.shields.io/github/commits-since/go-openapi/jsonpointer/latest
