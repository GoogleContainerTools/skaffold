/*
Copyright 2020 The Skaffold Authors

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

package binpack

import (
	"fmt"
	"testing"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/output/log"
	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestPartitions(t *testing.T) {
	level := log.GetLevel()
	defer log.SetLevel(level)
	log.SetLevel(log.TraceLevel)
	partitions, lastPartition := Partitions(nil, Timings, MaxBinTime)
	testutil.CheckDeepEqual(t, len(partitions), len(Timings))
	var bins []bin
	for i := 0; i <= lastPartition; i++ {
		bins = append(bins, bin{})
	}

	for testName, p := range partitions {
		if p > lastPartition {
			t.Errorf("invalid partition %d > max_partition(%d), for %s", p, lastPartition, testName)
		}
	}
	for _, timing := range Timings {
		p := partitions[timing.name]
		fmt.Printf("P:%d | %s: %f\n", p, timing.name, timing.time)
		bins[p].total += timing.time
		if bins[p].total > MaxBinTime {
			t.Errorf("partition %d is oversubscribed %f > %f", p, bins[p].total, MaxBinTime)
		}
	}
}
