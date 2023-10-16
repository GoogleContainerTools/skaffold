package here

import (
	"encoding/json"
)

type Module struct {
	Path      string
	Main      bool
	Dir       string
	GoMod     string
	GoVersion string
}

func (i Module) String() string {
	b, err := json.MarshalIndent(i, "", "  ")
	if err != nil {
		return err.Error()
	}
	return string(b)
}

func (i Module) IsZero() bool {
	const cla = "command-line-arguments"
	if i.Path == cla {
		i.Path = ""
	}
	return i.String() == Module{}.String()
}
