package phase

import (
	"strings"
)

func TruncateSha(sha string) string {
	rawSha := strings.TrimPrefix(sha, "sha256:")
	if len(sha) > 12 {
		return rawSha[0:12]
	}
	return rawSha
}

func removeStagePrefixes(mixins []string) []string {
	var result []string
	for _, m := range mixins {
		s := strings.SplitN(m, ":", 2)
		if len(s) == 1 {
			result = append(result, s[0])
		} else {
			result = append(result, s[1])
		}
	}
	return result
}
