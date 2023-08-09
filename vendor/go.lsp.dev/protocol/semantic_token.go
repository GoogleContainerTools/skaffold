// SPDX-FileCopyrightText: 2021 The Go Language Server Authors
// SPDX-License-Identifier: BSD-3-Clause

package protocol

// SemanticTokenTypes represents a type of semantic token.
//
// @since 3.16.0.
type SemanticTokenTypes string

// list of SemanticTokenTypes.
const (
	SemanticTokenNamespace SemanticTokenTypes = "namespace"

	// Represents a generic type. Acts as a fallback for types which
	// can't be mapped to a specific type like class or enum.
	SemanticTokenType          SemanticTokenTypes = "type"
	SemanticTokenClass         SemanticTokenTypes = "class"
	SemanticTokenEnum          SemanticTokenTypes = "enum"
	SemanticTokenInterface     SemanticTokenTypes = "interface"
	SemanticTokenStruct        SemanticTokenTypes = "struct"
	SemanticTokenTypeParameter SemanticTokenTypes = "typeParameter"
	SemanticTokenParameter     SemanticTokenTypes = "parameter"
	SemanticTokenVariable      SemanticTokenTypes = "variable"
	SemanticTokenProperty      SemanticTokenTypes = "property"
	SemanticTokenEnumMember    SemanticTokenTypes = "enumMember"
	SemanticTokenEvent         SemanticTokenTypes = "event"
	SemanticTokenFunction      SemanticTokenTypes = "function"
	SemanticTokenMethod        SemanticTokenTypes = "method"
	SemanticTokenMacro         SemanticTokenTypes = "macro"
	SemanticTokenKeyword       SemanticTokenTypes = "keyword"
	SemanticTokenModifier      SemanticTokenTypes = "modifier"
	SemanticTokenComment       SemanticTokenTypes = "comment"
	SemanticTokenString        SemanticTokenTypes = "string"
	SemanticTokenNumber        SemanticTokenTypes = "number"
	SemanticTokenRegexp        SemanticTokenTypes = "regexp"
	SemanticTokenOperator      SemanticTokenTypes = "operator"
)

// SemanticTokenModifiers represents a modifiers of semantic token.
//
// @since 3.16.0.
type SemanticTokenModifiers string

// list of SemanticTokenModifiers.
const (
	SemanticTokenModifierDeclaration    SemanticTokenModifiers = "declaration"
	SemanticTokenModifierDefinition     SemanticTokenModifiers = "definition"
	SemanticTokenModifierReadonly       SemanticTokenModifiers = "readonly"
	SemanticTokenModifierStatic         SemanticTokenModifiers = "static"
	SemanticTokenModifierDeprecated     SemanticTokenModifiers = "deprecated"
	SemanticTokenModifierAbstract       SemanticTokenModifiers = "abstract"
	SemanticTokenModifierAsync          SemanticTokenModifiers = "async"
	SemanticTokenModifierModification   SemanticTokenModifiers = "modification"
	SemanticTokenModifierDocumentation  SemanticTokenModifiers = "documentation"
	SemanticTokenModifierDefaultLibrary SemanticTokenModifiers = "defaultLibrary"
)

// TokenFormat is an additional token format capability to allow future extensions of the format.
//
// @since 3.16.0.
type TokenFormat string

// TokenFormatRelative described using relative positions.
const TokenFormatRelative TokenFormat = "relative"

// SemanticTokensLegend is the on the capability level types and modifiers are defined using strings.
//
// However the real encoding happens using numbers.
//
// The server therefore needs to let the client know which numbers it is using for which types and modifiers.
//
// @since 3.16.0.
type SemanticTokensLegend struct {
	// TokenTypes is the token types a server uses.
	TokenTypes []SemanticTokenTypes `json:"tokenTypes"`

	// TokenModifiers is the token modifiers a server uses.
	TokenModifiers []SemanticTokenModifiers `json:"tokenModifiers"`
}

// SemanticTokensParams params for the SemanticTokensFull request.
//
// @since 3.16.0.
type SemanticTokensParams struct {
	WorkDoneProgressParams
	PartialResultParams

	// TextDocument is the text document.
	TextDocument TextDocumentIdentifier `json:"textDocument"`
}

// SemanticTokens is the result of SemanticTokensFull request.
//
// @since 3.16.0.
type SemanticTokens struct {
	// ResultID an optional result id. If provided and clients support delta updating
	// the client will include the result id in the next semantic token request.
	//
	// A server can then instead of computing all semantic tokens again simply
	// send a delta.
	ResultID string `json:"resultId,omitempty"`

	// Data is the actual tokens.
	Data []uint32 `json:"data"`
}

// SemanticTokensPartialResult is the partial result of SemanticTokensFull request.
//
// @since 3.16.0.
type SemanticTokensPartialResult struct {
	// Data is the actual tokens.
	Data []uint32 `json:"data"`
}

// SemanticTokensDeltaParams params for the SemanticTokensFullDelta request.
//
// @since 3.16.0.
type SemanticTokensDeltaParams struct {
	WorkDoneProgressParams
	PartialResultParams

	// TextDocument is the text document.
	TextDocument TextDocumentIdentifier `json:"textDocument"`

	// PreviousResultID is the result id of a previous response.
	//
	// The result Id can either point to a full response or a delta response depending on what was received last.
	PreviousResultID string `json:"previousResultId"`
}

// SemanticTokensDelta result of SemanticTokensFullDelta request.
//
// @since 3.16.0.
type SemanticTokensDelta struct {
	// ResultID is the result id.
	//
	// This field is readonly.
	ResultID string `json:"resultId,omitempty"`

	// Edits is the semantic token edits to transform a previous result into a new
	// result.
	Edits []SemanticTokensEdit `json:"edits"`
}

// SemanticTokensDeltaPartialResult is the partial result of SemanticTokensFullDelta request.
//
// @since 3.16.0.
type SemanticTokensDeltaPartialResult struct {
	Edits []SemanticTokensEdit `json:"edits"`
}

// SemanticTokensEdit is the semantic token edit.
//
// @since 3.16.0.
type SemanticTokensEdit struct {
	// Start is the start offset of the edit.
	Start uint32 `json:"start"`

	// DeleteCount is the count of elements to remove.
	DeleteCount uint32 `json:"deleteCount"`

	// Data is the elements to insert.
	Data []uint32 `json:"data,omitempty"`
}

// SemanticTokensRangeParams params for the SemanticTokensRange request.
//
// @since 3.16.0.
type SemanticTokensRangeParams struct {
	WorkDoneProgressParams
	PartialResultParams

	// TextDocument is the text document.
	TextDocument TextDocumentIdentifier `json:"textDocument"`

	// Range is the range the semantic tokens are requested for.
	Range Range `json:"range"`
}
