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
package app

import (
	"errors"
	"regexp"
)

type ExitCoder interface {
	ExitCode() int
}

// ExitCode extracts the exit code from the error.
func ExitCode(err error) int {
	var exitCoder ExitCoder
	if errors.As(err, &exitCoder) {
		return exitCoder.ExitCode()
	}
	return 1
}

type invalidUsageError struct{ err error }

func (i invalidUsageError) Unwrap() error { return i.err }
func (i invalidUsageError) Error() string { return i.err.Error() }
func (i invalidUsageError) ExitCode() int { return 127 }

// compiled list of common validation error prefixes from cobra/args.go and cobra/command.go based on skaffold's usage
var cobraUsageErrorPatterns = []*regexp.Regexp{
	regexp.MustCompile(`^unknown command`),
	regexp.MustCompile(`^unknown( shorthand)? flag`),
	regexp.MustCompile(`^flag needs an argument:`),
	regexp.MustCompile(`^invalid argument `),
	regexp.MustCompile(`^accepts.*, received `),
}

func extractInvalidUsageError(err error) error {
	if err == nil {
		return nil
	}
	for _, pattern := range cobraUsageErrorPatterns {
		if pattern.MatchString(err.Error()) {
			return invalidUsageError{err}
		}
	}
	return err
}
