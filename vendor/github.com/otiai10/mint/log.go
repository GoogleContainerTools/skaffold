package mint

import (
	"fmt"
	"os"
)

// Log only output if -v flag is given.
// This is because the standard "t.Testing.Log" method decorates
// its caller: runtime.Caller(3) automatically.
func Log(args ...interface{}) {
	if isVerbose(os.Args) {
		fmt.Print(args...)
	}
}
