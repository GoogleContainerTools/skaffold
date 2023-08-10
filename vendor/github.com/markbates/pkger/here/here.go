package here

import (
	"github.com/gobuffalo/here"
)

type Info = here.Info
type Module = here.Module
type Path = here.Path

var Here = here.New()
var Dir = Here.Dir
var Package = Here.Package
var Current = Here.Current
