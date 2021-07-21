package main

import (
	"context"
	"fmt"
	"time"

	"github.com/GoogleContainerTools/skaffold/hack/time-comparison/events"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/event"
	"github.com/GoogleContainerTools/skaffold/proto/v1"
	"github.com/sirupsen/logrus"
)

// SkaffoldRunMetrics collects and aggregates metrics for the inner loop from an events file
func SkaffoldRunMetrics(ctx context.Context, eventsFileAbsPath string) ([]innerLoopMetric, error) {
	events.EventsFileAbsPath = eventsFileAbsPath
	ef, err := events.File()
	if err != nil {
		return []innerLoopMetric{}, fmt.Errorf("events file: %w", err)
	}
	logEntries, err := events.GetFromFile(ef)
	if err != nil {
		return []innerLoopMetric{}, fmt.Errorf("getting events from file: %w", err)
	}
	innerLoopMetrics := splitEntriesByDevLoop(logEntries)
	logrus.Infof("Inner loop metrics for this run: %v", innerLoopMetrics)
	return innerLoopMetrics, nil
}

func splitEntriesByDevLoop(logEntries []proto.LogEntry) []innerLoopMetric {
	var ilms []innerLoopMetric

	var current innerLoopMetric
	var buildStartTime, deployStartTime, statusCheckStartTime time.Time
	for _, le := range logEntries {
		switch le.Event.GetEventType().(type) {
		case *proto.Event_MetaEvent:
			fmt.Println("metadata event logic not yet implemented")
		case *proto.Event_DevLoopEvent:
			// we have reached the end of a dev loop
			status := le.GetEvent().GetDevLoopEvent().GetStatus()
			if status == event.Succeeded {
				buildStartTime, deployStartTime, statusCheckStartTime = time.Time{}, time.Time{}, time.Time{}
				ilms = append(ilms, current)
			}
		case *proto.Event_BuildEvent:
			status := le.GetEvent().GetBuildEvent().GetStatus()
			unixTimestamp := time.Unix(0, le.GetTimestamp().AsTime().UnixNano())
			if status == event.InProgress && buildStartTime.IsZero() {
				buildStartTime = unixTimestamp
			} else if status == event.Complete {
				current.buildTime = unixTimestamp.Sub(buildStartTime)
			}
		case *proto.Event_DeployEvent:
			status := le.GetEvent().GetDeployEvent().GetStatus()
			unixTimestamp := time.Unix(0, le.GetTimestamp().AsTime().UnixNano())
			if status == event.InProgress {
				deployStartTime = unixTimestamp
				// deploy is finished when it is marked "Complete"
			} else if status == event.Complete {
				current.deployTime = unixTimestamp.Sub(deployStartTime)
			}
		case *proto.Event_StatusCheckEvent:
			status := le.GetEvent().GetStatusCheckEvent().GetStatus()
			unixTimestamp := time.Unix(0, le.GetTimestamp().AsTime().UnixNano())
			if status == event.Started {
				statusCheckStartTime = unixTimestamp
			} else if status == event.Succeeded {
				current.statusCheckTime = unixTimestamp.Sub(statusCheckStartTime)
			}
		}
	}
	return ilms
}
