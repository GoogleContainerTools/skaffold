package here

import (
	"fmt"
)

type Path struct {
	Pkg  string
	Name string
}

func (p Path) String() string {
	if p.Name == "" {
		p.Name = "/"
	}
	if p.Pkg == "" {
		return p.Name
	}
	return fmt.Sprintf("%s:%s", p.Pkg, p.Name)
}
