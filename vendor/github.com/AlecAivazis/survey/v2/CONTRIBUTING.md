# Contributing to Survey

🎉🎉 First off, thanks for the interest in contributing to `survey`! 🎉🎉

The following is a set of guidelines to follow when contributing to this package. These are not hard rules, please use common sense and feel free to propose changes to this document in a pull request.

## Code of Conduct

This project and its contibutors are expected to uphold the [Go Community Code of Conduct](https://golang.org/conduct). By participating, you are expected to follow these guidelines.

## Getting help

* [Open an issue](https://github.com/AlecAivazis/survey/issues/new/choose)
* Reach out to `@AlecAivazis` or `@mislav` in the Gophers slack (please use only when urgent)

## Submitting a contribution

When submitting a contribution,

- Try to make a series of smaller changes instead of one large change
- Provide a description of each change that you are proposing
- Reference the issue addressed by your pull request (if there is one)
- Document all new exported Go APIs
- Update the project's README when applicable
- Include unit tests if possible
- Contributions with visual ramifications or interaction changes should be accompanied with an integration test—see below for details.

## Writing and running tests

When submitting features, please add as many units tests as necessary to test both positive and negative cases.

Integration tests for survey uses [go-expect](https://github.com/Netflix/go-expect) to expect a match on stdout and respond on stdin. Since `os.Stdout` in a `go test` process is not a TTY, you need a way to interpret terminal / ANSI escape sequences for things like `CursorLocation`. The stdin/stdout handled by `go-expect` is also multiplexed to a [virtual terminal](https://github.com/hinshun/vt10x).

For example, you can extend the tests for Input by specifying the following test case:

```go
{
  "Test Input prompt interaction",       // Name of the test.
  &Input{                                // An implementation of the survey.Prompt interface.
    Message: "What is your name?",
  },
  func(c *expect.Console) {              // An expect procedure. You can expect strings / regexps and
    c.ExpectString("What is your name?") // write back strings / bytes to its psuedoterminal for survey.
    c.SendLine("Johnny Appleseed")
    c.ExpectEOF()                        // Nothing is read from the tty without an expect, and once an
                                         // expectation is met, no further bytes are read. End your
                                         // procedure with `c.ExpectEOF()` to read until survey finishes.
  },
  "Johnny Appleseed",                    // The expected result.
}
```

If you want to write your own `go-expect` test from scratch, you'll need to instantiate a virtual terminal,
multiplex it into an `*expect.Console`, and hook up its tty with survey's optional stdio. Please see `go-expect`
[documentation](https://godoc.org/github.com/Netflix/go-expect) for more detail.
