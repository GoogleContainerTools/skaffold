//go:build (!amd64 || !arm64) && !noasm
// +build !amd64 !arm64
// +build !noasm

package sha1cd

type sliceHeader struct {
	base uintptr
	len  int
	cap  int
}
