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
	"math/big"
	"testing"
)

func TestIntMinBasic(t *testing.T) {
	ans, _ := IntMin(big.NewInt(2), big.NewInt(-2))
	if ans.Cmp(big.NewInt(-2)) != 0 {
		t.Errorf("IntMin(2, -2) = %d; want -2", ans)
	}
}

func TestIntMinTableDriven(t *testing.T) {
	var tests = []struct {
		a, b int
		want int
	}{
		{0, 1, 0},
		{1, 0, 0},
		{2, -2, -2},
		{0, -1, -1},
		{-1, 0, -1},
	}

	for _, tt := range tests {
		testname := fmt.Sprintf("%d,%d", tt.a, tt.b)
		x := big.NewInt(int64(tt.a))
		y := big.NewInt(int64(tt.b))
		want := big.NewInt(int64(tt.want))
		t.Run(testname, func(t *testing.T) {
			ans, _ := IntMin(x, y)
			if ans.Cmp(want) != 0 {
				t.Errorf("got %d, want %d", ans, want)
			}
		})
	}
}
