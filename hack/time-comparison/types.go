package main

import (
	"bytes"
	"fmt"
	"time"
)

type TimeComparisonOutput struct {
	binaryPath string
	binarySize int64
	// loopMetrics              []innerLoopMetric
	innerLoopBuildTime       time.Duration
	innerLoopDeployTime      time.Duration
	innerLoopStatusCheckTime time.Duration
	innerLoopTotalTime       time.Duration
}

func (tco *TimeComparisonOutput) String() string {
	// innerLoopBuildTime:       mtrcs[1].buildTime,
	// 		innerLoopDeployTime:      mtrcs[1].deployTime,
	// 		innerLoopStatusCheckTime: mtrcs[1].statusCheckTime,
	// 		innerLoopTotalTime:       mtrcs[1].buildTime + mtrcs[1].deployTime + mtrcs[1].statusCheckTime,

	// information for %v over %d runs:\n
	// median:%s, mean: %s, first run (not inner loop) %s
	var b bytes.Buffer
	fmt.Fprintln(&b, "==========")
	fmt.Fprintf(&b, "information for %v:\n", tco.binaryPath)
	fmt.Fprintf(&b, "binary size: %v\n", humanReadableBytesSizeSI(tco.binarySize))
	fmt.Fprintf(&b, "build time: %s\n", tco.innerLoopBuildTime)
	fmt.Fprintf(&b, "deploy time: %s\n", tco.innerLoopDeployTime)
	fmt.Fprintf(&b, "status check time: %s\n", tco.innerLoopStatusCheckTime)
	fmt.Fprintf(&b, "total time: %s\n", tco.innerLoopTotalTime)
	return b.String()
}

type innerLoopMetric struct {
	buildTime       time.Duration
	deployTime      time.Duration
	statusCheckTime time.Duration
}
