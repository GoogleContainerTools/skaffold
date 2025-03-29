package fakes

import "github.com/buildpacks/pack/internal/builder"

type FakeLabelManagerFactory struct {
	BuilderLabelManagerToReturn builder.LabelInspector

	ReceivedInspectable builder.Inspectable
}

func NewFakeLabelManagerFactory(builderLabelManagerToReturn builder.LabelInspector) *FakeLabelManagerFactory {
	return &FakeLabelManagerFactory{
		BuilderLabelManagerToReturn: builderLabelManagerToReturn,
	}
}

func (f *FakeLabelManagerFactory) BuilderLabelManager(inspectable builder.Inspectable) builder.LabelInspector {
	f.ReceivedInspectable = inspectable

	return f.BuilderLabelManagerToReturn
}
