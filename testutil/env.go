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

package testutil

import "os"

// SetEnvs takes a map of key values to set using t.Setenv and restore
// the environment variable to its original value after the test.
func (t *T) SetEnvs(envs map[string]string) {
	for key, value := range envs {
		t.Setenv(key, value)
	}
}

func (t *T) UnsetEnv(key string) {
	prevValue, ok := os.LookupEnv(key)
	if ok {
		if err := os.Unsetenv(key); err != nil {
			t.Fatalf("cannot unset environment variable: %v", err)
		}
		t.Cleanup(func() {
			os.Setenv(key, prevValue)
		})
	}
}
