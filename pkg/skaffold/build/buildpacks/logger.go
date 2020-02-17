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

package buildpacks

import (
	"io"

	"github.com/buildpacks/pack/logging"
	"github.com/mattn/go-colorable"
	"github.com/sirupsen/logrus"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
)

// logger exists to meet the requirements of the pack logger.
type logger struct {
	*logrus.Logger
	out io.Writer
}

func NewLogger(out io.Writer) logging.Logger {
	// If out is not a terminal, let's make sure no colors are printed.
	if _, isTerm := util.IsTerminal(out); !isTerm {
		out = colorable.NewNonColorable(out)
	}

	l := logrus.New()
	l.SetOutput(out)

	// By default, logrus prefixes lines with 'INFO[XXX]'.
	l.SetFormatter(new(plainFormatter))

	return &logger{
		Logger: l,
		out:    out,
	}
}

type plainFormatter struct{}

func (f *plainFormatter) Format(entry *logrus.Entry) ([]byte, error) {
	return []byte(entry.Message + "\n"), nil
}

func (l *logger) Debug(msg string) {
	l.Logger.Debug(msg)
}

func (l *logger) Info(msg string) {
	l.Logger.Info(msg)
}

func (l *logger) Warn(msg string) {
	l.Logger.Warn(msg)
}

func (l *logger) Error(msg string) {
	l.Logger.Error(msg)
}

func (l *logger) Writer() io.Writer {
	return l.out
}

func (l *logger) IsVerbose() bool {
	return false
}
