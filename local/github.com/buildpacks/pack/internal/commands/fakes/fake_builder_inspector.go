package fakes

import (
	"github.com/buildpacks/pack/pkg/client"
)

type FakeBuilderInspector struct {
	InfoForLocal   *client.BuilderInfo
	InfoForRemote  *client.BuilderInfo
	ErrorForLocal  error
	ErrorForRemote error

	ReceivedForLocalName      string
	ReceivedForRemoteName     string
	CalculatedConfigForLocal  client.BuilderInspectionConfig
	CalculatedConfigForRemote client.BuilderInspectionConfig
}

func (i *FakeBuilderInspector) InspectBuilder(
	name string,
	daemon bool,
	modifiers ...client.BuilderInspectionModifier,
) (*client.BuilderInfo, error) {
	if daemon {
		i.CalculatedConfigForLocal = client.BuilderInspectionConfig{}
		for _, mod := range modifiers {
			mod(&i.CalculatedConfigForLocal)
		}
		i.ReceivedForLocalName = name
		return i.InfoForLocal, i.ErrorForLocal
	}

	i.CalculatedConfigForRemote = client.BuilderInspectionConfig{}
	for _, mod := range modifiers {
		mod(&i.CalculatedConfigForRemote)
	}
	i.ReceivedForRemoteName = name
	return i.InfoForRemote, i.ErrorForRemote
}
