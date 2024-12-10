// SPDX-FileCopyrightText: 2021 The Go Language Server Authors
// SPDX-License-Identifier: BSD-3-Clause

package protocol

// SelectionRangeProviderOptions selection range provider options interface.
type SelectionRangeProviderOptions interface{}

// SelectionRange represents a selection range represents a part of a selection hierarchy.
//
// A selection range may have a parent selection range that contains it.
//
// @since 3.15.0.
type SelectionRange struct {
	// Range is the Range of this selection range.
	Range Range `json:"range"`

	// Parent is the parent selection range containing this range. Therefore `parent.range` must contain this Range.
	Parent *SelectionRange `json:"parent,omitempty"`
}

// EnableSelectionRange is the whether the selection range.
type EnableSelectionRange bool

// compile time check whether the EnableSelectionRange implements a SelectionRangeProviderOptions interface.
var _ SelectionRangeProviderOptions = (*EnableSelectionRange)(nil)

// Value implements SelectionRangeProviderOptions interface.
func (v EnableSelectionRange) Value() interface{} {
	return bool(v)
}

// NewEnableSelectionRange returns the new EnableSelectionRange underlying types SelectionRangeProviderOptions.
func NewEnableSelectionRange(enable bool) SelectionRangeProviderOptions {
	v := EnableSelectionRange(enable)

	return &v
}

// SelectionRangeOptions is the server capability of selection range.
type SelectionRangeOptions struct {
	WorkDoneProgressOptions
}

// compile time check whether the EnableSelectionRange implements a SelectionRangeProviderOptions interface.
var _ SelectionRangeProviderOptions = (*EnableSelectionRange)(nil)

// Value implements SelectionRangeProviderOptions interface.
func (v *SelectionRangeOptions) Value() interface{} {
	return v
}

// NewSelectionRangeOptions returns the new SelectionRangeOptions underlying types SelectionRangeProviderOptions.
func NewSelectionRangeOptions(enableWorkDoneProgress bool) SelectionRangeProviderOptions {
	v := SelectionRangeOptions{
		WorkDoneProgressOptions: WorkDoneProgressOptions{
			WorkDoneProgress: enableWorkDoneProgress,
		},
	}

	return &v
}

// SelectionRangeRegistrationOptions is the server capability of selection range registration.
type SelectionRangeRegistrationOptions struct {
	SelectionRangeOptions
	TextDocumentRegistrationOptions
	StaticRegistrationOptions
}

// compile time check whether the SelectionRangeRegistrationOptions implements a SelectionRangeProviderOptions interface.
var _ SelectionRangeProviderOptions = (*SelectionRangeRegistrationOptions)(nil)

// Value implements SelectionRangeProviderOptions interface.
func (v *SelectionRangeRegistrationOptions) Value() interface{} {
	return v
}

// NewSelectionRangeRegistrationOptions returns the new SelectionRangeRegistrationOptions underlying types SelectionRangeProviderOptions.
func NewSelectionRangeRegistrationOptions(enableWorkDoneProgress bool, selector DocumentSelector, id string) SelectionRangeProviderOptions {
	v := SelectionRangeRegistrationOptions{
		SelectionRangeOptions: SelectionRangeOptions{
			WorkDoneProgressOptions: WorkDoneProgressOptions{
				WorkDoneProgress: enableWorkDoneProgress,
			},
		},
		TextDocumentRegistrationOptions: TextDocumentRegistrationOptions{
			DocumentSelector: selector,
		},
		StaticRegistrationOptions: StaticRegistrationOptions{
			ID: id,
		},
	}

	return &v
}

// SelectionRangeParams represents a parameter literal used in selection range requests.
//
// @since 3.15.0.
type SelectionRangeParams struct {
	WorkDoneProgressParams
	PartialResultParams

	// TextDocument is the text document.
	TextDocument TextDocumentIdentifier `json:"textDocument"`

	// Positions is the positions inside the text document.
	Positions []Position `json:"positions"`
}
