# mint

[![Go](https://github.com/otiai10/mint/actions/workflows/go.yml/badge.svg)](https://github.com/otiai10/mint/actions/workflows/go.yml)
[![codecov](https://codecov.io/gh/otiai10/mint/branch/master/graph/badge.svg)](https://codecov.io/gh/otiai10/mint)
[![Go Report Card](https://goreportcard.com/badge/github.com/otiai10/mint)](https://goreportcard.com/report/github.com/otiai10/mint)
[![GoDoc](https://godoc.org/github.com/otiai10/mint?status.png)](https://godoc.org/github.com/otiai10/mint)
[![GitHub tag (latest SemVer)](https://img.shields.io/github/v/tag/otiai10/mint?sort=semver)](https://pkg.go.dev/github.com/otiai10/mint)
[![FOSSA Status](https://app.fossa.com/api/projects/git%2Bgithub.com%2Fotiai10%2Fmint.svg?type=shield)](https://app.fossa.com/projects/git%2Bgithub.com%2Fotiai10%2Fmint?ref=badge_shield)

The very minimum assertion for Go.

```go
package your_test

import (
    "testing"
    "pkg/your"
    . "github.com/otiai10/mint"
)

func TestFoo(t *testing.T) {

    foo := your.Foo()
    Expect(t, foo).ToBe(1234)
    Expect(t, foo).TypeOf("int")
    Expect(t, foo).Not().ToBe(nil)
    Expect(t, func() { yourFunc() }).Exit(1)

    // If assertion failed, exit 1 with message.
    Expect(t, foo).ToBe("foobarbuz")

    // You can run assertions without os.Exit
    res := Expect(t, foo).Dry().ToBe("bar")
    // res.OK() == false

    // You can omit repeated `t`.
    m := mint.Blend(t)
    m.Expect(foo).ToBe(1234)
}
```

# features

- Simple syntax
- Loosely coupled
- Plain implementation

# tests
```
go test ./...
```

# use cases

Projects bellow use `mint`

- [github.com/otiai10/gosseract](https://github.com/otiai10/gosseract/blob/master/all_test.go)
- [github.com/otiai10/marmoset](https://github.com/otiai10/marmoset/blob/master/all_test.go#L168-L190)


## License
[![FOSSA Status](https://app.fossa.com/api/projects/git%2Bgithub.com%2Fotiai10%2Fmint.svg?type=large)](https://app.fossa.com/projects/git%2Bgithub.com%2Fotiai10%2Fmint?ref=badge_large)