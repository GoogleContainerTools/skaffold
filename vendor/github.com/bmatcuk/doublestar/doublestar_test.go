// This file is mostly copied from Go's path/match_test.go

package doublestar

import (
	"os"
	"path"
	"path/filepath"
	"runtime"
	"testing"
)

type MatchTest struct {
	pattern, testPath []string // a pattern and path to test the pattern on
	shouldMatch       bool     // true if the pattern should match the path
	expectedErr       error    // an expected error
	testOnDisk        bool     // true: test pattern against files in "test" directory
}

// Tests which contain escapes and symlinks will not work on Windows
var onWindows = runtime.GOOS == "windows"

var matchTests = []MatchTest{
	{[]string{"abc"}, []string{"abc"}, true, nil, true},
	{[]string{"*"}, []string{"abc"}, true, nil, true},
	{[]string{"*c"}, []string{"abc"}, true, nil, true},
	{[]string{"a*"}, []string{"a"}, true, nil, true},
	{[]string{"a*"}, []string{"abc"}, true, nil, true},
	{[]string{"a*"}, []string{"ab", "c"}, false, nil, true},
	{[]string{"a*", "b"}, []string{"abc", "b"}, true, nil, true},
	{[]string{"a*", "b"}, []string{"a", "c", "b"}, false, nil, true},
	{[]string{"a*b*c*d*e*", "f"}, []string{"axbxcxdxe", "f"}, true, nil, true},
	{[]string{"a*b*c*d*e*", "f"}, []string{"axbxcxdxexxx", "f"}, true, nil, true},
	{[]string{"a*b*c*d*e*", "f"}, []string{"axbxcxdxe", "xxx", "f"}, false, nil, true},
	{[]string{"a*b*c*d*e*", "f"}, []string{"axbxcxdxexxx", "fff"}, false, nil, true},
	{[]string{"a*b?c*x"}, []string{"abxbbxdbxebxczzx"}, true, nil, true},
	{[]string{"a*b?c*x"}, []string{"abxbbxdbxebxczzy"}, false, nil, true},
	{[]string{"ab[c]"}, []string{"abc"}, true, nil, true},
	{[]string{"ab[b-d]"}, []string{"abc"}, true, nil, true},
	{[]string{"ab[e-g]"}, []string{"abc"}, false, nil, true},
	{[]string{"ab[^c]"}, []string{"abc"}, false, nil, true},
	{[]string{"ab[^b-d]"}, []string{"abc"}, false, nil, true},
	{[]string{"ab[^e-g]"}, []string{"abc"}, true, nil, true},
	{[]string{"a\\*b"}, []string{"ab"}, false, nil, true},
	{[]string{"a?b"}, []string{"a☺b"}, true, nil, true},
	{[]string{"a[^a]b"}, []string{"a☺b"}, true, nil, true},
	{[]string{"a???b"}, []string{"a☺b"}, false, nil, true},
	{[]string{"a[^a][^a][^a]b"}, []string{"a☺b"}, false, nil, true},
	{[]string{"[a-ζ]*"}, []string{"α"}, true, nil, true},
	{[]string{"*[a-ζ]"}, []string{"A"}, false, nil, true},
	{[]string{"a?b"}, []string{"a", "b"}, false, nil, true},
	{[]string{"a*b"}, []string{"a", "b"}, false, nil, true},
	{[]string{"[\\]a]"}, []string{"]"}, true, nil, !onWindows},
	{[]string{"[\\-]"}, []string{"-"}, true, nil, !onWindows},
	{[]string{"[x\\-]"}, []string{"x"}, true, nil, !onWindows},
	{[]string{"[x\\-]"}, []string{"-"}, true, nil, !onWindows},
	{[]string{"[x\\-]"}, []string{"z"}, false, nil, !onWindows},
	{[]string{"[\\-x]"}, []string{"x"}, true, nil, !onWindows},
	{[]string{"[\\-x]"}, []string{"-"}, true, nil, !onWindows},
	{[]string{"[\\-x]"}, []string{"a"}, false, nil, !onWindows},
	{[]string{"[]a]"}, []string{"]"}, false, ErrBadPattern, true},
	{[]string{"[-]"}, []string{"-"}, false, ErrBadPattern, true},
	{[]string{"[x-]"}, []string{"x"}, false, ErrBadPattern, true},
	{[]string{"[x-]"}, []string{"-"}, false, ErrBadPattern, true},
	{[]string{"[x-]"}, []string{"z"}, false, ErrBadPattern, true},
	{[]string{"[-x]"}, []string{"x"}, false, ErrBadPattern, true},
	{[]string{"[-x]"}, []string{"-"}, false, ErrBadPattern, true},
	{[]string{"[-x]"}, []string{"a"}, false, ErrBadPattern, true},
	{[]string{"\\"}, []string{"a"}, false, ErrBadPattern, !onWindows},
	{[]string{"[a-b-c]"}, []string{"a"}, false, ErrBadPattern, true},
	{[]string{"["}, []string{"a"}, false, ErrBadPattern, true},
	{[]string{"[^"}, []string{"a"}, false, ErrBadPattern, true},
	{[]string{"[^bc"}, []string{"a"}, false, ErrBadPattern, true},
	{[]string{"a["}, []string{"a"}, false, nil, false},
	{[]string{"a["}, []string{"ab"}, false, ErrBadPattern, true},
	{[]string{"*x"}, []string{"xxx"}, true, nil, true},
	{[]string{"[abc]"}, []string{"b"}, true, nil, true},
	{[]string{"a", "**"}, []string{"a"}, false, nil, true},
	{[]string{"a", "**"}, []string{"a", "b"}, true, nil, true},
	{[]string{"a", "**"}, []string{"a", "b", "c"}, true, nil, true},
	{[]string{"**", "c"}, []string{"c"}, true, nil, true},
	{[]string{"**", "c"}, []string{"b", "c"}, true, nil, true},
	{[]string{"**", "c"}, []string{"a", "b", "c"}, true, nil, true},
	{[]string{"**", "c"}, []string{"a", "b"}, false, nil, true},
	{[]string{"**", "c"}, []string{"abcd"}, false, nil, true},
	{[]string{"**", "c"}, []string{"a", "abc"}, false, nil, true},
	{[]string{"a", "**", "b"}, []string{"a", "b"}, true, nil, true},
	{[]string{"a", "**", "c"}, []string{"a", "b", "c"}, true, nil, true},
	{[]string{"a", "**", "d"}, []string{"a", "b", "c", "d"}, true, nil, true},
	{[]string{"a", "\\**"}, []string{"a", "b", "c"}, false, nil, !onWindows},
	{[]string{"ab{c,d}"}, []string{"abc"}, true, nil, true},
	{[]string{"ab{c,d,*}"}, []string{"abcde"}, true, nil, true},
	{[]string{"ab{c,d}["}, []string{"abcd"}, false, ErrBadPattern, true},
	{[]string{"abc", "**"}, []string{"abc", "b"}, true, nil, true},
	{[]string{"**", "abc"}, []string{"abc"}, true, nil, true},
	{[]string{"abc**"}, []string{"abc", "b"}, false, nil, true},
	{[]string{"broken-symlink"}, []string{"broken-symlink"}, true, nil, !onWindows},
	{[]string{"working-symlink", "c", "*"}, []string{"working-symlink", "c", "d"}, true, nil, !onWindows},
	{[]string{"working-sym*", "*"}, []string{"working-symlink", "c"}, true, nil, !onWindows},
	{[]string{"b", "**", "f"}, []string{"b", "symlink-dir", "f"}, true, nil, !onWindows},
}

