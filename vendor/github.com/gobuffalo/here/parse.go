package here

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
)

func (i Info) Parse(p string) (Path, error) {
	p = strings.TrimSpace(p)
	p = filepath.Clean(p)
	p = strings.TrimPrefix(p, i.Dir)

	p = strings.Replace(p, "\\", "/", -1)
	p = strings.TrimSpace(p)

	if len(p) == 0 || p == ":" || p == "." {
		return i.build("", "", "")
	}

	res := pathrx.FindAllStringSubmatch(p, -1)
	if len(res) == 0 {
		return Path{}, fmt.Errorf("could not parse %q", p)
	}

	matches := res[0]

	if len(matches) != 4 {
		return Path{}, fmt.Errorf("could not parse %q", p)
	}

	return i.build(p, matches[1], matches[3])
}

func (i Info) build(p, pkg, name string) (Path, error) {
	pt := Path{
		Pkg:  pkg,
		Name: name,
	}

	if strings.HasPrefix(pt.Pkg, "/") || len(pt.Pkg) == 0 {
		pt.Name = pt.Pkg
		pt.Pkg = i.Module.Path
	}

	if len(pt.Name) == 0 {
		pt.Name = "/"
	}

	if pt.Pkg == pt.Name {
		pt.Pkg = i.Module.Path
		pt.Name = "/"
	}

	if !strings.HasPrefix(pt.Name, "/") {
		pt.Name = "/" + pt.Name
	}
	pt.Name = strings.TrimPrefix(pt.Name, i.Dir)
	return pt, nil
}

var pathrx = regexp.MustCompile("([^:]+)(:(/.+))?")
