// SPDX-FileCopyrightText: 2021 The Go Language Server Authors
// SPDX-License-Identifier: BSD-3-Clause

package protocol

// CallHierarchy capabilities specific to the "textDocument/callHierarchy".
//
// @since 3.16.0.
type CallHierarchy struct {
	// DynamicRegistration whether implementation supports dynamic registration.
	//
	// If this is set to "true" the client supports the new
	// TextDocumentRegistrationOptions && StaticRegistrationOptions return
	// value for the corresponding server capability as well.
	DynamicRegistration bool `json:"dynamicRegistration,omitempty"`
}

// CallHierarchyPrepareParams params of CallHierarchyPrepare.
//
// @since 3.16.0.
type CallHierarchyPrepareParams struct {
	TextDocumentPositionParams
	WorkDoneProgressParams
}

// CallHierarchyItem is the result of a "textDocument/prepareCallHierarchy" request.
//
// @since 3.16.0.
type CallHierarchyItem struct {
	// name is the name of this item.
	Name string `json:"name"`

	// Kind is the kind of this item.
	Kind SymbolKind `json:"kind"`

	// Tags for this item.
	Tags []SymbolTag `json:"tags,omitempty"`

	// Detail more detail for this item, e.g. the signature of a function.
	Detail string `json:"detail,omitempty"`

	// URI is the resource identifier of this item.
	URI DocumentURI `json:"uri"`

	// Range is the range enclosing this symbol not including leading/trailing whitespace
	// but everything else, e.g. comments and code.
	Range Range `json:"range"`

	// SelectionRange is the range that should be selected and revealed when this symbol is being
	// picked, e.g. the name of a function. Must be contained by the
	// Range.
	SelectionRange Range `json:"selectionRange"`

	// Data is a data entry field that is preserved between a call hierarchy prepare and
	// incoming calls or outgoing calls requests.
	Data interface{} `json:"data,omitempty"`
}

// CallHierarchyIncomingCallsParams params of CallHierarchyIncomingCalls.
//
// @since 3.16.0.
type CallHierarchyIncomingCallsParams struct {
	WorkDoneProgressParams
	PartialResultParams

	// Item is the IncomingCalls item.
	Item CallHierarchyItem `json:"item"`
}

// CallHierarchyIncomingCall is the result of a "callHierarchy/incomingCalls" request.
//
// @since 3.16.0.
type CallHierarchyIncomingCall struct {
	// From is the item that makes the call.
	From CallHierarchyItem `json:"from"`

	// FromRanges is the ranges at which the calls appear. This is relative to the caller
	// denoted by From.
	FromRanges []Range `json:"fromRanges"`
}

// CallHierarchyOutgoingCallsParams params of CallHierarchyOutgoingCalls.
//
// @since 3.16.0.
type CallHierarchyOutgoingCallsParams struct {
	WorkDoneProgressParams
	PartialResultParams

	// Item is the OutgoingCalls item.
	Item CallHierarchyItem `json:"item"`
}

// CallHierarchyOutgoingCall is the result of a "callHierarchy/outgoingCalls" request.
//
// @since 3.16.0.
type CallHierarchyOutgoingCall struct {
	// To is the item that is called.
	To CallHierarchyItem `json:"to"`

	// FromRanges is the range at which this item is called. This is the range relative to
	// the caller, e.g the item passed to "callHierarchy/outgoingCalls" request.
	FromRanges []Range `json:"fromRanges"`
}
