package writer

import (
	"bytes"
	"encoding/json"
)

type JSON struct {
	StructuredFormat
}

func NewJSON() *JSON {
	return &JSON{
		StructuredFormat: StructuredFormat{
			MarshalFunc: func(i interface{}) ([]byte, error) {
				buf := bytes.NewBuffer(nil)
				if err := json.NewEncoder(buf).Encode(i); err != nil {
					return []byte{}, err
				}

				formattedBuf := bytes.NewBuffer(nil)
				if err := json.Indent(formattedBuf, buf.Bytes(), "", "  "); err != nil {
					return []byte{}, err
				}
				return formattedBuf.Bytes(), nil
			},
		},
	}
}
