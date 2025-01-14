package mint

// MintResult provide the results of assertion
// for `Dry` option.
type MintResult struct {
	ok      bool
	message string
}

// OK returns whether result is ok or not.
func (r MintResult) OK() bool {
	return r.ok
}

// NG is the opposite alias for OK().
func (r MintResult) NG() bool {
	return !r.ok
}

// Message returns failure message.
func (r MintResult) Message() string {
	return r.message
}
