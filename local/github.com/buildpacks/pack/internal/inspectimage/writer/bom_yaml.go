package writer

import (
	"bytes"

	"gopkg.in/yaml.v3"
)

type YAMLBOM struct {
	StructuredBOMFormat
}

func NewYAMLBOM() *YAMLBOM {
	return &YAMLBOM{
		StructuredBOMFormat: StructuredBOMFormat{
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
