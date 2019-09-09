package survey

import "strings"

var DefaultFilterFn = func(filter string, options []string) (answer []string) {
	filter = strings.ToLower(filter)
	for _, o := range options {
		if strings.Contains(strings.ToLower(o), filter) {
			answer = append(answer, o)
		}
	}
	return answer
}
