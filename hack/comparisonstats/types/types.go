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

package types

import (
	"bytes"
	"fmt"
	"math"
	"time"

	"github.com/GoogleContainerTools/skaffold/hack/comparisonstats/util"
)

type ComparisonStatsCmdArgs struct {
	SkaffoldBinaries []string
	ExampleAppName   string
	ExampleSrcFile   string
	CommentText      string
}

func ParseComparisonStatsCmdArgs(args []string) ComparisonStatsCmdArgs {
	return ComparisonStatsCmdArgs{
		SkaffoldBinaries: []string{args[0], args[1]},
		ExampleAppName:   args[2],
		ExampleSrcFile:   args[3],
		CommentText:      args[4],
	}
}

type Config struct {
	DevIterations       int64  `yaml:"devIterations"`
	FirstSkaffoldFlags  string `yaml:"mainBranchSkaffoldFlags"` // NOTE: names mismatched for make vs gh action UX usage
	SecondSkaffoldFlags string `yaml:"thisPRSkaffoldFlags"`
	ExampleAppName      string `yaml:"exampleAppName"`
	ExampleSrcFile      string `yaml:"exampleSrcFile"`
	CommentText         string `yaml:"commentText"`
}

type ComparisonStatsSummary struct {
	BinaryPath            string
	CmdArgs               []string
	BinarySize            int64
	DevIterations         int64
	DevLoopEventDurations *DevLoopTimes
}

// durations holds time.Duration values.
type durations []time.Duration

func (ds durations) avg() time.Duration {
	var total time.Duration
	for _, t := range ds {
		total += t
	}
	return time.Duration(int(total) / len(ds))
}

func (ds durations) stdDev() time.Duration {
	mean := ds.avg()
	var s float64
	for _, t := range ds {
		s += math.Pow(float64(mean-t), 2)
	}
	meansq := s / float64(len(ds))
	return time.Duration(math.Sqrt(meansq))
}

func (cs *ComparisonStatsSummary) String() string {
	var b bytes.Buffer

	fmt.Fprintln(&b, "")
	fmt.Fprintf(&b, "information for %v for %d iterations of %s:\n", cs.BinaryPath, cs.DevIterations, cs.CmdArgs)
	fmt.Fprintf(&b, "binary size: %v\n", util.HumanReadableBytesSizeIEC(cs.BinarySize))
	fmt.Fprintf(&b, "initial loop build, deploy, status-check times: %v\n", []time.Duration{
		cs.DevLoopEventDurations.InitialBuildTime, cs.DevLoopEventDurations.InitialDeployTime, cs.DevLoopEventDurations.InitialStatusCheckTime})
	fmt.Fprintf(&b, "inner loop build time avg: %s\n", cs.DevLoopEventDurations.InnerBuildTimes.avg())
	fmt.Fprintf(&b, "inner loop build time stdDev: %s\n", cs.DevLoopEventDurations.InnerBuildTimes.stdDev())
	fmt.Fprintf(&b, "inner loop deploy time avg: %s\n", cs.DevLoopEventDurations.InnerDeployTimes.avg())
	fmt.Fprintf(&b, "inner loop deploy time stdDev: %s\n", cs.DevLoopEventDurations.InnerDeployTimes.stdDev())
	fmt.Fprintf(&b, "inner loop status check time avg: %s\n", cs.DevLoopEventDurations.InnerStatusCheckTimes.avg())
	fmt.Fprintf(&b, "inner loop status check time stdDev: %s\n", cs.DevLoopEventDurations.InnerStatusCheckTimes.stdDev())
	return b.String()
}

type DevLoopTimes struct {
	InitialBuildTime       time.Duration
	InitialDeployTime      time.Duration
	InitialStatusCheckTime time.Duration
	InnerBuildTimes        durations
	InnerDeployTimes       durations
	InnerStatusCheckTimes  durations
}

// Application represends a single test application
type Application struct {
	Name          string            `yaml:"name" yamltags:"required"`
	Context       string            `yaml:"context" yamltags:"required"`
	Dev           Dev               `yaml:"dev" yamltags:"required"`
	DevIterations int64             `yaml:"devIterations" yamltags:"required"`
	Labels        map[string]string `yaml:"labels" yamltags:"required"`
}

// Dev describes necessary info for running `skaffold dev` on a test application
type Dev struct {
	Command string `yaml:"command" yamltags:"required"`
}