func TestMatch(t *testing.T) {
	for idx, tt := range matchTests {
		// Since Match() always uses "/" as the separator, we
		// don't need to worry about the tt.testOnDisk flag
		testMatchWith(t, idx, tt)
	}
}

func testMatchWith(t *testing.T, idx int, tt MatchTest) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("#%v. Match(%#q, %#q) panicked: %#v", idx, tt.pattern, tt.testPath, r)
		}
	}()

	// Match() always uses "/" as the separator
	ok, err := Match(path.Join(tt.pattern...), path.Join(tt.testPath...))
	if ok != tt.shouldMatch || err != tt.expectedErr {
		t.Errorf("#%v. Match(%#q, %#q) = %v, %v want %v, %v", idx, tt.pattern, tt.testPath, ok, err, tt.shouldMatch, tt.expectedErr)
	}
}

func TestPathMatch(t *testing.T) {
	for idx, tt := range matchTests {
		// Even though we aren't actually matching paths on disk, we are using
		// PathMatch() which will use the system's separator. As a result, any
		// patterns that might cause problems on-disk need to also be avoided
		// here in this test.
		if tt.testOnDisk {
			testPathMatchWith(t, idx, tt)
		}
	}
}

func testPathMatchWith(t *testing.T, idx int, tt MatchTest) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("#%v. Match(%#q, %#q) panicked: %#v", idx, tt.pattern, tt.testPath, r)
		}
	}()

	ok, err := PathMatch(filepath.Join(tt.pattern...), filepath.Join(tt.testPath...))
	if ok != tt.shouldMatch || err != tt.expectedErr {
		t.Errorf("#%v. Match(%#q, %#q) = %v, %v want %v, %v", idx, tt.pattern, tt.testPath, ok, err, tt.shouldMatch, tt.expectedErr)
	}
}

func TestGlob(t *testing.T) {
	abspath, err := os.Getwd()
	if err != nil {
		t.Errorf("Error getting current working directory: %v", err)
		return
	}

	for idx, tt := range matchTests {
		if tt.testOnDisk {
			// test both relative paths and absolute paths
			testGlobWith(t, idx, tt, "")
			testGlobWith(t, idx, tt, abspath)
		}
	}
}

func testGlobWith(t *testing.T, idx int, tt MatchTest, basepath string) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("#%v. Glob(%#q) panicked: %#v", idx, tt.pattern, r)
		}
	}()

	pattern := filepath.Join(basepath, "test", filepath.Join(tt.pattern...))
	testPath := filepath.Join(basepath, "test", filepath.Join(tt.testPath...))
	matches, err := Glob(pattern)
	if inSlice(testPath, matches) != tt.shouldMatch {
		if tt.shouldMatch {
			t.Errorf("#%v. Glob(%#q) = %#v - doesn't contain %v, but should", idx, tt.pattern, matches, tt.testPath)
		} else {
			t.Errorf("#%v. Glob(%#q) = %#v - contains %v, but shouldn't", idx, tt.pattern, matches, tt.testPath)
		}
	}
	if err != tt.expectedErr {
		t.Errorf("#%v. Glob(%#q) has error %v, but should be %v", idx, tt.pattern, err, tt.expectedErr)
	}
}

func inSlice(s string, a []string) bool {
	for _, i := range a {
		if i == s {
			return true
		}
	}
	return false
}
