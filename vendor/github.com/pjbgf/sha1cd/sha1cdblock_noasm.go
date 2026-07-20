//go:build (!amd64 && !arm64) || noasm

package sha1cd

func block(dig *digest, p []byte) {
	blockGeneric(dig, p)
}
