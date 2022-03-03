/*
Copyright 2021 The Skaffold Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package events

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"strings"
	"time"

	"github.com/golang/protobuf/jsonpb"
	"github.com/pkg/errors"

	"github.com/GoogleContainerTools/skaffold/hack/comparisonstats/types"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/event"
	v1 "github.com/GoogleContainerTools/skaffold/proto/v1"
)

// ParseEventDuration collects and aggregates metrics for the initial + inner loops from an events file
func ParseEventDuration(ctx context.Context, eventsFileAbsPath string) (*types.DevLoopTimes, error) {
	logEntries, err := getFromFile(eventsFileAbsPath)
	if err != nil {
		return nil, fmt.Errorf("getting events from file: %w", err)
	}
	return splitEntriesByDevLoop(logEntries), nil
}

func splitEntriesByDevLoop(logEntries []*v1.LogEntry) *types.DevLoopTimes {
	devLoopTimes := types.DevLoopTimes{
		InnerBuildTimes:       []time.Duration{},
		InnerDeployTimes:      []time.Duration{},
		InnerStatusCheckTimes: []time.Duration{},
	}

	var buildTime, deployTime, statusCheckTime time.Duration
	var buildStartTime, deployStartTime, statusCheckStartTime time.Time
	isFirstEntry := true
	for _, le := range logEntries {
		switch le.Event.GetEventType().(type) {
		case *v1.Event_MetaEvent:
			fmt.Println("metadata event logic not yet implemented")
		case *v1.Event_DevLoopEvent:
			// we have reached the end of a dev loop
			status := le.GetEvent().GetDevLoopEvent().GetStatus()
			if status == event.Succeeded {
				buildStartTime, deployStartTime, statusCheckStartTime = time.Time{}, time.Time{}, time.Time{}
				if isFirstEntry {
					isFirstEntry = false
					devLoopTimes.InitialBuildTime = buildTime
					devLoopTimes.InitialDeployTime = deployTime
					devLoopTimes.InitialStatusCheckTime = statusCheckTime
				} else {
					devLoopTimes.InnerBuildTimes = append(devLoopTimes.InnerBuildTimes, buildTime)
					devLoopTimes.InnerDeployTimes = append(devLoopTimes.InnerDeployTimes, deployTime)
					devLoopTimes.InnerStatusCheckTimes = append(devLoopTimes.InnerStatusCheckTimes, statusCheckTime)
				}
			}
		case *v1.Event_BuildEvent:
			status := le.GetEvent().GetBuildEvent().GetStatus()
			unixTimestamp := time.Unix(0, le.GetTimestamp().AsTime().UnixNano())
			if status == event.InProgress && buildStartTime.IsZero() {
				buildStartTime = unixTimestamp
			} else if status == event.Complete {
				buildTime = unixTimestamp.Sub(buildStartTime)
			}
		case *v1.Event_DeployEvent:
			status := le.GetEvent().GetDeployEvent().GetStatus()
			unixTimestamp := time.Unix(0, le.GetTimestamp().AsTime().UnixNano())
			if status == event.InProgress {
				deployStartTime = unixTimestamp
				// deploy is finished when it is marked "Complete"
			} else if status == event.Complete {
				deployTime = unixTimestamp.Sub(deployStartTime)
			}
		case *v1.Event_StatusCheckEvent:
			status := le.GetEvent().GetStatusCheckEvent().GetStatus()
			unixTimestamp := time.Unix(0, le.GetTimestamp().AsTime().UnixNano())
			if status == event.Started {
				statusCheckStartTime = unixTimestamp
			} else if status == event.Succeeded {
				statusCheckTime = unixTimestamp.Sub(statusCheckStartTime)
			}
		}
	}
	return &devLoopTimes
}

func get(contents []byte) ([]*v1.LogEntry, error) {
	entries := strings.Split(string(contents), "\n")
	var logEntries []*v1.LogEntry
	unmarshaller := jsonpb.Unmarshaler{}
	for _, entry := range entries {
		if entry == "" {
			continue
		}
		logEntry := new(v1.LogEntry)
		buf := bytes.NewBuffer([]byte(entry))
		if err := unmarshaller.Unmarshal(buf, logEntry); err != nil {
			return nil, errors.Wrap(err, "unmarshalling")
		}
		logEntries = append(logEntries, logEntry)
	}
	return logEntries, nil
}

func getFromFile(fp string) ([]*v1.LogEntry, error) {
	contents, err := ioutil.ReadFile(fp)
	if err != nil {
		return nil, errors.Wrapf(err, "reading %s", fp)
	}
	return get(contents)
}
