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

package latest

// TestStruct for testing the schema generator.
type TestStruct struct {
	// Int32Field is an integer
	Int32Field int32 `yaml:"int32Field"`

	// Int64Field is an integer
	Int64Field int32 `yaml:"int64Field"`

	// IntField is an integer
	IntField int `yaml:"intField"`
}
