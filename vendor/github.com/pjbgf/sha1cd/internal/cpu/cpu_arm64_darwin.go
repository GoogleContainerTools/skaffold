//go:build !noasm && gc && arm64 && darwin

package cpu

// Every Apple arm64 chip implements the ARMv8 Cryptographic Extension. The Go
// runtime makes the same assumption for crypto/sha1 on darwin/arm64.
func init() {
	ARM64.HasSHA1 = true
}
