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
	// RequiredField should be required
	RequiredField          string `yaml:"reqField" yamltags:"required"`
	InlineOneOfStruct      `yaml:"inline"`
	InlineOneOfStructAnyOf `yaml:"inline"`
}

// InlineOneOfStruct is embedded inline into TestStruct
type InlineOneOfStruct struct {

	// Field1 should be the first choice
	Field1 string `yaml:"f1"`

	// Field2 should be the second choice
	Field2 string `yaml:"f2"`
}

// InlineOneOfStructAnyOf is embedded inline into TestStruct
type InlineOneOfStructAnyOf struct {

	// Choice1 should be the first choice
	Choice1 string `yaml:"choice1" yamltags:"oneOf=fooBar"`

	// Choice2 should be the second choice
	Choice2 string `yaml:"choice2" yamltags:"oneOf=fooBar"`
}
