package survey

import (
	"bytes"
	"io"
	"os"
	"os/exec"
	"testing"
	"time"

	expect "github.com/Netflix/go-expect"
	"github.com/stretchr/testify/assert"
	"gopkg.in/AlecAivazis/survey.v1/core"
	"gopkg.in/AlecAivazis/survey.v1/terminal"
)

func init() {
	// disable color output for all prompts to simplify testing
	core.DisableColor = true
}

func TestEditorRender(t *testing.T) {
	tests := []struct {
		title    string
		prompt   Editor
		data     EditorTemplateData
		expected string
	}{
		{
			"Test Editor question output without default",
			Editor{Message: "What is your favorite month:"},
			EditorTemplateData{},
			"? What is your favorite month: [Enter to launch editor] ",
		},
		{
			"Test Editor question output with default",
			Editor{Message: "What is your favorite month:", Default: "April"},
			EditorTemplateData{},
			"? What is your favorite month: (April) [Enter to launch editor] ",
		},
		{
			"Test Editor question output with HideDefault",
			Editor{Message: "What is your favorite month:", Default: "April", HideDefault: true},
			EditorTemplateData{},
			"? What is your favorite month: [Enter to launch editor] ",
		},
		{
			"Test Editor answer output",
			Editor{Message: "What is your favorite month:"},
			EditorTemplateData{Answer: "October", ShowAnswer: true},
			"? What is your favorite month: October\n",
		},
		{
			"Test Editor question output without default but with help hidden",
			Editor{Message: "What is your favorite month:", Help: "This is helpful"},
			EditorTemplateData{},
			"? What is your favorite month: [? for help] [Enter to launch editor] ",
		},
		{
			"Test Editor question output with default and with help hidden",
			Editor{Message: "What is your favorite month:", Default: "April", Help: "This is helpful"},
			EditorTemplateData{},
			"? What is your favorite month: [? for help] (April) [Enter to launch editor] ",
		},
		{
			"Test Editor question output without default but with help shown",
			Editor{Message: "What is your favorite month:", Help: "This is helpful"},
			EditorTemplateData{ShowHelp: true},
			`ⓘ This is helpful
? What is your favorite month: [Enter to launch editor] `,
		},
		{
			"Test Editor question output with default and with help shown",
			Editor{Message: "What is your favorite month:", Default: "April", Help: "This is helpful"},
			EditorTemplateData{ShowHelp: true},
			`ⓘ This is helpful
? What is your favorite month: (April) [Enter to launch editor] `,
		},
	}

	for _, test := range tests {
		r, w, err := os.Pipe()
		assert.Nil(t, err, test.title)

		test.prompt.WithStdio(terminal.Stdio{Out: w})
		test.data.Editor = test.prompt
		err = test.prompt.Render(
			EditorQuestionTemplate,
			test.data,
		)
		assert.Nil(t, err, test.title)

		w.Close()
		var buf bytes.Buffer
		io.Copy(&buf, r)

		assert.Contains(t, buf.String(), test.expected, test.title)
	}
}

func TestEditorPrompt(t *testing.T) {
	if _, err := exec.LookPath("vi"); err != nil {
		t.Skip("vi not found in PATH")
	}

	tests := []PromptTest{
		{
			"Test Editor prompt interaction",
			&Editor{
				Editor:  "vi",
				Message: "Edit git commit message",
			},
			func(c *expect.Console) {
				c.ExpectString("Edit git commit message [Enter to launch editor]")
				c.SendLine("")
				go c.ExpectEOF()
				time.Sleep(time.Millisecond)
				c.Send("iAdd editor prompt tests\x1b")
				c.SendLine(":wq!")
			},
			"Add editor prompt tests\n",
		},
		{
			"Test Editor prompt interaction with default",
			&Editor{
				Editor:  "vi",
				Message: "Edit git commit message",
				Default: "No comment",
			},
			func(c *expect.Console) {
				c.ExpectString("Edit git commit message (No comment) [Enter to launch editor]")
				c.SendLine("")
				go c.ExpectEOF()
				time.Sleep(time.Millisecond)
				c.SendLine(":q!")
			},
			"No comment",
		},
		{
			"Test Editor prompt interaction overriding default",
			&Editor{
				Editor:  "vi",
				Message: "Edit git commit message",
				Default: "No comment",
			},
			func(c *expect.Console) {
				c.ExpectString("Edit git commit message (No comment) [Enter to launch editor]")
				c.SendLine("")
				go c.ExpectEOF()
				time.Sleep(time.Millisecond)
				c.Send("iAdd editor prompt tests\x1b")
				c.SendLine(":wq!")
			},
			"Add editor prompt tests\n",
		},
		{
			"Test Editor prompt interaction hiding default",
			&Editor{
				Editor:      "vi",
				Message:     "Edit git commit message",
				Default:     "No comment",
				HideDefault: true,
			},
			func(c *expect.Console) {
				c.ExpectString("Edit git commit message [Enter to launch editor]")
				c.SendLine("")
				go c.ExpectEOF()
				time.Sleep(time.Millisecond)
				c.SendLine(":q!")
			},
			"No comment",
		},
		{
			"Test Editor prompt interaction and prompt for help",
			&Editor{
				Editor:  "vi",
				Message: "Edit git commit message",
				Help:    "Describe your git commit",
			},
			func(c *expect.Console) {
				c.ExpectString("Edit git commit message [? for help] [Enter to launch editor]")
				c.SendLine("?")
				c.ExpectString("Describe your git commit")
				c.SendLine("")
				go c.ExpectEOF()
				time.Sleep(time.Millisecond)
				c.Send("iAdd editor prompt tests\x1b")
				c.SendLine(":wq!")
			},
			"Add editor prompt tests\n",
		},
		{
			"Test Editor prompt interaction with default and append default",
			&Editor{
				Editor:        "vi",
				Message:       "Edit git commit message",
				Default:       "No comment",
				AppendDefault: true,
			},
			func(c *expect.Console) {
				c.ExpectString("Edit git commit message (No comment) [Enter to launch editor]")
				c.SendLine("")
				c.ExpectString("No comment")
				c.SendLine("dd")
				c.SendLine(":wq!")
				c.ExpectEOF()
			},
			"",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			RunPromptTest(t, test)
		})
	}
}
