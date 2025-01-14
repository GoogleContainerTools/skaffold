// Package stack provides helpers for querying the callstack.
package stack

import (
	"path"
	"runtime"
	"strings"
)

// Frames returns at most max callstack Frames, starting with its caller and
// skipping skip Frames.
func Frames(skip, max int) []runtime.Frame {
	pc := make([]uintptr, max)
	n := runtime.Callers(skip+2, pc)
	if n == 0 {
		return nil
	}
	pc = pc[:n]
	frames := runtime.CallersFrames(pc)
	var fs []runtime.Frame
	for {
		f, more := frames.Next()
		fs = append(fs, f)
		if !more {
			break
		}
	}
	return fs
}

// Match returns the first stack frame for which the predicate function returns
// true. Returns nil if no match is found. Starts matching after skip frames,
// starting with its caller.
func Match(skip int, predicate func(runtime.Frame) bool) *runtime.Frame {
	i, n := skip+1, 16
	for {
		fs := Frames(i, n)
		for j, f := range fs {
			if predicate(f) {
				return &fs[j]
			}
		}
		if len(fs) < n {
			break
		}
		i += n
	}
	return nil
}

// Main returns the main() function Frame.
func Main() *runtime.Frame {
	return Match(1, func(f runtime.Frame) bool {
		return f.Function == "main.main"
	})
}

// ExternalCaller returns the first frame outside the callers package.
func ExternalCaller() *runtime.Frame {
	var first *runtime.Frame
	return Match(1, func(f runtime.Frame) bool {
		if first == nil {
			first = &f
		}
		return pkg(first.Function) != pkg(f.Function)
	})
}

func pkg(ident string) string {
	dir, name := path.Split(ident)
	parts := strings.Split(name, ".")
	return dir + parts[0]
}
