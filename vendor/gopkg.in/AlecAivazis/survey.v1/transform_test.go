package survey

import (
	"strings"
	"testing"
)

func testStringTransformer(t *testing.T, f func(string) string) {
	transformer := TransformString(f)

	tests := []string{
		"hello my name is",
		"where are you from",
		"does that matter?",
	}

	for _, tt := range tests {
		if expected, got := f(tt), transformer(tt); expected != got {
			t.Errorf("TransformString transformer failed to transform the answer, expected '%s' but got '%s'.", expected, got)
		}
	}
}

func TestTransformString(t *testing.T) {
	testStringTransformer(t, strings.ToTitle) // all letters titled
	testStringTransformer(t, strings.ToLower) // all letters lowercase
}

func TestComposeTransformers(t *testing.T) {
	// create a transformer which makes no sense,
	// remember: transformer can be used for any type
	// we just test the built'n functions that
	// happens to be for strings only.
	transformer := ComposeTransformers(
		Title,
		ToLower,
	)

	ans := "my name is"
	if expected, got := strings.ToLower(ans), transformer(ans); expected != got {
		// the result should be lowercase.
		t.Errorf("TestComposeTransformers transformer failed to transform the answer to title->lowercase, expected '%s' but got '%s'.", expected, got)
	}
}
