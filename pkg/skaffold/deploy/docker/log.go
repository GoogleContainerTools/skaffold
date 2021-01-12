/*
Copyright 2020 The Skaffold Authors

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

package docker

import (
	"context"
	"time"
)

type Logger struct{}

func NewLogger() *Logger {
	return nil
}

func (l *Logger) Start(context.Context) error {
	// TODO(nkubala): implement
	return nil
}

func (l *Logger) Stop() {
	// TODO(nkubala): implement
}

func (l *Logger) Mute() {
	// TODO(nkubala): implement
}

func (l *Logger) Unmute() {
	// TODO(nkubala): implement
}

func (l *Logger) IsMuted() bool {
	// TODO(nkubala): implement
	return false
}

func (l *Logger) SetSince(time.Time) {
	// TODO(nkubala): implement
}
