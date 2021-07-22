/*
Copyright 2019 The Skaffold Authors

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

package main

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"time"

	"github.com/GoogleContainerTools/skaffold/hack/time-comparison/events"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/event"
	"github.com/GoogleContainerTools/skaffold/proto/v1"
	"github.com/sirupsen/logrus"
)

const binariesToCompare = 2

func main() {

	if len(os.Args) != binariesToCompare+1 {
		logrus.Fatalf("time-comparison expects input of the form: timer-comparison /usr/bin/bin1 /usr/bin/bin2")
	}

	ctx := context.Background()

	var b bytes.Buffer
	workDir, err := os.Getwd()
	if err != nil {
		logrus.Fatal(err)
	}

	for i := 0; i < binariesToCompare; i++ {
		eventsFileAbsPath := filepath.Join(workDir, "events-"+strconv.Itoa(i))
		if err := goTest(ctx, os.Args[i+1], eventsFileAbsPath); err != nil {
			os.Exit(1)
		}
		ilms, err := InnerLoopMetrics(ctx, eventsFileAbsPath)
		if err != nil {
			logrus.Fatal(err)
		}

		ilm := ilms[1]
		if i != 0 {
			fmt.Fprintln(&b, "===================")
		}
		fmt.Fprintf(&b, "build times for %v:\n", os.Args[i+1])
		fmt.Fprintf(&b, "build time: %v\n", ilm.buildTime)
		fmt.Fprintf(&b, "deploy time: %v\n", ilm.deployTime)
		fmt.Fprintf(&b, "status check time: %v\n", ilm.statusCheckTime)
		fmt.Fprintf(&b, "total time: %v\n", ilm.buildTime+ilm.deployTime+ilm.statusCheckTime)
	}
	fmt.Println(b.String())
	if err := ioutil.WriteFile(filepath.Join(workDir, "gh-comment.txt"), b.Bytes(), 0644); err != nil {
		logrus.Fatal(err)
	}
}

func goTest(ctx context.Context, skaffoldBinaryPath string, eventsFileAbsPath string) error {
	args := append([]string{"--export-metrics=false", "--cleanup=false",
		"--eventsFileAbsPath", eventsFileAbsPath}, "--skaffoldBinaryPath", skaffoldBinaryPath)

	metricsCollectorCmd := "metrics-collector"
	cmd := exec.CommandContext(ctx, metricsCollectorCmd, args...)

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err != nil {
		logrus.Fatalf("command %v failed with %v:%v:%v", metricsCollectorCmd, err)
	}
	return err
}

// InnerLoopMetrics collects metrics for the inner loop and exports them
// to Cloud Monitoring
func InnerLoopMetrics(ctx context.Context, eventsFileAbsPath string) ([]innerLoopMetric, error) {
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
			unixTimestamp := time.Unix(le.GetTimestamp().AsTime().Unix(), 0)
			if status == event.InProgress && buildStartTime.IsZero() {
				buildStartTime = unixTimestamp
			} else if status == event.Complete {
				current.buildTime = unixTimestamp.Sub(buildStartTime).Seconds()
			}
		case *proto.Event_DeployEvent:
			status := le.GetEvent().GetDeployEvent().GetStatus()
			unixTimestamp := time.Unix(le.GetTimestamp().AsTime().Unix(), 0)
			if status == event.InProgress {
				deployStartTime = unixTimestamp
				// deploy is finished when it is marked "Complete"
			} else if status == event.Complete {
				current.deployTime = unixTimestamp.Sub(deployStartTime).Seconds()
			}
		case *proto.Event_StatusCheckEvent:
			status := le.GetEvent().GetStatusCheckEvent().GetStatus()
			unixTimestamp := time.Unix(le.GetTimestamp().AsTime().Unix(), 0)
			if status == event.Started {
				statusCheckStartTime = unixTimestamp
			} else if status == event.Succeeded {
				current.statusCheckTime = unixTimestamp.Sub(statusCheckStartTime).Seconds()
			}
		}
	}
	return ilms
}

type innerLoopMetric struct {
	buildTime       float64
	deployTime      float64
	statusCheckTime float64
}
