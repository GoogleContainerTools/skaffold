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

package status

import (
	"context"
	"io"
)

// Monitor is an interface for checking resource deployment status.
type Monitor interface {
	// Check runs the status check monitor for a deployment
	Check(context.Context, io.Writer) error
	// Reset executes any reset behavior required by the status monitor between dev loops.
	Reset()
}

type NoopMonitor struct{}

func (n *NoopMonitor) Check(context.Context, io.Writer) error { return nil }

func (n *NoopMonitor) Reset() {}
