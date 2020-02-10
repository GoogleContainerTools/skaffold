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

package analyze

// directoryAnalyzer is a base analyzer that can be included in every analyzer as a convenience
// it saves the current directory on enterDir events. Benefits to include this into other analyzers is that
// they can rely on the current directory var, but also they don't have to implement enterDir and exitDir.
type directoryAnalyzer struct {
	currentDir string
}

func (a *directoryAnalyzer) analyzeFile(_ string) error {
	return nil
}

func (a *directoryAnalyzer) enterDir(dir string) {
	a.currentDir = dir
}

func (a *directoryAnalyzer) exitDir(_ string) {
	//pass
}
