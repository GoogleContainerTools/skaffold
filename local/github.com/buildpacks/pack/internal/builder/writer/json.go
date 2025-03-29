package writer

import (
	"bytes"
	"encoding/json"
)

type JSON struct {
	StructuredFormat
}

func NewJSON() BuilderWriter {
	return &JSON{
		StructuredFormat: StructuredFormat{
			MarshalFunc: func(i interface{}) ([]byte, error) {
				buf, err := json.Marshal(i)
				if err != nil {
					return []byte{}, err
				}
				formattedBuf := bytes.NewBuffer(nil)
				err = json.Indent(formattedBuf, buf, "", "  ")
				return formattedBuf.Bytes(), err
			},
		},
	}
}
