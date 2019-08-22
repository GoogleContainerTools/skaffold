// +build gofuzz

package docker

func Fuzz(data []byte) int {
	if _, err := ParseReference(string(data)); err != nil {
		return 0
	}
	return 1
}
