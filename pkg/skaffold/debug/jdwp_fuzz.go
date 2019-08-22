// +build gofuzz

package debug

func Fuzz(data []byte) int {
	parseJdwpSpec(string(data))
	return 1
}
