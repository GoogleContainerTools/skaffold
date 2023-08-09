package survey

import (
	"errors"

	"github.com/AlecAivazis/survey/v2/core"
	"github.com/AlecAivazis/survey/v2/terminal"
)

/*
Input is a regular text input that prints each character the user types on the screen
and accepts the input with the enter key. Response type is a string.

	name := ""
	prompt := &survey.Input{ Message: "What is your name?" }
	survey.AskOne(prompt, &name)
*/
type Input struct {
	Renderer
	Message       string
	Default       string
	Help          string
	Suggest       func(toComplete string) []string
	answer        string
	typedAnswer   string
	options       []core.OptionAnswer
	selectedIndex int
	showingHelp   bool
}

// data available to the templates when processing
type InputTemplateData struct {
	Input
	ShowAnswer    bool
	ShowHelp      bool
	Answer        string
	PageEntries   []core.OptionAnswer
	SelectedIndex int
	Config        *PromptConfig
}

// Templates with Color formatting. See Documentation: https://github.com/mgutz/ansi#style-format
var InputQuestionTemplate = `
{{- if .ShowHelp }}{{- color .Config.Icons.Help.Format }}{{ .Config.Icons.Help.Text }} {{ .Help }}{{color "reset"}}{{"\n"}}{{end}}
{{- color .Config.Icons.Question.Format }}{{ .Config.Icons.Question.Text }} {{color "reset"}}
{{- color "default+hb"}}{{ .Message }} {{color "reset"}}
{{- if .ShowAnswer}}
  {{- color "cyan"}}{{.Answer}}{{color "reset"}}{{"\n"}}
{{- else if .PageEntries -}}
  {{- .Answer}} [Use arrows to move, enter to select, type to continue]
  {{- "\n"}}
  {{- range $ix, $choice := .PageEntries}}
    {{- if eq $ix $.SelectedIndex }}{{color $.Config.Icons.SelectFocus.Format }}{{ $.Config.Icons.SelectFocus.Text }} {{else}}{{color "default"}}  {{end}}
    {{- $choice.Value}}
    {{- color "reset"}}{{"\n"}}
  {{- end}}
{{- else }}
  {{- if or (and .Help (not .ShowHelp)) .Suggest }}{{color "cyan"}}[
    {{- if and .Help (not .ShowHelp)}}{{ print .Config.HelpInput }} for help {{- if and .Suggest}}, {{end}}{{end -}}
    {{- if and .Suggest }}{{color "cyan"}}{{ print .Config.SuggestInput }} for suggestions{{end -}}
  ]{{color "reset"}} {{end}}
  {{- if .Default}}{{color "white"}}({{.Default}}) {{color "reset"}}{{end}}
{{- end}}`

func (i *Input) onRune(config *PromptConfig) terminal.OnRuneFn {
	return terminal.OnRuneFn(func(key rune, line []rune) ([]rune, bool, error) {
		if i.options != nil && (key == terminal.KeyEnter || key == '\n') {
			return []rune(i.answer), true, nil
		} else if i.options != nil && key == terminal.KeyEscape {
			i.answer = i.typedAnswer
			i.options = nil
		} else if key == terminal.KeyArrowUp && len(i.options) > 0 {
			if i.selectedIndex == 0 {
				i.selectedIndex = len(i.options) - 1
			} else {
				i.selectedIndex--
			}
			i.answer = i.options[i.selectedIndex].Value
		} else if (key == terminal.KeyArrowDown || key == terminal.KeyTab) && len(i.options) > 0 {
			if i.selectedIndex == len(i.options)-1 {
				i.selectedIndex = 0
			} else {
				i.selectedIndex++
			}
			i.answer = i.options[i.selectedIndex].Value
		} else if key == terminal.KeyTab && i.Suggest != nil {
			i.answer = string(line)
			i.typedAnswer = i.answer
			options := i.Suggest(i.answer)
			i.selectedIndex = 0
			if len(options) == 0 {
				return line, false, nil
			}

			i.answer = options[0]
			if len(options) == 1 {
				i.typedAnswer = i.answer
				i.options = nil
			} else {
				i.options = core.OptionAnswerList(options)
			}
		} else {
			if i.options == nil {
				return line, false, nil
			}

			if key >= terminal.KeySpace {
				i.answer += string(key)
			}
			i.typedAnswer = i.answer

			i.options = nil
		}

		pageSize := config.PageSize
		opts, idx := paginate(pageSize, i.options, i.selectedIndex)
		err := i.Render(
			InputQuestionTemplate,
			InputTemplateData{
				Input:         *i,
				Answer:        i.answer,
				ShowHelp:      i.showingHelp,
				SelectedIndex: idx,
				PageEntries:   opts,
				Config:        config,
			},
		)

		if err == nil {
			err = readLineAgain
		}

		return []rune(i.typedAnswer), true, err
	})
}

var readLineAgain = errors.New("read line again")

func (i *Input) Prompt(config *PromptConfig) (interface{}, error) {
	// render the template
	err := i.Render(
		InputQuestionTemplate,
		InputTemplateData{
			Input:    *i,
			Config:   config,
			ShowHelp: i.showingHelp,
		},
	)
	if err != nil {
		return "", err
	}

	// start reading runes from the standard in
	rr := i.NewRuneReader()
	rr.SetTermMode()
	defer rr.RestoreTermMode()

	cursor := i.NewCursor()
	if !config.ShowCursor {
		cursor.Hide()       // hide the cursor
		defer cursor.Show() // show the cursor when we're done
	}

	var line []rune

	for {
		if i.options != nil {
			line = []rune{}
		}

		line, err = rr.ReadLineWithDefault(0, line, i.onRune(config))
		if err == readLineAgain {
			continue
		}

		if err != nil {
			return "", err
		}

		break
	}

	i.answer = string(line)
	// readline print an empty line, go up before we render the follow up
	cursor.Up(1)

	// if we ran into the help string
	if i.answer == config.HelpInput && i.Help != "" {
		// show the help and prompt again
		i.showingHelp = true
		return i.Prompt(config)
	}

	// if the line is empty
	if len(i.answer) == 0 {
		// use the default value
		return i.Default, err
	}

	lineStr := i.answer

	i.AppendRenderedText(lineStr)

	// we're done
	return lineStr, err
}

func (i *Input) Cleanup(config *PromptConfig, val interface{}) error {
	// use the default answer when cleaning up the prompt if necessary
	ans := i.answer
	if ans == "" && i.Default != "" {
		ans = i.Default
	}

	// render the cleanup
	return i.Render(
		InputQuestionTemplate,
		InputTemplateData{
			Input:      *i,
			ShowAnswer: true,
			Config:     config,
			Answer:     ans,
		},
	)
}
