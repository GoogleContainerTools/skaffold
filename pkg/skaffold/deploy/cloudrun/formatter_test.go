/*
Copyright 2022 The Skaffold Authors

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

package cloudrun

import (
	"bufio"
	"bytes"
	"fmt"
	"testing"

	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/output"
)

func TestLogFormatter_Name(t *testing.T) {
	serviceName := "test"
	formatter := LogFormatter{prefix: serviceName, outputColor: output.Red}
	if formatter.Name() != serviceName {
		t.Fatalf("expected service name to be %s but got %s", serviceName, formatter.Name())
	}
}

func TestLogFormatter_Printline(t *testing.T) {
	testLog := "Testing log output"
	serviceName := "test"
	var b bytes.Buffer
	writer := bufio.NewWriter(&b)
	formatter := LogFormatter{prefix: serviceName, outputColor: output.Red}
	formatter.PrintLine(writer, testLog)
	writer.Flush()
	result := b.Bytes()
	expectedLogOutput := fmt.Sprintf("[%s] %s", serviceName, testLog)
	if string(result) != expectedLogOutput {
		t.Fatalf("expected log output to be %s but got %s", expectedLogOutput, result)
	}
}
