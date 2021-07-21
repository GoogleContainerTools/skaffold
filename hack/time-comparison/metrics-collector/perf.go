package timecomp

import (
	"context"
	"flag"
	"fmt"

	"github.com/GoogleContainerTools/skaffold/hack/time-comparison/events"
	"github.com/GoogleContainerTools/skaffold/hack/time-comparison/metrics-collector/config"
	"github.com/GoogleContainerTools/skaffold/hack/time-comparison/metrics-collector/skaffold"
)

type TimeComparisonInput struct {
	ConfigFile         string
	SkaffoldBinaryPath string
	EventsFileAbsPath  string
	SkaffoldFlags      string
	Cleanup            bool
}

var timeComparisonInput TimeComparisonInput

func CollectTimingInformation(tci TimeComparisonInput) error {
	timeComparisonInput = tci
	if err := collectTimingInformation(context.Background()); err != nil {
		return err
	}
	return nil
}

func collectTimingInformation(ctx context.Context) error {
	cfg, err := config.Get(timeComparisonInput.ConfigFile)
	if err != nil {
		return fmt.Errorf("getting config: %w", err)
	}
	flag.Parse()
	events.EventsFileAbsPath = timeComparisonInput.EventsFileAbsPath
	skaffold.SkaffoldBinaryPath = timeComparisonInput.SkaffoldBinaryPath
	for _, app := range cfg.Apps {
		if err := skaffold.Dev(ctx, app, timeComparisonInput.SkaffoldFlags); err != nil {
			fmt.Printf("%v\n", err)
			continue
		}
		if timeComparisonInput.Cleanup {
			events.Cleanup()
		}
	}
	return nil
}
