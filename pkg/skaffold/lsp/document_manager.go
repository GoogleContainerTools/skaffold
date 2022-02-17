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

package lsp

import (
	"github.com/spf13/afero"
)

// DocumentManager manages syncing memMapFs for the LSP server
type DocumentManager struct {
	memMapFs afero.Fs
}

// NewDocumentManager creates a new DocumentManager object
func NewDocumentManager(fs afero.Fs) *DocumentManager {
	return &DocumentManager{
		memMapFs: fs,
	}
}

// UpdateDocument updates the string value for a memMapFs
func (m *DocumentManager) UpdateDocument(documentURI string, doc string) {
	afero.WriteFile(m.memMapFs, documentURI, []byte(doc), 0644)
}

// GetDocument gets the string value for a memMapFs
func (m *DocumentManager) GetDocument(documentURI string) string {
	b, err := afero.ReadFile(m.memMapFs, documentURI)
	if err != nil {
		return ""
	}
	return string(b)
}
