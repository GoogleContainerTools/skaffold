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

package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sort"
	"strings"
	"sync"
)

type LogLine struct {
	Action  string
	Test    string
	Package string
	Output  string
	Elapsed float32
}

func main() {
	if err := goTest(os.Args[1:]); err != nil {
		os.Exit(1)
	}
}

func goTest(testArgs []string) error {
	args := append([]string{"test", "-json"}, testArgs...)
	verbose := isVerbose(testArgs)

	cmd := exec.CommandContext(context.Background(), "go", args...)

	pr, pw := io.Pipe()
	cmd.Stderr = pw
	cmd.Stdout = pw

	failedTests := map[string]bool{}
	var failedLogs []LogLine
	var allLogs []LogLine

	var wc sync.WaitGroup
	wc.Add(1)

	go func() {
		defer wc.Done()

		// Print logs while tests are running
		scanner := bufio.NewScanner(pr)
		for i := 0; scanner.Scan(); i++ {
			line := scanner.Bytes()

			var l LogLine
			if err := json.Unmarshal(line, &l); err != nil {
				// Sometimes, `go test -json` outputs plain text instead of json.
				// For example in case of a build error.
				fmt.Println(failInRed(string(line)))
				continue
			}

			allLogs = append(allLogs, l)

			if l.Action == "output" {
				if verbose || (l.Test == "" && l.Output != "PASS\n" && l.Output != "FAIL\n" && !strings.HasPrefix(l.Output, "coverage:") && !strings.Contains(l.Output, "[no test files]")) {
					fmt.Print(failInRed(l.Output))
				}
			}

			// Is this an error?
			if (l.Action == "fail" || strings.Contains(l.Output, "FAIL")) && l.Test != "" {
				if failedTests[l.Package+"/"+l.Test] {
					continue
				}
				failedTests[l.Package+"/"+l.Test] = true

				failedLogs = append(failedLogs, l)
			}
		}

		// Print detailed information about failures.
		if len(failedLogs) > 0 {
			fmt.Println(red("\n=== Failed Tests ==="))

			for _, l := range failedLogs {
				fmt.Println(bold(trimPackage(l.Package) + "/" + l.Test))

				for _, l := range logsForTest(l.Test, l.Package, allLogs) {
					if l.Action == "output" && l.Output != "" && !strings.HasPrefix(l.Output, "=== RUN") {
						fmt.Print(failInRed(l.Output))
					}
				}
			}
		}

		// Print top slowest tests.
		fmt.Println(yellow("\n=== Slow Tests ==="))
		for _, l := range topSlowest(20, allLogs) {
			fmt.Printf("%.2fs\t%s\n", l.Elapsed, l.Test)
		}
	}()

	err := cmd.Run()
	if err != nil {
		pr.CloseWithError(err)
	} else {
		pr.Close()
	}
	wc.Wait()
	return err
}

func failInRed(msg string) string {
	return strings.ReplaceAll(msg, "FAIL", red("FAIL"))
}

func red(msg string) string {
	return "\033[0;31m" + msg + "\033[0m"
}

func yellow(msg string) string {
	return "\033[0;33m" + msg + "\033[0m"
}

func bold(msg string) string {
	return "\033[1m" + msg + "\033[0m"
}

func trimPackage(pkg string) string {
	return strings.TrimPrefix(pkg, "github.com/GoogleContainerTools/skaffold")
}

func isVerbose(args []string) bool {
	for _, arg := range args {
		if arg == "-v" {
			return true
		}
	}

	return false
}

func logsForTest(test, pkg string, all []LogLine) []LogLine {
	var forTest []LogLine

	for _, l := range all {
		if l.Package == pkg && l.Test == test {
			forTest = append(forTest, l)
		}
	}

	return forTest
}

func topSlowest(max int, all []LogLine) []LogLine {
	var top []LogLine

	for _, l := range all {
		if l.Test != "" && l.Elapsed > 0 {
			top = append(top, l)
		}
	}

	sort.Slice(top, func(i, j int) bool { return top[i].Elapsed > top[j].Elapsed })

	if len(top) <= max {
		return top
	}
	return top[0:max]
}
