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

package stringslice

// Contains returns true if a string slice contains the given string
func Contains(sl []string, s string) bool {
	return Index(sl, s) >= 0
}

// Index returns the index of a first occurrence of a string within a string slice
func Index(sl []string, s string) int {
	for i, a := range sl {
		if a == s {
			return i
		}
	}
	return -1
}

// Insert inserts a string slice into another string slice at the given index
func Insert(sl []string, index int, insert []string) []string {
	newSlice := make([]string, len(sl)+len(insert))
	copy(newSlice[0:index], sl[0:index])
	copy(newSlice[index:index+len(insert)], insert)
	copy(newSlice[index+len(insert):], sl[index:])
	return newSlice
}

// Remove removes a string from a slice of strings
func Remove(s []string, target string) []string {
	for i := len(s) - 1; i >= 0; i-- {
		if s[i] == target {
			s = append(s[:i], s[i+1:]...)
		}
	}
	return s
}
