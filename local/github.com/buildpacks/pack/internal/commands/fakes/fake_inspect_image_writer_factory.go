package fakes

import (
	"github.com/buildpacks/pack/internal/inspectimage/writer"
)

type FakeInspectImageWriterFactory struct {
	ReturnForWriter writer.InspectImageWriter
	ErrorForWriter  error

	ReceivedForKind string
	ReceivedForBOM  bool
}

func (f *FakeInspectImageWriterFactory) Writer(kind string, bom bool) (writer.InspectImageWriter, error) {
	f.ReceivedForKind = kind
	f.ReceivedForBOM = bom

	return f.ReturnForWriter, f.ErrorForWriter
}
