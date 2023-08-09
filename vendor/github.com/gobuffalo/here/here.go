package here

import (
	"bytes"
	"fmt"
	"os/exec"
	"regexp"
	"sync"
)

type Here struct {
	infos   *infoMap
	curOnce sync.Once
	curErr  error
	current Info
}

// New returns a Here type that will cache
// all results. This speeds up repeated calls,
// and can be useful for testing.
func New() Here {
	return Here{
		infos: &infoMap{
			data: &sync.Map{},
		},
	}
}

func run(n string, args ...string) ([]byte, error) {
	c := exec.Command(n, args...)

	bb := &bytes.Buffer{}
	ebb := &bytes.Buffer{}
	c.Stdout = bb
	c.Stderr = ebb
	err := c.Run()
	if err != nil {
		return nil, fmt.Errorf("%s: %s", err, ebb)
	}

	return bb.Bytes(), nil
}

func (h Here) cache(p string, fn func(string) (Info, error)) (Info, error) {
	i, ok := h.infos.Load(p)
	if ok {
		return i, nil
	}
	i, err := fn(p)
	if err != nil {
		return i, err
	}
	h.infos.Store(p, i)
	return i, nil
}

var nonGoDirRx = regexp.MustCompile(`cannot find main|go help modules|go: |build .:|no Go files|can't load package|not using modules`)
