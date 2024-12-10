package survey

import (
	"errors"
	"fmt"

	"github.com/AlecAivazis/survey/v2/core"
	"github.com/AlecAivazis/survey/v2/terminal"
)

/*
Select is a prompt that presents a list of various options to the user
for them to select using the arrow keys and enter. Response type is a string.

	color := ""
	prompt := &survey.Select{
		Message: "Choose a color:",
		Options: []string{"red", "blue", "green"},
	}
	survey.AskOne(prompt, &color)
*/
type Select struct {
	Renderer
	Message       string
	Options       []string
	Default       interface{}
	Help          string
	PageSize      int
	VimMode       bool
	FilterMessage string
	Filter        func(filter string, value string, index int) bool
	Description   func(value string, index int) string
	filter        string
	selectedIndex int
	showingHelp   bool
}

// SelectTemplateData is the data available to the templates when processing
type SelectTemplateData struct {
	Select
	PageEntries   []core.OptionAnswer
	SelectedIndex int
	Answer        string
	ShowAnswer    bool
	ShowHelp      bool
	Description   func(value string, index int) string
	Config        *PromptConfig

	// These fields are used when rendering an individual option
	CurrentOpt   core.OptionAnswer
	CurrentIndex int
}

// IterateOption sets CurrentOpt and CurrentIndex appropriately so a select option can be rendered individually
func (s SelectTemplateData) IterateOption(ix int, opt core.OptionAnswer) interface{} {
	copy := s
	copy.CurrentIndex = ix
	copy.CurrentOpt = opt
	return copy
}

func (s SelectTemplateData) GetDescription(opt core.OptionAnswer) string {
	if s.Description == nil {
		return ""
	}
	return s.Description(opt.Value, opt.Index)
}

var SelectQuestionTemplate = `
{{- define "option"}}
    {{- if eq .SelectedIndex .CurrentIndex }}{{color .Config.Icons.SelectFocus.Format }}{{ .Config.Icons.SelectFocus.Text }} {{else}}{{color "default"}}  {{end}}
    {{- .CurrentOpt.Value}}{{ if ne ($.GetDescription .CurrentOpt) "" }} - {{color "cyan"}}{{ $.GetDescription .CurrentOpt }}{{end}}
    {{- color "reset"}}
{{end}}
{{- if .ShowHelp }}{{- color .Config.Icons.Help.Format }}{{ .Config.Icons.Help.Text }} {{ .Help }}{{color "reset"}}{{"\n"}}{{end}}
{{- color .Config.Icons.Question.Format }}{{ .Config.Icons.Question.Text }} {{color "reset"}}
{{- color "default+hb"}}{{ .Message }}{{ .FilterMessage }}{{color "reset"}}
{{- if .ShowAnswer}}{{color "cyan"}} {{.Answer}}{{color "reset"}}{{"\n"}}
{{- else}}
  {{- "  "}}{{- color "cyan"}}[Use arrows to move, type to filter{{- if and .Help (not .ShowHelp)}}, {{ .Config.HelpInput }} for more help{{end}}]{{color "reset"}}
  {{- "\n"}}
  {{- range $ix, $option := .PageEntries}}
    {{- template "option" $.IterateOption $ix $option}}
  {{- end}}
{{- end}}`

// OnChange is called on every keypress.
func (s *Select) OnChange(key rune, config *PromptConfig) bool {
	options := s.filterOptions(config)
	oldFilter := s.filter

	// if the user pressed the enter key and the index is a valid option
	if key == terminal.KeyEnter || key == '\n' {
		// if the selected index is a valid option
		if len(options) > 0 && s.selectedIndex < len(options) {

			// we're done (stop prompting the user)
			return true
		}

		// we're not done (keep prompting)
		return false

		// if the user pressed the up arrow or 'k' to emulate vim
	} else if (key == terminal.KeyArrowUp || (s.VimMode && key == 'k')) && len(options) > 0 {
		// if we are at the top of the list
		if s.selectedIndex == 0 {
			// start from the button
			s.selectedIndex = len(options) - 1
		} else {
			// otherwise we are not at the top of the list so decrement the selected index
			s.selectedIndex--
		}

		// if the user pressed down or 'j' to emulate vim
	} else if (key == terminal.KeyTab || key == terminal.KeyArrowDown || (s.VimMode && key == 'j')) && len(options) > 0 {
		// if we are at the bottom of the list
		if s.selectedIndex == len(options)-1 {
			// start from the top
			s.selectedIndex = 0
		} else {
			// increment the selected index
			s.selectedIndex++
		}
		// only show the help message if we have one
	} else if string(key) == config.HelpInput && s.Help != "" {
		s.showingHelp = true
		// if the user wants to toggle vim mode on/off
	} else if key == terminal.KeyEscape {
		s.VimMode = !s.VimMode
		// if the user hits any of the keys that clear the filter
	} else if key == terminal.KeyDeleteWord || key == terminal.KeyDeleteLine {
		s.filter = ""
		// if the user is deleting a character in the filter
	} else if key == terminal.KeyDelete || key == terminal.KeyBackspace {
		// if there is content in the filter to delete
		if s.filter != "" {
			runeFilter := []rune(s.filter)
			// subtract a line from the current filter
			s.filter = string(runeFilter[0 : len(runeFilter)-1])
			// we removed the last value in the filter
		}
	} else if key >= terminal.KeySpace {
		s.filter += string(key)
		// make sure vim mode is disabled
		s.VimMode = false
	}

	s.FilterMessage = ""
	if s.filter != "" {
		s.FilterMessage = " " + s.filter
	}
	if oldFilter != s.filter {
		// filter changed
		options = s.filterOptions(config)
		if len(options) > 0 && len(options) <= s.selectedIndex {
			s.selectedIndex = len(options) - 1
		}
	}

	// figure out the options and index to render
	// figure out the page size
	pageSize := s.PageSize
	// if we dont have a specific one
	if pageSize == 0 {
		// grab the global value
		pageSize = config.PageSize
	}

	// TODO if we have started filtering and were looking at the end of a list
	// and we have modified the filter then we should move the page back!
	opts, idx := paginate(pageSize, options, s.selectedIndex)

	tmplData := SelectTemplateData{
		Select:        *s,
		SelectedIndex: idx,
		ShowHelp:      s.showingHelp,
		Description:   s.Description,
		PageEntries:   opts,
		Config:        config,
	}

	// render the options
	_ = s.RenderWithCursorOffset(SelectQuestionTemplate, tmplData, opts, idx)

	// keep prompting
	return false
}

