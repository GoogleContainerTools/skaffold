package fakes

import (
	"github.com/buildpacks/pack/internal/builder/writer"
	"github.com/buildpacks/pack/internal/config"
	"github.com/buildpacks/pack/pkg/client"
	"github.com/buildpacks/pack/pkg/logging"
)

type FakeBuilderWriter struct {
	PrintForLocal  string
	PrintForRemote string
	ErrorForPrint  error

	ReceivedInfoForLocal   *client.BuilderInfo
	ReceivedInfoForRemote  *client.BuilderInfo
	ReceivedErrorForLocal  error
	ReceivedErrorForRemote error
	ReceivedBuilderInfo    writer.SharedBuilderInfo
	ReceivedLocalRunImages []config.RunImage
}

func (w *FakeBuilderWriter) Print(
	logger logging.Logger,
	localRunImages []config.RunImage,
	local, remote *client.BuilderInfo,
	localErr, remoteErr error,
	builderInfo writer.SharedBuilderInfo,
) error {
	w.ReceivedInfoForLocal = local
	w.ReceivedInfoForRemote = remote
	w.ReceivedErrorForLocal = localErr
	w.ReceivedErrorForRemote = remoteErr
	w.ReceivedBuilderInfo = builderInfo
	w.ReceivedLocalRunImages = localRunImages

	logger.Infof("\nLOCAL:\n%s\n", w.PrintForLocal)
	logger.Infof("\nREMOTE:\n%s\n", w.PrintForRemote)

	return w.ErrorForPrint
}
