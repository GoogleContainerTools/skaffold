package writer

import (
	"bytes"

	"github.com/pelletier/go-toml"
)

type TOML struct {
	StructuredFormat
}

func NewTOML() *TOML {
	return &TOML{
		StructuredFormat: StructuredFormat{
			MarshalFunc: func(i interface{}) ([]byte, error) {
				buf := bytes.NewBuffer(nil)
				if err := toml.NewEncoder(buf).Encode(i); err != nil {
					return []byte{}, err
				}
				return buf.Bytes(), nil
			},
		},
	}
}
