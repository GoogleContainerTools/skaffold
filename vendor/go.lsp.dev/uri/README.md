# uri

[![test][test-badge]][test]
[![pkg.go.dev][pkg.go.dev-badge]][pkg.go.dev]
[![Go module][module-badge]][module]
[![codecov.io][codecov-badge]][codecov]

Package uri is a canonical `vscode-uri` parser and formatter for Go.

## Canonical escaping

Constructors such as `Parse`, `File`, `FileFor`, and `From` store the canonical
encoded form returned by `String`, text marshaling, and JSON marshaling. The
unreserved ASCII set `A-Z a-z 0-9 - . _ ~` stays raw everywhere. Other bytes are
escaped with uppercase percent triplets unless they are syntax characters that
are valid for the component being formatted.

| Component | Additional raw syntax | Examples |
| --- | --- | --- |
| Path | `/` | `@` -> `%40`, `:` -> `%3A`, `\` -> `%5C`, Unicode bytes -> UTF-8 percent triplets |
| Query and fragment | none | `=` -> `%3D`, `&` -> `%26`, `/` -> `%2F`, `@` -> `%40` |
| Authority host/port | `:`, `[`, `]` | `[::1]:8080` stays raw; `/` in an authority part becomes `%2F` |
| Authority userinfo delimiter | `@` | `http://user:pass@host:8080/p` keeps the delimiter raw |

For example,
`FileFor(PlatformPOSIX, "/Users/me/go/pkg/mod/example.com/mod@v1.2.3/file.go")`
formats as `file:///Users/me/go/pkg/mod/example.com/mod%40v1.2.3/file.go`.
`StringNoEncoding()` returns the `vscode-uri` `toString(true)` style form and can
expose decoded characters such as `@`, `=`, `&`, or Unicode text. The direct
`URI("...")` compatibility path described below is not a constructor path, so it
is outside this canonicalization step.

Constructor-produced `URI` values compare by canonical string identity. Direct
`URI("...")` conversions remain available for compatibility, but they do not
validate or canonicalize input. To keep native Go equality and map keys safe,
decoded component accessors expose the same view as reparsing `vscode-uri`'s
`URI.parse(input).toString()` output for canonical values: original
parse-history-only casing such as `file://SERVER/...` authorities or
`file:///C:/...` drive letters is normalized in `Authority`, `Path`, and
`FsPath`.

Performance notes and reproducible benchmark commands are in
[docs/perf.md](docs/perf.md). Conformance vectors are regenerated from the
pinned Node dependency in [tools/genvectors](tools/genvectors/README.md).


<!-- badge links -->
[test]: https://github.com/go-language-server/uri/actions/workflows/test.yaml
[pkg.go.dev]: https://pkg.go.dev/go.lsp.dev/uri
[module]: https://github.com/go-language-server/uri/releases/latest
[codecov]: https://app.codecov.io/gh/go-language-server/uri

[test-badge]: https://img.shields.io/github/actions/workflow/status/go-language-server/uri/test.yaml?branch=main&style=for-the-badge&label=TEST&logo=github
[pkg.go.dev-badge]: https://img.shields.io/badge/pkg.go.dev-doc-00add8?style=for-the-badge&logo=go
[module-badge]: https://img.shields.io/github/release/go-language-server/uri.svg?color=00add8&label=MODULE&style=for-the-badge&logo=go
[codecov-badge]: https://img.shields.io/codecov/c/github/go-language-server/uri/main?logo=codecov&style=for-the-badge
