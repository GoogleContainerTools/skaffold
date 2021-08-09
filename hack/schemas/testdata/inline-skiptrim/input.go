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

package latest

// TestStruct for testing the schema generator.
type TestStruct struct {

	// RequiredField should be required
	RequiredField string `yaml:"reqField" yamltags:"required"`

	// AnotherField has reference to InlineStruct
	AnotherField *InlineStruct `yaml:"anotherField"`
}

// AnotherTestStruct for testing the schema s generator.
type AnotherTestStruct struct {
	InlineStruct `yaml:"inline" yamltags:"skipTrim"`
}

// InlineStruct is embedded inline into TestStruct
type InlineStruct struct {

	// Field1 should be the first choice
	Field1 string `yaml:"f1"`

	// Field2 should be the second choice
	Field2 string `yaml:"f2"`
}