func (s *Select) filterOptions(config *PromptConfig) []core.OptionAnswer {
	// the filtered list
	answers := []core.OptionAnswer{}

	// if there is no filter applied
	if s.filter == "" {
		return core.OptionAnswerList(s.Options)
	}

	// the filter to apply
	filter := s.Filter
	if filter == nil {
		filter = config.Filter
	}

	for i, opt := range s.Options {
		// i the filter says to include the option
		if filter(s.filter, opt, i) {
			answers = append(answers, core.OptionAnswer{
				Index: i,
				Value: opt,
			})
		}
	}

	// return the list of answers
	return answers
}

func (s *Select) Prompt(config *PromptConfig) (interface{}, error) {
	// if there are no options to render
	if len(s.Options) == 0 {
		// we failed
		return "", errors.New("please provide options to select from")
	}

	s.selectedIndex = 0
	if s.Default != nil {
		switch defaultValue := s.Default.(type) {
		case string:
			var found bool
			for i, opt := range s.Options {
				if opt == defaultValue {
					s.selectedIndex = i
					found = true
				}
			}
			if !found {
				return "", fmt.Errorf("default value %q not found in options", defaultValue)
			}
		case int:
			if defaultValue >= len(s.Options) {
				return "", fmt.Errorf("default index %d exceeds the number of options", defaultValue)
			}
			s.selectedIndex = defaultValue
		default:
			return "", errors.New("default value of select must be an int or string")
		}
	}

	// figure out the page size
	pageSize := s.PageSize
	// if we dont have a specific one
	if pageSize == 0 {
		// grab the global value
		pageSize = config.PageSize
	}

	// figure out the options and index to render
	opts, idx := paginate(pageSize, core.OptionAnswerList(s.Options), s.selectedIndex)

	cursor := s.NewCursor()
	cursor.Save()          // for proper cursor placement during selection
	cursor.Hide()          // hide the cursor
	defer cursor.Show()    // show the cursor when we're done
	defer cursor.Restore() // clear any accessibility offsetting on exit

	tmplData := SelectTemplateData{
		Select:        *s,
		SelectedIndex: idx,
		Description:   s.Description,
		ShowHelp:      s.showingHelp,
		PageEntries:   opts,
		Config:        config,
	}

	// ask the question
	err := s.RenderWithCursorOffset(SelectQuestionTemplate, tmplData, opts, idx)
	if err != nil {
		return "", err
	}

	rr := s.NewRuneReader()
	_ = rr.SetTermMode()
	defer func() {
		_ = rr.RestoreTermMode()
	}()

	// start waiting for input
	for {
		r, _, err := rr.ReadRune()
		if err != nil {
			return "", err
		}
		if r == terminal.KeyInterrupt {
			return "", terminal.InterruptErr
		}
		if r == terminal.KeyEndTransmission {
			break
		}
		if s.OnChange(r, config) {
			break
		}
	}

	options := s.filterOptions(config)
	s.filter = ""
	s.FilterMessage = ""

	if s.selectedIndex < len(options) {
		return options[s.selectedIndex], err
	}

	return options[0], err
}

func (s *Select) Cleanup(config *PromptConfig, val interface{}) error {
	cursor := s.NewCursor()
	cursor.Restore()
	return s.Render(
		SelectQuestionTemplate,
		SelectTemplateData{
			Select:      *s,
			Answer:      val.(core.OptionAnswer).Value,
			ShowAnswer:  true,
			Description: s.Description,
			Config:      config,
		},
	)
}
