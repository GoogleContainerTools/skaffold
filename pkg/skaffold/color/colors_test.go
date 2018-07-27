/*
Copyright 2018 The Skaffold Authors

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

package color

import (
	"testing"
)

func TestColorSprint(t *testing.T) {
	c := Red.Sprint("TEXT")

	expected := "\033[31mTEXT\033[0m"
	if c != expected {
		t.Errorf("Expected %s. Got %s", expected, c)
	}
}

func TestColorSprintf(t *testing.T) {
	c := Green.Sprintf("A GREAT NUMBER IS %d", 5)

	expected := "\033[32mA GREAT NUMBER IS 5\033[0m"
	if c != expected {
		t.Errorf("Expected %s. Got %s", expected, c)
	}
}
