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
	"sync"
)

// DocumentManager manages syncing documents for the LSP server
type DocumentManager struct {
	documents map[string]string
	mtx       sync.RWMutex
}

// NewDocumentManager creates a new DocumentManager object
func NewDocumentManager() *DocumentManager {
	return &DocumentManager{
		documents: make(map[string]string),
	}
}

// UpdateDocument updates the string value for a document
func (m *DocumentManager) UpdateDocument(documentURI string, doc string) {
	m.mtx.Lock()
	defer m.mtx.Unlock()
	m.documents[documentURI] = doc
}

// GetDocument gets the string value for a document
func (m *DocumentManager) GetDocument(documentURI string) string {
	m.mtx.RLock()
	defer m.mtx.RUnlock()
	if doc, ok := m.documents[documentURI]; ok {
		return doc
	}
	return ""
}
