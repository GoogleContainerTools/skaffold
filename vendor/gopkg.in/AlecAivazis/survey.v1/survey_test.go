package survey

import (
	"fmt"
	"testing"
	"time"

	expect "github.com/Netflix/go-expect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/AlecAivazis/survey.v1/core"
	"gopkg.in/AlecAivazis/survey.v1/terminal"
)

func init() {
	// disable color output for all prompts to simplify testing
	core.DisableColor = true
}

func Stdio(c *expect.Console) terminal.Stdio {
	return terminal.Stdio{c.Tty(), c.Tty(), c.Tty()}
}

type PromptTest struct {
	name      string
	prompt    Prompt
	procedure func(*expect.Console)
	expected  interface{}
}

func RunPromptTest(t *testing.T, test PromptTest) {
	var answer interface{}
	RunTest(t, test.procedure, func(stdio terminal.Stdio) error {
		var err error
		if p, ok := test.prompt.(wantsStdio); ok {
			p.WithStdio(stdio)
		}
		answer, err = test.prompt.Prompt()
		return err
	})
	require.Equal(t, test.expected, answer)
}

func TestAsk(t *testing.T) {
	tests := []struct {
		name      string
		questions []*Question
		procedure func(*expect.Console)
		expected  map[string]interface{}
	}{
		{
			"Test Ask for all prompts",
			[]*Question{
				{
					Name: "pizza",
					Prompt: &Confirm{
						Message: "Is pizza your favorite food?",
					},
				},
				{
					Name: "commit-message",
					Prompt: &Editor{
						Message: "Edit git commit message",
					},
				},
				{
					Name: "name",
					Prompt: &Input{
						Message: "What is your name?",
					},
				},
				{
					Name: "day",
					Prompt: &MultiSelect{
						Message: "What days do you prefer:",
						Options: []string{"Sunday", "Monday", "Tuesday", "Wednesday", "Thursday", "Friday", "Saturday"},
					},
				},
				{
					Name: "password",
					Prompt: &Password{
						Message: "Please type your password",
					},
				},
				{
					Name: "color",
					Prompt: &Select{
						Message: "Choose a color:",
						Options: []string{"red", "blue", "green", "yellow"},
					},
				},
			},
			func(c *expect.Console) {
				// Confirm
				c.ExpectString("Is pizza your favorite food? (y/N)")
				c.SendLine("Y")

				// Editor
				c.ExpectString("Edit git commit message [Enter to launch editor]")
				c.SendLine("")
				go func() {
					time.Sleep(time.Millisecond)
					c.Send("iAdd editor prompt tests\x1b")
					c.SendLine(":wq!")
				}()

				// Input
				c.ExpectString("What is your name?")
				c.SendLine("Johnny Appleseed")

				// MultiSelect
				c.ExpectString("What days do you prefer:  [Use arrows to move, type to filter]")
				// Select Monday.
				c.Send(string(terminal.KeyArrowDown))
				c.Send(" ")
				// Select Wednesday.
				c.Send(string(terminal.KeyArrowDown))
				c.Send(string(terminal.KeyArrowDown))
				c.SendLine(" ")

				// Password
				c.ExpectString("Please type your password")
				c.Send("secret")
				c.SendLine("")

				// Select
				c.ExpectString("Choose a color:  [Use arrows to move, type to filter]")
				c.SendLine("yellow")
				c.ExpectEOF()
			},
			map[string]interface{}{
				"pizza":          true,
				"commit-message": "Add editor prompt tests\n",
				"name":           "Johnny Appleseed",
				"day":            []string{"Monday", "Wednesday"},
				"password":       "secret",
				"color":          "yellow",
			},
		},
		{
			"Test Ask with validate survey.Required",
			[]*Question{
				{
					Name: "name",
					Prompt: &Input{
						Message: "What is your name?",
					},
					Validate: Required,
				},
			},
			func(c *expect.Console) {
				c.ExpectString("What is your name?")
				c.SendLine("")
				c.ExpectString("Sorry, your reply was invalid: Value is required")
				c.SendLine("Johnny Appleseed")
				c.ExpectEOF()
			},
			map[string]interface{}{
				"name": "Johnny Appleseed",
			},
		},
		{
			"Test Ask with transformer survey.ToLower",
			[]*Question{
				{
					Name: "name",
					Prompt: &Input{
						Message: "What is your name?",
					},
					Transform: ToLower,
				},
			},
			func(c *expect.Console) {
				c.ExpectString("What is your name?")
				c.SendLine("Johnny Appleseed")
				c.ExpectEOF()
			},
			map[string]interface{}{
				"name": "johnny appleseed",
			},
		},
	}

	for _, test := range tests {
		// Capture range variable.
		test := test
		t.Run(test.name, func(t *testing.T) {
			answers := make(map[string]interface{})
			RunTest(t, test.procedure, func(stdio terminal.Stdio) error {
				return Ask(test.questions, &answers, WithStdio(stdio.In, stdio.Out, stdio.Err))
			})
			require.Equal(t, test.expected, answers)
		})
	}
}

