//go:build windows || js || plan9
// +build windows js plan9

package copy

func preserveLtimes(src, dest string) error {
	return nil // Unsupported
}
