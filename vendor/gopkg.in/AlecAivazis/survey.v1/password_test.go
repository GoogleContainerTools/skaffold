package survey

import (
	"testing"

	expect "github.com/Netflix/go-expect"
	"github.com/stretchr/testify/assert"
	"gopkg.in/AlecAivazis/survey.v1/core"
)

func init() {
	// disable color output for all prompts to simplify testing
	core.DisableColor = true
}

func TestPasswordRender(t *testing.T) {

	tests := []struct {
		title    string
		prompt   Password
		data     PasswordTemplateData
		expected string
	}{
		{
			"Test Password question output",
			Password{Message: "Tell me your secret:"},
			PasswordTemplateData{},
			"? Tell me your secret: ",
		},
		{
			"Test Password question output with help hidden",
			Password{Message: "Tell me your secret:", Help: "This is helpful"},
			PasswordTemplateData{},
			"? Tell me your secret: [? for help] ",
		},
		{
			"Test Password question output with help shown",
			Password{Message: "Tell me your secret:", Help: "This is helpful"},
			PasswordTemplateData{ShowHelp: true},
			`ⓘ This is helpful
? Tell me your secret: `,
		},
	}

	for _, test := range tests {
		test.data.Password = test.prompt
		actual, err := core.RunTemplate(
			PasswordQuestionTemplate,
			&test.data,
		)
		assert.Nil(t, err, test.title)
		assert.Equal(t, test.expected, actual, test.title)
	}
}

func TestPasswordPrompt(t *testing.T) {
	tests := []PromptTest{
		{
			"Test Password prompt interaction",
			&Password{
				Message: "Please type your password",
			},
			func(c *expect.Console) {
				c.ExpectString("Please type your password")
				c.Send("secret")
				c.SendLine("")
				c.ExpectEOF()
			},
			"secret",
		},
		{
			"Test Password prompt interaction with help",
			&Password{
				Message: "Please type your password",
				Help:    "It's a secret",
			},
			func(c *expect.Console) {
				c.ExpectString("Please type your password")
				c.SendLine("?")
				c.ExpectString("It's a secret")
				c.Send("secret")
				c.SendLine("")
				c.ExpectEOF()
			},
			"secret",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			RunPromptTest(t, test)
		})
	}
}
