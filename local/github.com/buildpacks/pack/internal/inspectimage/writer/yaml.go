package writer

import (
	"bytes"

	"gopkg.in/yaml.v3"
)

type YAML struct {
	StructuredFormat
}

func NewYAML() *YAML {
	return &YAML{
		StructuredFormat: StructuredFormat{
			MarshalFunc: func(i interface{}) ([]byte, error) {
				buf := bytes.NewBuffer(nil)
				if err := yaml.NewEncoder(buf).Encode(i); err != nil {
					return []byte{}, err
				}
				return buf.Bytes(), nil
			},
		},
	}
}
