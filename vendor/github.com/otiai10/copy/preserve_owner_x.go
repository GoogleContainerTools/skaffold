//go:build windows || plan9
// +build windows plan9

package copy

func preserveOwner(src, dest string, info fileInfo) (err error) {
	return nil
}
