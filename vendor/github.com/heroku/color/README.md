# Color [![GoDoc](https://godoc.org/github.com/heroku/color?status.svg)](https://godoc.org/github.com/heroku/color) [![CircleCI](https://circleci.com/gh/heroku/color.svg?style=svg)](https://circleci.com/gh/heroku/color)

Color is based on the [github.com/fatih/color](https://github.com/fatih/color) package. Unfortunately the original
color package is archived and is no longer supported. Like the original, this package lets you use colorized outputs in 
terms of [ANSI Escape Codes](http://en.wikipedia.org/wiki/ANSI_escape_code#Colors) in Golang but offers a number of 
improvements from the original. Posix and Windows platforms are supported.  

Color seeks to remain *mostly* backward compatible with fatih/color but has a number of changes to support concurrency,
improved performance and a more idiomatic style. 

## Changes And Improvements 

The methods of the new `Color` struct do not mutate the sender. This
results in better concurrency support and improved performance. 

You don't need to remember to wrap io.Writer arguments in `colorable.NewColorable` in order to support Windows functionality.

Package public global variables are removed.  `color.NoColor` was removed and replaced with the `Disable` function.
 Colored output can be toggled using the `Disable`. `color.Output` and `color.Error`replaced by `Stdout()` and `Stderr()` 
.  

Instances of `Console` can be passed to methods in third party packages that take `io.Writer` as an argument. If the 
 third party package emits ANSI color information the passed in writer will be interpreted correctly on Windows. In 
 addition, color information can be stripped for a console by calling `Console.DisableColors(true)`.

Performance is improved significantly, as much as 400%.  Note that some functions that you'd expect to take an 
array of interface{} take an array of strings instead because underlying calls to fmt.SprintXX functions are slow. 

`fatih/color` has race conditions.  This package was developed with `test.Parallel` and `-race` enabled for tests. Thus 
far no race conditions are known and so this package is suitable for use in a multi goroutine environment. 

## Examples

### Standard colors

```go
// Print with default helper functions
color.Cyan("Prints text in cyan.")

// A newline will be appended automatically
color.Blue("Prints %s in blue.", "text")
```

### Mix and reuse colors

```go
// Create a new color object
color.Stdout().Println(color.New(color.FgCyan, color.Underline), "Prints cyan text with an underline.")
```

### Use your own output (io.Writer)

```go
// Use your own io.Writer output
wtr := color.NewConsole(os.Stderr)
wtr.Println(color.New(color.FgBlue), "Hello! I'm blue.")
```

### Custom print functions (PrintFunc)

```go
// Create a custom print function for convenience
red := color.StdErr().PrintfFunc(color.New(color.FgRed))
red("Warning")
red("Error: %s", err)

// Mix up multiple attributes
notice := color.Stdout().PrintlnFunc(color.New(color.Bold, color.FgGreen))
notice("Don't forget this...")
```
### Insert into noncolor strings (SprintFunc)

```go
// Create SprintXxx functions to mix strings with other non-colorized strings:
yellow := color.New(color.FgYellow).SprintFunc()
red := color.New(color.FgRed).SprintFunc()
fmt.Printf("This is a %s and this is %s.\n", yellow("warning"), red("error"))

info := color.New(color.FgWhite, color.BgGreen).SprintFunc()
fmt.Printf("This %s rocks!\n", info("package"))

// Use helper functions
fmt.Println("This", color.RedString("warning"), "should be not neglected.")
fmt.Printf("%v %v\n", color.GreenString("Info:"), "an important message.")
```
### Disable/Enable color
 
There might be a case where you want to explicitly disable/enable color output. 

`Color` has support to disable/enable colors on a per `Console` basis.  
For example suppose you have a CLI app and a `--no-color` bool flag. You 
can easily disable the color output with:

```go

var flagNoColor = flag.Bool("no-color", false, "Disable color output")
color.Stdout().DisableColors(*flagNoColor)

```
## Credits

 * [Fatih Arslan](https://github.com/fatih)
 * Windows support via @mattn: [colorable](https://github.com/mattn/go-colorable)

## License

The MIT License (MIT) - see [`LICENSE.md`](https://github.com/heroku/color/blob/master/LICENSE) for more details


