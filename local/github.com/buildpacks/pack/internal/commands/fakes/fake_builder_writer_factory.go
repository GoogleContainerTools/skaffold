package fakes

import (
	"github.com/buildpacks/pack/internal/builder/writer"
)

type FakeBuilderWriterFactory struct {
	ReturnForWriter writer.BuilderWriter
	ErrorForWriter  error

	ReceivedForKind string
}

func (f *FakeBuilderWriterFactory) Writer(kind string) (writer.BuilderWriter, error) {
	f.ReceivedForKind = kind

	return f.ReturnForWriter, f.ErrorForWriter
}