func TestValidationError(t *testing.T) {

	err := fmt.Errorf("Football is not a valid month")

	actual, err := core.RunTemplate(
		core.ErrorTemplate,
		err,
	)
	if err != nil {
		t.Errorf("Failed to run template to format error: %s", err)
	}

	expected := `✘ Sorry, your reply was invalid: Football is not a valid month
`

	if actual != expected {
		t.Errorf("Formatted error was not formatted correctly. Found:\n%s\nExpected:\n%s", actual, expected)
	}
}

func TestAsk_returnsErrorIfTargetIsNil(t *testing.T) {
	// pass an empty place to leave the answers
	err := Ask([]*Question{}, nil)

	// if we didn't get an error
	if err == nil {
		// the test failed
		t.Error("Did not encounter error when asking with no where to record.")
	}
}

func TestPagination_tooFew(t *testing.T) {
	// a small list of options
	choices := []string{"choice1", "choice2", "choice3"}

	// a page bigger than the total number
	pageSize := 4
	// the current selection
	sel := 3

	// compute the page info
	page, idx := paginate(pageSize, choices, sel)

	// make sure we see the full list of options
	assert.Equal(t, choices, page)
	// with the second index highlighted (no change)
	assert.Equal(t, 3, idx)
}

func TestPagination_firstHalf(t *testing.T) {
	// the choices for the test
	choices := []string{"choice1", "choice2", "choice3", "choice4", "choice5", "choice6"}

	// section the choices into groups of 4 so the choice is somewhere in the middle
	// to verify there is no displacement of the page
	pageSize := 4
	// test the second item
	sel := 2

	// compute the page info
	page, idx := paginate(pageSize, choices, sel)

	// we should see the first three options
	assert.Equal(t, choices[0:4], page)
	// with the second index highlighted
	assert.Equal(t, 2, idx)
}

func TestPagination_middle(t *testing.T) {
	// the choices for the test
	choices := []string{"choice0", "choice1", "choice2", "choice3", "choice4", "choice5"}

	// section the choices into groups of 3
	pageSize := 2
	// test the second item so that we can verify we are in the middle of the list
	sel := 3

	// compute the page info
	page, idx := paginate(pageSize, choices, sel)

	// we should see the first three options
	assert.Equal(t, choices[2:4], page)
	// with the second index highlighted
	assert.Equal(t, 1, idx)
}

func TestPagination_lastHalf(t *testing.T) {
	// the choices for the test
	choices := []string{"choice0", "choice1", "choice2", "choice3", "choice4", "choice5"}

	// section the choices into groups of 3
	pageSize := 3
	// test the last item to verify we're not in the middle
	sel := 5

	// compute the page info
	page, idx := paginate(pageSize, choices, sel)

	// we should see the first three options
	assert.Equal(t, choices[3:6], page)
	// we should be at the bottom of the list
	assert.Equal(t, 2, idx)
}
