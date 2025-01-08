package survey

import (
	"fmt"
	"strings"

	"github.com/AlecAivazis/survey/v2/core"
	"github.com/AlecAivazis/survey/v2/terminal"
)

/*
Password is like a normal Input but the text shows up as *'s and there is no default. Response
type is a string.

	password := ""
	prompt := &survey.Password{ Message: "Please type your password" }
	survey.AskOne(prompt, &password)
*/
type Password struct {
	Renderer
	Message string
	Help    string
}

type PasswordTemplateData struct {
	Password
	ShowHelp bool
	Config   *PromptConfig
}

// PasswordQuestionTemplate is a template with color formatting. See Documentation: https://github.com/mgutz/ansi#style-format
var PasswordQuestionTemplate = `
{{- if .ShowHelp }}{{- color .Config.Icons.Help.Format }}{{ .Config.Icons.Help.Text }} {{ .Help }}{{color "reset"}}{{"\n"}}{{end}}
{{- color .Config.Icons.Question.Format }}{{ .Config.Icons.Question.Text }} {{color "reset"}}
{{- color "default+hb"}}{{ .Message }} {{color "reset"}}
{{- if and .Help (not .ShowHelp)}}{{color "cyan"}}[{{ .Config.HelpInput }} for help]{{color "reset"}} {{end}}`

func (p *Password) Prompt(config *PromptConfig) (interface{}, error) {
	// render the question template
	userOut, _, err := core.RunTemplate(
		PasswordQuestionTemplate,
		PasswordTemplateData{
			Password: *p,
			Config:   config,
		},
	)
	if err != nil {
		return "", err
	}

	if _, err := fmt.Fprint(terminal.NewAnsiStdout(p.Stdio().Out), userOut); err != nil {
		return "", err
	}

	rr := p.NewRuneReader()
	_ = rr.SetTermMode()
	defer func() {
		_ = rr.RestoreTermMode()
	}()

	// no help msg?  Just return any response
	if p.Help == "" {
		line, err := rr.ReadLine(config.HideCharacter)
		return string(line), err
	}

	cursor := p.NewCursor()

	var line []rune
	// process answers looking for help prompt answer
	for {
		line, err = rr.ReadLine(config.HideCharacter)
		if err != nil {
			return string(line), err
		}

		if string(line) == config.HelpInput {
			// terminal will echo the \n so we need to jump back up one row
			cursor.PreviousLine(1)

			err = p.Render(
				PasswordQuestionTemplate,
				PasswordTemplateData{
					Password: *p,
					ShowHelp: true,
					Config:   config,
				},
			)
			if err != nil {
				return "", err
			}
			continue
		}

		break
	}

	lineStr := string(line)
	p.AppendRenderedText(strings.Repeat(string(config.HideCharacter), len(lineStr)))
	return lineStr, err
}

// Cleanup hides the string with a fixed number of characters.
func (prompt *Password) Cleanup(config *PromptConfig, val interface{}) error {
	return nil
}
