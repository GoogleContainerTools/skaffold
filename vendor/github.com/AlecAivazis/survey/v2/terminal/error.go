package terminal

import (
	"errors"
)

var (
	//lint:ignore ST1012 keeping old name for backwards compatibility
	InterruptErr = errors.New("interrupt")
)
