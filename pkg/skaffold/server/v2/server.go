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

package v2

var (
	Srv *Server
)

type Server struct {
	BuildIntentCallback   func()
	SyncIntentCallback    func()
	DeployIntentCallback  func()
	DevloopIntentCallback func()
	AutoBuildCallback     func(bool)
	AutoSyncCallback      func(bool)
	AutoDeployCallback    func(bool)
	AutoDevloopCallback   func(bool)
}

// TODO(marlongamez): Add Set*Callback() funcs once going for v1 feature parity
