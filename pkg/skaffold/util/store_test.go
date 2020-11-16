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

package util

import (
	"fmt"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestSyncStore(t *testing.T) {
	testutil.Run(t, "test store", func(t *testutil.T) {
		// This test runs a counter function twice for each key from [0, 5) and tests that the function only executes once for each key when called inside `once.Do` method.
		counts := make([]int32, 5)
		f := func(i int) int {
			atomic.AddInt32(&counts[i], 1)
			return i
		}
		var wg sync.WaitGroup
		wg.Add(10)

		s := NewSyncStore()
		for i := 0; i < 5; i++ {
			for j := 0; j < 2; j++ {
				go func(i int) {
					k := strconv.Itoa(i)
					val := s.Exec(k, func() interface{} {
						return f(i)
					})
					t.CheckDeepEqual(i, val)
					wg.Done()
				}(i)
			}
		}
		wg.Wait()
		for i := 0; i < 5; i++ {
			if counts[i] > 1 {
				t.Fatalf("hash func called more than once for image%d", i)
			}
		}
	})

	testutil.Run(t, "test panic handled correctly", func(t *testutil.T) {
		s := NewSyncStore()
		val := s.Exec("panic", func() interface{} {
			panic(fmt.Errorf("message"))
		})
		// make sure val is of type StoreError
		switch tv := val.(type) {
		case StoreError:
			t.CheckDeepEqual("internal error retrieving cached results for key panic: message", tv.Error())
		default:
			t.Fatalf("expected to retrieve result of type StoreError but found %T", tv)
		}
	})
}
