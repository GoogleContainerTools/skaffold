package pkging

import "os"

type Adder interface {
	Add(files ...*os.File) error
}
