# Survey

[![GoDoc](http://img.shields.io/badge/godoc-reference-5272B4.svg)](https://pkg.go.dev/github.com/AlecAivazis/survey/v2)

A library for building interactive and accessible prompts on terminals supporting ANSI escape sequences.

<img width="550" src="https://thumbs.gfycat.com/VillainousGraciousKouprey-size_restricted.gif"/>

```go
package main

import (
    "fmt"
    "github.com/AlecAivazis/survey/v2"
)

// the questions to ask
var qs = []*survey.Question{
    {
        Name:     "name",
        Prompt:   &survey.Input{Message: "What is your name?"},
        Validate: survey.Required,
        Transform: survey.Title,
    },
    {
        Name: "color",
        Prompt: &survey.Select{
            Message: "Choose a color:",
            Options: []string{"red", "blue", "green"},
            Default: "red",
        },
    },
    {
        Name: "age",
        Prompt:   &survey.Input{Message: "How old are you?"},
    },
}

func main() {
    // the answers will be written to this struct
    answers := struct {
        Name          string                  // survey will match the question and field names
        FavoriteColor string `survey:"color"` // or you can tag fields to match a specific name
        Age           int                     // if the types don't match, survey will convert it
    }{}

    // perform the questions
    err := survey.Ask(qs, &answers)
    if err != nil {
        fmt.Println(err.Error())
        return
    }

    fmt.Printf("%s chose %s.", answers.Name, answers.FavoriteColor)
}
```

## Examples

Examples can be found in the `examples/` directory. Run them
to see basic behavior:

```bash
go run examples/simple.go
go run examples/validation.go
```

## Running the Prompts

There are two primary ways to execute prompts and start collecting information from your users: `Ask` and
`AskOne`. The primary difference is whether you are interested in collecting a single piece of information
or if you have a list of questions to ask whose answers should be collected in a single struct.
For most basic usecases, `Ask` should be enough. However, for surveys with complicated branching logic,
we recommend that you break out your questions into multiple calls to both of these functions to fit your needs.

### Configuring the Prompts

Most prompts take fine-grained configuration through fields on the structs you instantiate. It is also
possible to change survey's default behaviors by passing `AskOpts` to either `Ask` or `AskOne`. Examples
in this document will do both interchangeably:

```golang
prompt := &Select{
    Message: "Choose a color:",
    Options: []string{"red", "blue", "green"},
    // can pass a validator directly
    Validate: survey.Required,
}

// or define a default for the single call to `AskOne`
// the answer will get written to the color variable
survey.AskOne(prompt, &color, survey.WithValidator(survey.Required))

// or define a default for every entry in a list of questions
// the answer will get copied into the matching field of the struct as shown above
survey.Ask(questions, &answers, survey.WithValidator(survey.Required))
```

## Prompts

### Input

<img src="https://thumbs.gfycat.com/LankyBlindAmericanpainthorse-size_restricted.gif" width="400px"/>

```golang
name := ""
prompt := &survey.Input{
    Message: "ping",
}
survey.AskOne(prompt, &name)
```

#### Suggestion Options

<img src="https://i.imgur.com/Q7POpA1.gif" width="800px"/>

```golang
file := ""
prompt := &survey.Input{
    Message: "inform a file to save:",
    Suggest: func (toComplete string) []string {
        files, _ := filepath.Glob(toComplete + "*")
        return files
    },
}
}
survey.AskOne(prompt, &file)
```

### Multiline

<img src="https://thumbs.gfycat.com/ImperfectShimmeringBeagle-size_restricted.gif" width="400px"/>

```golang
text := ""
prompt := &survey.Multiline{
    Message: "ping",
}
survey.AskOne(prompt, &text)
```

### Password

<img src="https://thumbs.gfycat.com/CompassionateSevereHypacrosaurus-size_restricted.gif" width="400px" />

```golang
password := ""
prompt := &survey.Password{
    Message: "Please type your password",
}
survey.AskOne(prompt, &password)
```

### Confirm

<img src="https://thumbs.gfycat.com/UnkemptCarefulGermanpinscher-size_restricted.gif" width="400px"/>

```golang
name := false
prompt := &survey.Confirm{
    Message: "Do you like pie?",
}
survey.AskOne(prompt, &name)
```

### Select

<img src="https://thumbs.gfycat.com/GrimFilthyAmazonparrot-size_restricted.gif" width="450px"/>

```golang
color := ""
prompt := &survey.Select{
    Message: "Choose a color:",
    Options: []string{"red", "blue", "green"},
}
survey.AskOne(prompt, &color)
```

Fields and values that come from a `Select` prompt can be one of two different things. If you pass an `int`
the field will have the value of the selected index. If you instead pass a string, the string value selected
will be written to the field.

The user can also press `esc` to toggle the ability cycle through the options with the j and k keys to do down and up respectively.

By default, the select prompt is limited to showing 7 options at a time
and will paginate lists of options longer than that. This can be changed a number of ways:

```golang
// as a field on a single select
prompt := &survey.MultiSelect{..., PageSize: 10}

// or as an option to Ask or AskOne
survey.AskOne(prompt, &days, survey.WithPageSize(10))
```

#### Select options description

The optional description text can be used to add extra information to each option listed in the select prompt:

```golang
color := ""
prompt := &survey.Select{
    Message: "Choose a color:",
    Options: []string{"red", "blue", "green"},
    Description: func(value string, index int) string {
        if value == "red" {
            return "My favorite color"
        }
        return ""
    },
}
survey.AskOne(prompt, &color)

// Assuming that the user chose "red - My favorite color":
fmt.Println(color) //=> "red"
```

### MultiSelect

![Example](img/multi-select-all-none.gif)

```golang
days := []string{}
prompt := &survey.MultiSelect{
    Message: "What days do you prefer:",
    Options: []string{"Sunday", "Monday", "Tuesday", "Wednesday", "Thursday", "Friday", "Saturday"},
}
survey.AskOne(prompt, &days)
```

Fields and values that come from a `MultiSelect` prompt can be one of two different things. If you pass an `int`
the field will have a slice of the selected indices. If you instead pass a string, a slice of the string values
selected will be written to the field.

The user can also press `esc` to toggle the ability cycle through the options with the j and k keys to do down and up respectively.

By default, the MultiSelect prompt is limited to showing 7 options at a time
and will paginate lists of options longer than that. This can be changed a number of ways:

```golang
// as a field on a single select
prompt := &survey.MultiSelect{..., PageSize: 10}

// or as an option to Ask or AskOne
survey.AskOne(prompt, &days, survey.WithPageSize(10))
```

### Editor

Launches the user's preferred editor (defined by the \$VISUAL or \$EDITOR environment variables) on a
temporary file. Once the user exits their editor, the contents of the temporary file are read in as
the result. If neither of those are present, notepad (on Windows) or vim (Linux or Mac) is used.

You can also specify a [pattern](https://golang.org/pkg/io/ioutil/#TempFile) for the name of the temporary file. This
can be useful for ensuring syntax highlighting matches your usecase.

```golang
prompt := &survey.Editor{
    Message: "Shell code snippet",
    FileName: "*.sh",
}

survey.AskOne(prompt, &content)
```

## Filtering Options

By default, the user can filter for options in Select and MultiSelects by typing while the prompt
is active. This will filter out all options that don't contain the typed string anywhere in their name, ignoring case.

A custom filter function can also be provided to change this behavior:

```golang
func myFilter(filterValue string, optValue string, optIndex int) bool {
    // only include the option if it includes the filter and has length greater than 5
    return strings.Contains(optValue, filterValue) && len(optValue) >= 5
}

// configure it for a specific prompt
&Select{
    Message: "Choose a color:",
    Options: []string{"red", "blue", "green"},
    Filter: myFilter,
}

// or define a default for all of the questions
survey.AskOne(prompt, &color, survey.WithFilter(myFilter))
```

## Keeping the filter active

By default the filter will disappear if the user selects one of the filtered elements. Once the user selects one element the filter setting is gone.

However the user can prevent this from happening and keep the filter active for multiple selections in a e.g. MultiSelect:

```golang
// configure it for a specific prompt
&Select{
    Message:    "Choose a color:",
    Options:    []string{"light-green", "green", "dark-green", "red"},
    KeepFilter: true,
}

// or define a default for all of the questions
survey.AskOne(prompt, &color, survey.WithKeepFilter(true))
```

## Validation

Validating individual responses for a particular question can be done by defining a
`Validate` field on the `survey.Question` to be validated. This function takes an
`interface{}` type and returns an error to show to the user, prompting them for another
response. Like usual, validators can be provided directly to the prompt or with `survey.WithValidator`:

```golang
q := &survey.Question{
    Prompt: &survey.Input{Message: "Hello world validation"},
    Validate: func (val interface{}) error {
        // since we are validating an Input, the assertion will always succeed
        if str, ok := val.(string) ; !ok || len(str) > 10 {
            return errors.New("This response cannot be longer than 10 characters.")
        }
	return nil
    },
}

color := ""
prompt := &survey.Input{ Message: "Whats your name?" }

// you can pass multiple validators here and survey will make sure each one passes
survey.AskOne(prompt, &color, survey.WithValidator(survey.Required))
```

### Built-in Validators

`survey` comes prepackaged with a few validators to fit common situations. Currently these
validators include:

| name         | valid types    | description                                                      | notes                                                                                 |
| ------------ | -------------- | ---------------------------------------------------------------- | ------------------------------------------------------------------------------------- |
| Required     | any            | Rejects zero values of the response type                         | Boolean values pass straight through since the zero value (false) is a valid response |
| MinLength(n) | string         | Enforces that a response is at least the given length            |                                                                                       |
| MaxLength(n) | string         | Enforces that a response is no longer than the given length      |                                                                                       |
| MaxItems(n)  | []OptionAnswer | Enforces that a response has no more selections of the indicated |                                                                                       |
| MinItems(n)  | []OptionAnswer | Enforces that a response has no less selections of the indicated |                                                                                       |

## Help Text

All of the prompts have a `Help` field which can be defined to provide more information to your users:

<img src="https://thumbs.gfycat.com/CloudyRemorsefulFossa-size_restricted.gif" width="400px" style="margin-top: 8px"/>

```golang
&survey.Input{
    Message: "What is your phone number:",
    Help:    "Phone number should include the area code",
}
```

## Removing the "Select All" and "Select None" options

By default, users can select all of the multi-select options using the right arrow key. To prevent users from being able to do this (and remove the `<right> to all` message from the prompt), use the option `WithRemoveSelectAll`:

```golang
import (
    "github.com/AlecAivazis/survey/v2"
)

number := ""
prompt := &survey.Input{
    Message: "This question has the select all option removed",
}

survey.AskOne(prompt, &number, survey.WithRemoveSelectAll())
```

Also by default, users can use the left arrow key to unselect all of the options. To prevent users from being able to do this (and remove the `<left> to none` message from the prompt), use the option `WithRemoveSelectNone`:

```golang
import (
    "github.com/AlecAivazis/survey/v2"
)

number := ""
prompt := &survey.Input{
    Message: "This question has the select all option removed",
}

survey.AskOne(prompt, &number, survey.WithRemoveSelectNone())
```


### Changing the input rune

In some situations, `?` is a perfectly valid response. To handle this, you can change the rune that survey
looks for with `WithHelpInput`:

```golang
import (
    "github.com/AlecAivazis/survey/v2"
)

number := ""
prompt := &survey.Input{
    Message: "If you have this need, please give me a reasonable message.",
    Help:    "I couldn't come up with one.",
}

survey.AskOne(prompt, &number, survey.WithHelpInput('^'))
```

## Changing the Icons

Changing the icons and their color/format can be done by passing the `WithIcons` option. The format
follows the patterns outlined [here](https://github.com/mgutz/ansi#style-format). For example:

```golang
import (
    "github.com/AlecAivazis/survey/v2"
)

number := ""
prompt := &survey.Input{
    Message: "If you have this need, please give me a reasonable message.",
    Help:    "I couldn't come up with one.",
}

survey.AskOne(prompt, &number, survey.WithIcons(func(icons *survey.IconSet) {
    // you can set any icons
    icons.Question.Text = "⁇"
    // for more information on formatting the icons, see here: https://github.com/mgutz/ansi#style-format
    icons.Question.Format = "yellow+hb"
}))
```

The icons and their default text and format are summarized below:

| name           | text | format     | description                                                   |
| -------------- | ---- | ---------- | ------------------------------------------------------------- |
| Error          | X    | red        | Before an error                                               |
| Help           | i    | cyan       | Before help text                                              |
| Question       | ?    | green+hb   | Before the message of a prompt                                |
| SelectFocus    | >    | green      | Marks the current focus in `Select` and `MultiSelect` prompts |
| UnmarkedOption | [ ]  | default+hb | Marks an unselected option in a `MultiSelect` prompt          |
| MarkedOption   | [x]  | cyan+b     | Marks a chosen selection in a `MultiSelect` prompt            |

## Custom Types

survey will assign prompt answers to your custom types if they implement this interface:

```golang
type Settable interface {
    WriteAnswer(field string, value interface{}) error
}
```

Here is an example how to use them:

```golang
type MyValue struct {
    value string
}
func (my *MyValue) WriteAnswer(name string, value interface{}) error {
     my.value = value.(string)
}

myval := MyValue{}
survey.AskOne(
    &survey.Input{
        Message: "Enter something:",
    },
    &myval
)
```

## Testing

You can test your program's interactive prompts using [go-expect](https://github.com/Netflix/go-expect). The library
can be used to expect a match on stdout and respond on stdin. Since `os.Stdout` in a `go test` process is not a TTY,
if you are manipulating the cursor or using `survey`, you will need a way to interpret terminal / ANSI escape sequences
for things like `CursorLocation`. `vt10x.NewVT10XConsole` will create a `go-expect` console that also multiplexes
stdio to an in-memory [virtual terminal](https://github.com/hinshun/vt10x).

For some examples, you can see any of the tests in this repo.

## FAQ

### What kinds of IO are supported by `survey`?

survey aims to support most terminal emulators; it expects support for ANSI escape sequences.
This means that reading from piped stdin or writing to piped stdout is **not supported**,
and likely to break your application in these situations. See [#337](https://github.com/AlecAivazis/survey/pull/337#issue-581351617)

### Why isn't Ctrl-C working?

Ordinarily, when you type Ctrl-C, the terminal recognizes this as the QUIT button and delivers a SIGINT signal to the process, which terminates it.
However, Survey temporarily configures the terminal to deliver control codes as ordinary input bytes.
When Survey reads a ^C byte (ASCII \x03, "end of text"), it interrupts the current survey and returns a
`github.com/AlecAivazis/survey/v2/terminal.InterruptErr` from `Ask` or `AskOne`.
If you want to stop the process, handle the returned error in your code:

```go
err := survey.AskOne(prompt, &myVar)
if err != nil {
	if err == terminal.InterruptErr {
		log.Fatal("interrupted")
	}
	...
}
```
