/*
Copyright 2019 The Kubernetes Authors.

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

package log

// NoopLogger implements the Logger interface and never logs anything
type NoopLogger struct{}

// Warn meets the Logger interface but does nothing
func (n NoopLogger) Warn(message string) {}

// Warnf meets the Logger interface but does nothing
func (n NoopLogger) Warnf(format string, args ...interface{}) {}

// Error meets the Logger interface but does nothing
func (n NoopLogger) Error(message string) {}

// Errorf meets the Logger interface but does nothing
func (n NoopLogger) Errorf(format string, args ...interface{}) {}

// V meets the Logger interface but does nothing
func (n NoopLogger) V(level Level) InfoLogger { return NoopInfoLogger{} }

// NoopInfoLogger implements the InfoLogger interface and never logs anything
type NoopInfoLogger struct{}

// Enabled meets the InfoLogger interface but always returns false
func (n NoopInfoLogger) Enabled() bool { return false }

// Info meets the InfoLogger interface but does nothing
func (n NoopInfoLogger) Info(message string) {}

// Infof meets the InfoLogger interface but does nothing
func (n NoopInfoLogger) Infof(format string, args ...interface{}) {}
