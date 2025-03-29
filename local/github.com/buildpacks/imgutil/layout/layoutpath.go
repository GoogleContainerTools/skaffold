package layout

import (
	"path/filepath"

	ggcr "github.com/google/go-containerregistry/pkg/v1/layout"
)

type Path struct {
	ggcr.Path
}

func (l Path) append(elem ...string) string {
	complete := []string{string(l.Path)}
	return filepath.Join(append(complete, elem...)...)
}
