//go:build freebsd
// +build freebsd

package mint

// Exit ...
func (testee *Testee) Exit(expectedCode int) MintResult {
	panic("Exit method can NOT be used on FreeBSD, for now.")
	return MintResult{ok: false}
}
