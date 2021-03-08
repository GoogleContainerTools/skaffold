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

package main

import (
	"fmt"
	"testing"
)

func TestMinIntBasic(tb *testing.T) {
	fmt.Println("Running Basic test.")
	min := MinInt(5, -5)
	if min != -5 {
		tb.Errorf("MinInt(5, -5) returned %d; expecting -5", min)
	}
}

func TestMinIntTableDriven(tdt *testing.T) {
	var tests = []struct {
		x, y int
		want int
	}{
		{0, 0, 0},
		{1, 0, 0},
		{0, 1, 0},
		{0, -1, -1},
		{-1, 0, -1},
		{-2, -5, -5},
		{-5, -2, -5},
	}

	fmt.Println("Running Table driven test.")
	for _, t := range tests {
		testname := fmt.Sprintf("TestMinInt(): %d,%d", t.x, t.y)
		tdt.Run(testname, func(tdt *testing.T) {
			min := MinInt(t.x, t.y)
			if min != t.want {
				tdt.Errorf("MinInt(%d, %d) returned %d; expecting %d", t.x, t.y, min, t.want)
			}
		})
	}
}
