// SPDX-FileCopyrightText: 2019 The Go Language Server Authors
// SPDX-License-Identifier: BSD-3-Clause

package protocol

import (
	"strconv"
)

// DidOpenTextDocumentParams params of DidOpenTextDocument notification.
type DidOpenTextDocumentParams struct {
	// TextDocument is the document that was opened.
	TextDocument TextDocumentItem `json:"textDocument"`
}

// DidChangeTextDocumentParams params of DidChangeTextDocument notification.
type DidChangeTextDocumentParams struct {
	// TextDocument is the document that did change. The version number points
	// to the version after all provided content changes have
	// been applied.
	TextDocument VersionedTextDocumentIdentifier `json:"textDocument"`

	// ContentChanges is the actual content changes. The content changes describe single state changes
	// to the document. So if there are two content changes c1 and c2 for a document
	// in state S then c1 move the document to S' and c2 to S''.
	ContentChanges []TextDocumentContentChangeEvent `json:"contentChanges"` // []TextDocumentContentChangeEvent | text
}

// TextDocumentSaveReason represents reasons why a text document is saved.
type TextDocumentSaveReason float64

const (
	// TextDocumentSaveReasonManual is the manually triggered, e.g. by the user pressing save, by starting debugging,
	// or by an API call.
	TextDocumentSaveReasonManual TextDocumentSaveReason = 1

	// TextDocumentSaveReasonAfterDelay is the automatic after a delay.
	TextDocumentSaveReasonAfterDelay TextDocumentSaveReason = 2

	// TextDocumentSaveReasonFocusOut when the editor lost focus.
	TextDocumentSaveReasonFocusOut TextDocumentSaveReason = 3
)

// String implements fmt.Stringer.
func (t TextDocumentSaveReason) String() string {
	switch t {
	case TextDocumentSaveReasonManual:
		return "Manual"
	case TextDocumentSaveReasonAfterDelay:
		return "AfterDelay"
	case TextDocumentSaveReasonFocusOut:
		return "FocusOut"
	default:
		return strconv.FormatFloat(float64(t), 'f', -10, 64)
	}
}

// TextDocumentChangeRegistrationOptions describe options to be used when registering for text document change events.
type TextDocumentChangeRegistrationOptions struct {
	TextDocumentRegistrationOptions

	// SyncKind how documents are synced to the server. See TextDocumentSyncKind.Full
	// and TextDocumentSyncKind.Incremental.
	SyncKind TextDocumentSyncKind `json:"syncKind"`
}

// WillSaveTextDocumentParams is the parameters send in a will save text document notification.
type WillSaveTextDocumentParams struct {
	// TextDocument is the document that will be saved.
	TextDocument TextDocumentIdentifier `json:"textDocument"`

	// Reason is the 'TextDocumentSaveReason'.
	Reason TextDocumentSaveReason `json:"reason,omitempty"`
}

// DidSaveTextDocumentParams params of DidSaveTextDocument notification.
type DidSaveTextDocumentParams struct {
	// Text optional the content when saved. Depends on the includeText value
	// when the save notification was requested.
	Text string `json:"text,omitempty"`

	// TextDocument is the document that was saved.
	TextDocument TextDocumentIdentifier `json:"textDocument"`
}

// TextDocumentContentChangeEvent an event describing a change to a text document. If range and rangeLength are omitted
// the new text is considered to be the full content of the document.
type TextDocumentContentChangeEvent struct {
	// Range is the range of the document that changed.
	Range Range `json:"range"`

	// RangeLength is the length of the range that got replaced.
	RangeLength uint32 `json:"rangeLength,omitempty"`

	// Text is the new text of the document.
	Text string `json:"text"`
}

// TextDocumentSaveRegistrationOptions TextDocumentSave Registration options.
type TextDocumentSaveRegistrationOptions struct {
	TextDocumentRegistrationOptions

	// IncludeText is the client is supposed to include the content on save.
	IncludeText bool `json:"includeText,omitempty"`
}

// DidCloseTextDocumentParams params of DidCloseTextDocument notification.
type DidCloseTextDocumentParams struct {
	// TextDocument the document that was closed.
	TextDocument TextDocumentIdentifier `json:"textDocument"`
}
