package mint

import "testing"

// Because is context printer.
func Because(t *testing.T, context string, wrapper func(*testing.T)) {
	Log("  Because ", context, "\n")
	wrapper(t)
}

// When is an alternative of `Because`
func When(t *testing.T, context string, wrapper func(*testing.T)) {
	Log("  When ", context, "\n")
	wrapper(t)
}
