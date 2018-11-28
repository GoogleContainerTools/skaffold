package survey

import (
	"testing"

	expect "github.com/Netflix/go-expect"
	"gopkg.in/AlecAivazis/survey.v1/terminal"
)

func RunTest(t *testing.T, procedure func(*expect.Console), test func(terminal.Stdio) error) {
	t.Skip("Windows does not support psuedoterminals")
}
