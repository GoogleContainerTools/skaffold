package survey

import (
	"bytes"
	"io"
	"os"
	"testing"

	expect "github.com/Netflix/go-expect"
	"github.com/stretchr/testify/assert"
	"gopkg.in/AlecAivazis/survey.v1/core"
	"gopkg.in/AlecAivazis/survey.v1/terminal"
)

func init() {
	// disable color output for all prompts to simplify testing
	core.DisableColor = true
}

func TestInputRender(t *testing.T) {

	tests := []struct {
		title    string
		prompt   Input
		data     InputTemplateData
		expected string
	}{
		{
			"Test Input question output without default",
			Input{Message: "What is your favorite month:"},
			InputTemplateData{},
			"? What is your favorite month: ",
		},
		{
			"Test Input question output with default",
			Input{Message: "What is your favorite month:", Default: "April"},
			InputTemplateData{},
			"? What is your favorite month: (April) ",
		},
		{
			"Test Input answer output",
			Input{Message: "What is your favorite month:"},
			InputTemplateData{Answer: "October", ShowAnswer: true},
			"? What is your favorite month: October\n",
		},
		{
			"Test Input question output without default but with help hidden",
			Input{Message: "What is your favorite month:", Help: "This is helpful"},
			InputTemplateData{},
			"? What is your favorite month: [? for help] ",
		},
		{
			"Test Input question output with default and with help hidden",
			Input{Message: "What is your favorite month:", Default: "April", Help: "This is helpful"},
			InputTemplateData{},
			"? What is your favorite month: [? for help] (April) ",
		},
		{
			"Test Input question output without default but with help shown",
			Input{Message: "What is your favorite month:", Help: "This is helpful"},
			InputTemplateData{ShowHelp: true},
			`ⓘ This is helpful
? What is your favorite month: `,
		},
		{
			"Test Input question output with default and with help shown",
			Input{Message: "What is your favorite month:", Default: "April", Help: "This is helpful"},
			InputTemplateData{ShowHelp: true},
			`ⓘ This is helpful
? What is your favorite month: (April) `,
		},
	}

	for _, test := range tests {
		r, w, err := os.Pipe()
		assert.Nil(t, err, test.title)

		test.prompt.WithStdio(terminal.Stdio{Out: w})
		test.data.Input = test.prompt
		err = test.prompt.Render(
			InputQuestionTemplate,
			test.data,
		)
		assert.Nil(t, err, test.title)

		w.Close()
		var buf bytes.Buffer
		io.Copy(&buf, r)

		assert.Contains(t, buf.String(), test.expected, test.title)
	}
}

func TestInputPrompt(t *testing.T) {
	tests := []PromptTest{
		{
			"Test Input prompt interaction",
			&Input{
				Message: "What is your name?",
			},
			func(c *expect.Console) {
				c.ExpectString("What is your name?")
				c.SendLine("Larry Bird")
				c.ExpectEOF()
			},
			"Larry Bird",
		},
		{
			"Test Input prompt interaction with default",
			&Input{
				Message: "What is your name?",
				Default: "Johnny Appleseed",
			},
			func(c *expect.Console) {
				c.ExpectString("What is your name?")
				c.SendLine("")
				c.ExpectEOF()
			},
			"Johnny Appleseed",
		},
		{
			"Test Input prompt interaction overriding default",
			&Input{
				Message: "What is your name?",
				Default: "Johnny Appleseed",
			},
			func(c *expect.Console) {
				c.ExpectString("What is your name?")
				c.SendLine("Larry Bird")
				c.ExpectEOF()
			},
			"Larry Bird",
		},
		{
			"Test Input prompt interaction and prompt for help",
			&Input{
				Message: "What is your name?",
				Help:    "It might be Satoshi Nakamoto",
			},
			func(c *expect.Console) {
				c.ExpectString("What is your name?")
				c.SendLine("?")
				c.ExpectString("It might be Satoshi Nakamoto")
				c.SendLine("Satoshi Nakamoto")
				c.ExpectEOF()
			},
			"Satoshi Nakamoto",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			RunPromptTest(t, test)
		})
	}
}
