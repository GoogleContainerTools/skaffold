// SPDX-FileCopyrightText: 2019 The Go Language Server Authors
// SPDX-License-Identifier: BSD-3-Clause

package protocol

import (
	"strconv"
)

// CompletionParams params of Completion request.
type CompletionParams struct {
	TextDocumentPositionParams
	WorkDoneProgressParams
	PartialResultParams

	// Context is the completion context. This is only available if the client specifies
	// to send this using `ClientCapabilities.textDocument.completion.contextSupport === true`
	Context *CompletionContext `json:"context,omitempty"`
}

// CompletionTriggerKind how a completion was triggered.
type CompletionTriggerKind float64

const (
	// CompletionTriggerKindInvoked completion was triggered by typing an identifier (24x7 code
	// complete), manual invocation (e.g Ctrl+Space) or via API.
	CompletionTriggerKindInvoked CompletionTriggerKind = 1

	// CompletionTriggerKindTriggerCharacter completion was triggered by a trigger character specified by
	// the `triggerCharacters` properties of the `CompletionRegistrationOptions`.
	CompletionTriggerKindTriggerCharacter CompletionTriggerKind = 2

	// CompletionTriggerKindTriggerForIncompleteCompletions completion was re-triggered as the current completion list is incomplete.
	CompletionTriggerKindTriggerForIncompleteCompletions CompletionTriggerKind = 3
)

// String implements fmt.Stringer.
func (k CompletionTriggerKind) String() string {
	switch k {
	case CompletionTriggerKindInvoked:
		return "Invoked"
	case CompletionTriggerKindTriggerCharacter:
		return "TriggerCharacter"
	case CompletionTriggerKindTriggerForIncompleteCompletions:
		return "TriggerForIncompleteCompletions"
	default:
		return strconv.FormatFloat(float64(k), 'f', -10, 64)
	}
}

// CompletionContext contains additional information about the context in which a completion request is triggered.
type CompletionContext struct {
	// TriggerCharacter is the trigger character (a single character) that has trigger code complete.
	// Is undefined if `triggerKind !== CompletionTriggerKind.TriggerCharacter`
	TriggerCharacter string `json:"triggerCharacter,omitempty"`

	// TriggerKind how the completion was triggered.
	TriggerKind CompletionTriggerKind `json:"triggerKind"`
}

// CompletionList represents a collection of [completion items](#CompletionItem) to be presented
// in the editor.
type CompletionList struct {
	// IsIncomplete this list it not complete. Further typing should result in recomputing
	// this list.
	IsIncomplete bool `json:"isIncomplete"`

	// Items is the completion items.
	Items []CompletionItem `json:"items"`
}

// InsertTextFormat defines whether the insert text in a completion item should be interpreted as
// plain text or a snippet.
type InsertTextFormat float64

const (
	// InsertTextFormatPlainText is the primary text to be inserted is treated as a plain string.
	InsertTextFormatPlainText InsertTextFormat = 1

	// InsertTextFormatSnippet is the primary text to be inserted is treated as a snippet.
	//
	// A snippet can define tab stops and placeholders with `$1`, `$2`
	// and `${3:foo}`. `$0` defines the final tab stop, it defaults to
	// the end of the snippet. Placeholders with equal identifiers are linked,
	// that is typing in one will update others too.
	InsertTextFormatSnippet InsertTextFormat = 2
)

// String implements fmt.Stringer.
func (tf InsertTextFormat) String() string {
	switch tf {
	case InsertTextFormatPlainText:
		return "PlainText"
	case InsertTextFormatSnippet:
		return "Snippet"
	default:
		return strconv.FormatFloat(float64(tf), 'f', -10, 64)
	}
}

// InsertReplaceEdit is a special text edit to provide an insert and a replace operation.
//
// @since 3.16.0.
type InsertReplaceEdit struct {
	// NewText is the string to be inserted.
	NewText string `json:"newText"`

	// Insert is the range if the insert is requested.
	Insert Range `json:"insert"`

	// Replace is the range if the replace is requested.
	Replace Range `json:"replace"`
}

// InsertTextMode how whitespace and indentation is handled during completion
// item insertion.
//
// @since 3.16.0.
type InsertTextMode float64

const (
	// AsIs is the insertion or replace strings is taken as it is. If the
	// value is multi line the lines below the cursor will be
	// inserted using the indentation defined in the string value.
	// The client will not apply any kind of adjustments to the
	// string.
	InsertTextModeAsIs InsertTextMode = 1

	// AdjustIndentation is the editor adjusts leading whitespace of new lines so that
	// they match the indentation up to the cursor of the line for
	// which the item is accepted.
	//
	// Consider a line like this: <2tabs><cursor><3tabs>foo. Accepting a
	// multi line completion item is indented using 2 tabs and all
	// following lines inserted will be indented using 2 tabs as well.
	InsertTextModeAdjustIndentation InsertTextMode = 2
)

// String returns a string representation of the InsertTextMode.
func (k InsertTextMode) String() string {
	switch k {
	case InsertTextModeAsIs:
		return "AsIs"
	case InsertTextModeAdjustIndentation:
		return "AdjustIndentation"
	default:
		return strconv.FormatFloat(float64(k), 'f', -10, 64)
	}
}

// CompletionItem item of CompletionList.
type CompletionItem struct {
	// AdditionalTextEdits an optional array of additional text edits that are applied when
	// selecting this completion. Edits must not overlap (including the same insert position)
	// with the main edit nor with themselves.
	//
	// Additional text edits should be used to change text unrelated to the current cursor position
	// (for example adding an import statement at the top of the file if the completion item will
	// insert an unqualified type).
	AdditionalTextEdits []TextEdit `json:"additionalTextEdits,omitempty"`

	// Command an optional command that is executed *after* inserting this completion. *Note* that
	// additional modifications to the current document should be described with the
	// additionalTextEdits-property.
	Command *Command `json:"command,omitempty"`

	// CommitCharacters an optional set of characters that when pressed while this completion is active will accept it first and
	// then type that character. *Note* that all commit characters should have `length=1` and that superfluous
	// characters will be ignored.
	CommitCharacters []string `json:"commitCharacters,omitempty"`

	// Tags is the tag for this completion item.
	//
	// @since 3.15.0.
	Tags []CompletionItemTag `json:"tags,omitempty"`

	// Data an data entry field that is preserved on a completion item between
	// a completion and a completion resolve request.
	Data interface{} `json:"data,omitempty"`

	// Deprecated indicates if this item is deprecated.
	Deprecated bool `json:"deprecated,omitempty"`

	// Detail a human-readable string with additional information
	// about this item, like type or symbol information.
	Detail string `json:"detail,omitempty"`

	// Documentation a human-readable string that represents a doc-comment.
	Documentation interface{} `json:"documentation,omitempty"`

	// FilterText a string that should be used when filtering a set of
	// completion items. When `falsy` the label is used.
	FilterText string `json:"filterText,omitempty"`

	// InsertText a string that should be inserted into a document when selecting
	// this completion. When `falsy` the label is used.
	//
	// The `insertText` is subject to interpretation by the client side.
	// Some tools might not take the string literally. For example
	// VS Code when code complete is requested in this example `con<cursor position>`
	// and a completion item with an `insertText` of `console` is provided it
	// will only insert `sole`. Therefore it is recommended to use `textEdit` instead
	// since it avoids additional client side interpretation.
	InsertText string `json:"insertText,omitempty"`

	// InsertTextFormat is the format of the insert text. The format applies to both the `insertText` property
	// and the `newText` property of a provided `textEdit`.
	InsertTextFormat InsertTextFormat `json:"insertTextFormat,omitempty"`

	// InsertTextMode how whitespace and indentation is handled during completion
	// item insertion. If not provided the client's default value depends on
	// the `textDocument.completion.insertTextMode` client capability.
	//
	// @since 3.16.0.
	InsertTextMode InsertTextMode `json:"insertTextMode,omitempty"`

	// Kind is the kind of this completion item. Based of the kind
	// an icon is chosen by the editor.
	Kind CompletionItemKind `json:"kind,omitempty"`

	// Label is the label of this completion item. By default
	// also the text that is inserted when selecting
	// this completion.
	Label string `json:"label"`

	// Preselect select this item when showing.
	//
	// *Note* that only one completion item can be selected and that the
	// tool / client decides which item that is. The rule is that the *first*
	// item of those that match best is selected.
	Preselect bool `json:"preselect,omitempty"`

	// SortText a string that should be used when comparing this item
	// with other items. When `falsy` the label is used.
	SortText string `json:"sortText,omitempty"`

	// TextEdit an edit which is applied to a document when selecting this completion. When an edit is provided the value of
	// `insertText` is ignored.
	//
	// NOTE: The range of the edit must be a single line range and it must contain the position at which completion
	// has been requested.
	//
	// Most editors support two different operations when accepting a completion
	// item. One is to insert a completion text and the other is to replace an
	// existing text with a completion text. Since this can usually not be
	// predetermined by a server it can report both ranges. Clients need to
	// signal support for `InsertReplaceEdits` via the
	// "textDocument.completion.insertReplaceSupport" client capability
	// property.
	//
	// NOTE 1: The text edit's range as well as both ranges from an insert
	// replace edit must be a [single line] and they must contain the position
	// at which completion has been requested.
	//
	// NOTE 2: If an "InsertReplaceEdit" is returned the edit's insert range
	// must be a prefix of the edit's replace range, that means it must be
	// contained and starting at the same position.
	//
	// @since 3.16.0 additional type "InsertReplaceEdit".
	TextEdit *TextEdit `json:"textEdit,omitempty"` // *TextEdit | *InsertReplaceEdit
}

// CompletionItemKind is the completion item kind values the client supports. When this
// property exists the client also guarantees that it will
// handle values outside its set gracefully and falls back
// to a default value when unknown.
//
// If this property is not present the client only supports
// the completion items kinds from `Text` to `Reference` as defined in
// the initial version of the protocol.
type CompletionItemKind float64

const (
	// CompletionItemKindText text completion kind.
	CompletionItemKindText CompletionItemKind = 1
	// CompletionItemKindMethod method completion kind.
	CompletionItemKindMethod CompletionItemKind = 2
	// CompletionItemKindFunction function completion kind.
	CompletionItemKindFunction CompletionItemKind = 3
	// CompletionItemKindConstructor constructor completion kind.
	CompletionItemKindConstructor CompletionItemKind = 4
	// CompletionItemKindField field completion kind.
	CompletionItemKindField CompletionItemKind = 5
	// CompletionItemKindVariable variable completion kind.
	CompletionItemKindVariable CompletionItemKind = 6
	// CompletionItemKindClass class completion kind.
	CompletionItemKindClass CompletionItemKind = 7
	// CompletionItemKindInterface interface completion kind.
	CompletionItemKindInterface CompletionItemKind = 8
	// CompletionItemKindModule module completion kind.
	CompletionItemKindModule CompletionItemKind = 9
	// CompletionItemKindProperty property completion kind.
	CompletionItemKindProperty CompletionItemKind = 10
	// CompletionItemKindUnit unit completion kind.
	CompletionItemKindUnit CompletionItemKind = 11
	// CompletionItemKindValue value completion kind.
	CompletionItemKindValue CompletionItemKind = 12
	// CompletionItemKindEnum enum completion kind.
	CompletionItemKindEnum CompletionItemKind = 13
	// CompletionItemKindKeyword keyword completion kind.
	CompletionItemKindKeyword CompletionItemKind = 14
	// CompletionItemKindSnippet snippet completion kind.
	CompletionItemKindSnippet CompletionItemKind = 15
	// CompletionItemKindColor color completion kind.
	CompletionItemKindColor CompletionItemKind = 16
	// CompletionItemKindFile file completion kind.
	CompletionItemKindFile CompletionItemKind = 17
	// CompletionItemKindReference reference completion kind.
	CompletionItemKindReference CompletionItemKind = 18
	// CompletionItemKindFolder folder completion kind.
	CompletionItemKindFolder CompletionItemKind = 19
	// CompletionItemKindEnumMember enum member completion kind.
	CompletionItemKindEnumMember CompletionItemKind = 20
	// CompletionItemKindConstant constant completion kind.
	CompletionItemKindConstant CompletionItemKind = 21
	// CompletionItemKindStruct struct completion kind.
	CompletionItemKindStruct CompletionItemKind = 22
	// CompletionItemKindEvent event completion kind.
	CompletionItemKindEvent CompletionItemKind = 23
	// CompletionItemKindOperator operator completion kind.
	CompletionItemKindOperator CompletionItemKind = 24
	// CompletionItemKindTypeParameter type parameter completion kind.
	CompletionItemKindTypeParameter CompletionItemKind = 25
)

// String implements fmt.Stringer.
//nolint:cyclop
func (k CompletionItemKind) String() string {
	switch k {
	case CompletionItemKindText:
		return "Text"
	case CompletionItemKindMethod:
		return "Method"
	case CompletionItemKindFunction:
		return "Function"
	case CompletionItemKindConstructor:
		return "Constructor"
	case CompletionItemKindField:
		return "Field"
	case CompletionItemKindVariable:
		return "Variable"
	case CompletionItemKindClass:
		return "Class"
	case CompletionItemKindInterface:
		return "Interface"
	case CompletionItemKindModule:
		return "Module"
	case CompletionItemKindProperty:
		return "Property"
	case CompletionItemKindUnit:
		return "Unit"
	case CompletionItemKindValue:
		return "Value"
	case CompletionItemKindEnum:
		return "Enum"
	case CompletionItemKindKeyword:
		return "Keyword"
	case CompletionItemKindSnippet:
		return "Snippet"
	case CompletionItemKindColor:
		return "Color"
	case CompletionItemKindFile:
		return "File"
	case CompletionItemKindReference:
		return "Reference"
	case CompletionItemKindFolder:
		return "Folder"
	case CompletionItemKindEnumMember:
		return "EnumMember"
	case CompletionItemKindConstant:
		return "Constant"
	case CompletionItemKindStruct:
		return "Struct"
	case CompletionItemKindEvent:
		return "Event"
	case CompletionItemKindOperator:
		return "Operator"
	case CompletionItemKindTypeParameter:
		return "TypeParameter"
	default:
		return strconv.FormatFloat(float64(k), 'f', -10, 64)
	}
}

// CompletionItemTag completion item tags are extra annotations that tweak the rendering of a completion
// item.
//
// @since 3.15.0.
type CompletionItemTag float64

// list of CompletionItemTag.
const (
	// CompletionItemTagDeprecated is the render a completion as obsolete, usually using a strike-out.
	CompletionItemTagDeprecated CompletionItemTag = 1
)

// String returns a string representation of the type.
func (c CompletionItemTag) String() string {
	switch c {
	case CompletionItemTagDeprecated:
		return "Deprecated"
	default:
		return strconv.FormatFloat(float64(c), 'f', -10, 64)
	}
}

// CompletionRegistrationOptions CompletionRegistration options.
type CompletionRegistrationOptions struct {
	TextDocumentRegistrationOptions

	// TriggerCharacters most tools trigger completion request automatically without explicitly requesting
	// it using a keyboard shortcut (e.g. Ctrl+Space). Typically they do so when the user
	// starts to type an identifier. For example if the user types `c` in a JavaScript file
	// code complete will automatically pop up present `console` besides others as a
	// completion item. Characters that make up identifiers don't need to be listed here.
	//
	// If code complete should automatically be trigger on characters not being valid inside
	// an identifier (for example `.` in JavaScript) list them in `triggerCharacters`.
	TriggerCharacters []string `json:"triggerCharacters,omitempty"`

	// ResolveProvider is the server provides support to resolve additional
	// information for a completion item.
	ResolveProvider bool `json:"resolveProvider,omitempty"`
}

// HoverParams params of Hover request.
//
// @since 3.15.0.
type HoverParams struct {
	TextDocumentPositionParams
	WorkDoneProgressParams
}

// Hover is the result of a hover request.
type Hover struct {
	// Contents is the hover's content
	Contents MarkupContent `json:"contents"`

	// Range an optional range is a range inside a text document
	// that is used to visualize a hover, e.g. by changing the background color.
	Range *Range `json:"range,omitempty"`
}

// SignatureHelpParams params of SignatureHelp request.
//
// @since 3.15.0.
type SignatureHelpParams struct {
	TextDocumentPositionParams
	WorkDoneProgressParams

	// context is the signature help context.
	//
	// This is only available if the client specifies to send this using the
	// client capability `textDocument.signatureHelp.contextSupport === true`.
	//
	// @since 3.15.0.
	Context *SignatureHelpContext `json:"context,omitempty"`
}

// SignatureHelpTriggerKind is the how a signature help was triggered.
//
// @since 3.15.0.
type SignatureHelpTriggerKind float64

// list of SignatureHelpTriggerKind.
const (
	// SignatureHelpTriggerKindInvoked is the signature help was invoked manually by the user or by a command.
	SignatureHelpTriggerKindInvoked SignatureHelpTriggerKind = 1

	// SignatureHelpTriggerKindTriggerCharacter is the signature help was triggered by a trigger character.
	SignatureHelpTriggerKindTriggerCharacter SignatureHelpTriggerKind = 2

	// SignatureHelpTriggerKindContentChange is the signature help was triggered by the cursor moving or
	// by the document content changing.
	SignatureHelpTriggerKindContentChange SignatureHelpTriggerKind = 3
)

// String returns a string representation of the type.
func (s SignatureHelpTriggerKind) String() string {
	switch s {
	case SignatureHelpTriggerKindInvoked:
		return "Invoked"
	case SignatureHelpTriggerKindTriggerCharacter:
		return "TriggerCharacter"
	case SignatureHelpTriggerKindContentChange:
		return "ContentChange"
	default:
		return strconv.FormatFloat(float64(s), 'f', -10, 64)
	}
}

// SignatureHelpContext is the additional information about the context in which a
// signature help request was triggered.
//
// @since 3.15.0.
type SignatureHelpContext struct {
	// TriggerKind is the action that caused signature help to be triggered.
	TriggerKind SignatureHelpTriggerKind `json:"triggerKind"`

	// Character that caused signature help to be triggered.
	//
	// This is undefined when
	//  TriggerKind != SignatureHelpTriggerKindTriggerCharacter
	TriggerCharacter string `json:"triggerCharacter,omitempty"`

	// IsRetrigger is the `true` if signature help was already showing when it was triggered.
	//
	// Retriggers occur when the signature help is already active and can be
	// caused by actions such as typing a trigger character, a cursor move,
	// or document content changes.
	IsRetrigger bool `json:"isRetrigger"`

	// ActiveSignatureHelp is the currently active SignatureHelp.
	//
	// The `activeSignatureHelp` has its `SignatureHelp.activeSignature` field
	// updated based on the user navigating through available signatures.
	ActiveSignatureHelp *SignatureHelp `json:"activeSignatureHelp,omitempty"`
}

// SignatureHelp signature help represents the signature of something
// callable. There can be multiple signature but only one
// active and only one active parameter.
type SignatureHelp struct {
	// Signatures one or more signatures.
	Signatures []SignatureInformation `json:"signatures"`

	// ActiveParameter is the active parameter of the active signature. If omitted or the value
	// lies outside the range of `signatures[activeSignature].parameters`
	// defaults to 0 if the active signature has parameters. If
	// the active signature has no parameters it is ignored.
	// In future version of the protocol this property might become
	// mandatory to better express the active parameter if the
	// active signature does have any.
	ActiveParameter uint32 `json:"activeParameter,omitempty"`

	// ActiveSignature is the active signature. If omitted or the value lies outside the
	// range of `signatures` the value defaults to zero or is ignored if
	// `signatures.length === 0`. Whenever possible implementors should
	// make an active decision about the active signature and shouldn't
	// rely on a default value.
	// In future version of the protocol this property might become
	// mandatory to better express this.
	ActiveSignature uint32 `json:"activeSignature,omitempty"`
}

// SignatureInformation is the client supports the following `SignatureInformation`
// specific properties.
type SignatureInformation struct {
	// Label is the label of this signature. Will be shown in
	// the UI.
	//
	// @since 3.16.0.
	Label string `json:"label"`

	// Documentation is the human-readable doc-comment of this signature. Will be shown
	// in the UI but can be omitted.
	//
	// @since 3.16.0.
	Documentation interface{} `json:"documentation,omitempty"` // string | *MarkupContent

	// Parameters is the parameters of this signature.
	//
	// @since 3.16.0.
	Parameters []ParameterInformation `json:"parameters,omitempty"`

	// ActiveParameterSupport is the client supports the `activeParameter` property on
	// `SignatureInformation` literal.
	//
	// @since 3.16.0.
	ActiveParameter uint32 `json:"activeParameter,omitempty"`
}

// ParameterInformation represents a parameter of a callable-signature. A parameter can
// have a label and a doc-comment.
type ParameterInformation struct {
	// Label is the label of this parameter information.
	//
	// Either a string or an inclusive start and exclusive end offsets within its containing
	// signature label. (see SignatureInformation.label). The offsets are based on a UTF-16
	// string representation as "Position" and "Range" does.
	//
	// *Note*: a label of type string should be a substring of its containing signature label.
	// Its intended use case is to highlight the parameter label part in the "SignatureInformation.label".
	Label string `json:"label"` // string | [uint32, uint32]

	// Documentation is the human-readable doc-comment of this parameter. Will be shown
	// in the UI but can be omitted.
	Documentation interface{} `json:"documentation,omitempty"` // string | MarkupContent
}

// SignatureHelpRegistrationOptions SignatureHelp Registration options.
type SignatureHelpRegistrationOptions struct {
	TextDocumentRegistrationOptions

	// TriggerCharacters is the characters that trigger signature help
	// automatically.
	TriggerCharacters []string `json:"triggerCharacters,omitempty"`
}

// ReferenceParams params of References request.
//
// @since 3.15.0.
type ReferenceParams struct {
	TextDocumentPositionParams
	WorkDoneProgressParams
	PartialResultParams

	// Context is the ReferenceParams context.
	Context ReferenceContext `json:"context"`
}

// ReferenceContext context of ReferenceParams.
type ReferenceContext struct {
	// IncludeDeclaration include the declaration of the current symbol.
	IncludeDeclaration bool `json:"includeDeclaration"`
}

// DocumentHighlight a document highlight is a range inside a text document which deserves
// special attention. Usually a document highlight is visualized by changing
// the background color of its range.
type DocumentHighlight struct {
	// Range is the range this highlight applies to.
	Range Range `json:"range"`

	// Kind is the highlight kind, default is DocumentHighlightKind.Text.
	Kind DocumentHighlightKind `json:"kind,omitempty"`
}

// DocumentHighlightKind a document highlight kind.
type DocumentHighlightKind float64

const (
	// DocumentHighlightKindText a textual occurrence.
	DocumentHighlightKindText DocumentHighlightKind = 1

	// DocumentHighlightKindRead read-access of a symbol, like reading a variable.
	DocumentHighlightKindRead DocumentHighlightKind = 2

	// DocumentHighlightKindWrite write-access of a symbol, like writing to a variable.
	DocumentHighlightKindWrite DocumentHighlightKind = 3
)

// String implements fmt.Stringer.
func (k DocumentHighlightKind) String() string {
	switch k {
	case DocumentHighlightKindText:
		return "Text"
	case DocumentHighlightKindRead:
		return "Read"
	case DocumentHighlightKindWrite:
		return "Write"
	default:
		return strconv.FormatFloat(float64(k), 'f', -10, 64)
	}
}

// DocumentSymbolParams params of Document Symbols request.
type DocumentSymbolParams struct {
	WorkDoneProgressParams
	PartialResultParams

	// TextDocument is the text document.
	TextDocument TextDocumentIdentifier `json:"textDocument"`
}

// SymbolKind specific capabilities for the `SymbolKind`.
// The symbol kind values the client supports. When this
// property exists the client also guarantees that it will
// handle values outside its set gracefully and falls back
// to a default value when unknown.
//
// If this property is not present the client only supports
// the symbol kinds from `File` to `Array` as defined in
// the initial version of the protocol.
type SymbolKind float64

const (
	// SymbolKindFile symbol of file.
	SymbolKindFile SymbolKind = 1
	// SymbolKindModule symbol of module.
	SymbolKindModule SymbolKind = 2
	// SymbolKindNamespace symbol of namespace.
	SymbolKindNamespace SymbolKind = 3
	// SymbolKindPackage symbol of package.
	SymbolKindPackage SymbolKind = 4
	// SymbolKindClass symbol of class.
	SymbolKindClass SymbolKind = 5
	// SymbolKindMethod symbol of method.
	SymbolKindMethod SymbolKind = 6
	// SymbolKindProperty symbol of property.
	SymbolKindProperty SymbolKind = 7
	// SymbolKindField symbol of field.
	SymbolKindField SymbolKind = 8
	// SymbolKindConstructor symbol of constructor.
	SymbolKindConstructor SymbolKind = 9
	// SymbolKindEnum symbol of enum.
	SymbolKindEnum SymbolKind = 10
	// SymbolKindInterface symbol of interface.
	SymbolKindInterface SymbolKind = 11
	// SymbolKindFunction symbol of function.
	SymbolKindFunction SymbolKind = 12
	// SymbolKindVariable symbol of variable.
	SymbolKindVariable SymbolKind = 13
	// SymbolKindConstant symbol of constant.
	SymbolKindConstant SymbolKind = 14
	// SymbolKindString symbol of string.
	SymbolKindString SymbolKind = 15
	// SymbolKindNumber symbol of number.
	SymbolKindNumber SymbolKind = 16
	// SymbolKindBoolean symbol of boolean.
	SymbolKindBoolean SymbolKind = 17
	// SymbolKindArray symbol of array.
	SymbolKindArray SymbolKind = 18
	// SymbolKindObject symbol of object.
	SymbolKindObject SymbolKind = 19
	// SymbolKindKey symbol of key.
	SymbolKindKey SymbolKind = 20
	// SymbolKindNull symbol of null.
	SymbolKindNull SymbolKind = 21
	// SymbolKindEnumMember symbol of enum member.
	SymbolKindEnumMember SymbolKind = 22
	// SymbolKindStruct symbol of struct.
	SymbolKindStruct SymbolKind = 23
	// SymbolKindEvent symbol of event.
	SymbolKindEvent SymbolKind = 24
	// SymbolKindOperator symbol of operator.
	SymbolKindOperator SymbolKind = 25
	// SymbolKindTypeParameter symbol of type parameter.
	SymbolKindTypeParameter SymbolKind = 26
)

// String implements fmt.Stringer.
//nolint:cyclop
func (k SymbolKind) String() string {
	switch k {
	case SymbolKindFile:
		return "File"
	case SymbolKindModule:
		return "Module"
	case SymbolKindNamespace:
		return "Namespace"
	case SymbolKindPackage:
		return "Package"
	case SymbolKindClass:
		return "Class"
	case SymbolKindMethod:
		return "Method"
	case SymbolKindProperty:
		return "Property"
	case SymbolKindField:
		return "Field"
	case SymbolKindConstructor:
		return "Constructor"
	case SymbolKindEnum:
		return "Enum"
	case SymbolKindInterface:
		return "Interface"
	case SymbolKindFunction:
		return "Function"
	case SymbolKindVariable:
		return "Variable"
	case SymbolKindConstant:
		return "Constant"
	case SymbolKindString:
		return "String"
	case SymbolKindNumber:
		return "Number"
	case SymbolKindBoolean:
		return "Boolean"
	case SymbolKindArray:
		return "Array"
	case SymbolKindObject:
		return "Object"
	case SymbolKindKey:
		return "Key"
	case SymbolKindNull:
		return "Null"
	case SymbolKindEnumMember:
		return "EnumMember"
	case SymbolKindStruct:
		return "Struct"
	case SymbolKindEvent:
		return "Event"
	case SymbolKindOperator:
		return "Operator"
	case SymbolKindTypeParameter:
		return "TypeParameter"
	default:
		return strconv.FormatFloat(float64(k), 'f', -10, 64)
	}
}

// SymbolTag symbol tags are extra annotations that tweak the rendering of a symbol.
//
// @since 3.16.0.
type SymbolTag float64

// list of SymbolTag.
const (
	// SymbolTagDeprecated render a symbol as obsolete, usually using a strike-out.
	SymbolTagDeprecated SymbolTag = 1
)

// String returns a string representation of the SymbolTag.
func (k SymbolTag) String() string {
	switch k {
	case SymbolTagDeprecated:
		return "Deprecated"
	default:
		return strconv.FormatFloat(float64(k), 'f', -10, 64)
	}
}

// DocumentSymbol represents programming constructs like variables, classes, interfaces etc. that appear in a document. Document symbols can be
// hierarchical and they have two ranges: one that encloses its definition and one that points to its most interesting range,
// e.g. the range of an identifier.
type DocumentSymbol struct {
	// Name is the name of this symbol. Will be displayed in the user interface and therefore must not be
	// an empty string or a string only consisting of white spaces.
	Name string `json:"name"`

	// Detail is the more detail for this symbol, e.g the signature of a function.
	Detail string `json:"detail,omitempty"`

	// Kind is the kind of this symbol.
	Kind SymbolKind `json:"kind"`

	// Tags for this document symbol.
	//
	// @since 3.16.0.
	Tags []SymbolTag `json:"tags,omitempty"`

	// Deprecated indicates if this symbol is deprecated.
	Deprecated bool `json:"deprecated,omitempty"`

	// Range is the range enclosing this symbol not including leading/trailing whitespace but everything else
	// like comments. This information is typically used to determine if the clients cursor is
	// inside the symbol to reveal in the symbol in the UI.
	Range Range `json:"range"`

	// SelectionRange is the range that should be selected and revealed when this symbol is being picked, e.g the name of a function.
	// Must be contained by the `range`.
	SelectionRange Range `json:"selectionRange"`

	// Children children of this symbol, e.g. properties of a class.
	Children []DocumentSymbol `json:"children,omitempty"`
}

// SymbolInformation represents information about programming constructs like variables, classes,
// interfaces etc.
type SymbolInformation struct {
	// Name is the name of this symbol.
	Name string `json:"name"`

	// Kind is the kind of this symbol.
	Kind SymbolKind `json:"kind"`

	// Tags for this completion item.
	//
	// @since 3.16.0.
	Tags []SymbolTag `json:"tags,omitempty"`

	// Deprecated indicates if this symbol is deprecated.
	Deprecated bool `json:"deprecated,omitempty"`

	// Location is the location of this symbol. The location's range is used by a tool
	// to reveal the location in the editor. If the symbol is selected in the
	// tool the range's start information is used to position the cursor. So
	// the range usually spans more then the actual symbol's name and does
	// normally include things like visibility modifiers.
	//
	// The range doesn't have to denote a node range in the sense of a abstract
	// syntax tree. It can therefore not be used to re-construct a hierarchy of
	// the symbols.
	Location Location `json:"location"`

	// ContainerName is the name of the symbol containing this symbol. This information is for
	// user interface purposes (e.g. to render a qualifier in the user interface
	// if necessary). It can't be used to re-infer a hierarchy for the document
	// symbols.
	ContainerName string `json:"containerName,omitempty"`
}

// CodeActionParams params for the CodeActionRequest.
type CodeActionParams struct {
	WorkDoneProgressParams
	PartialResultParams

	// TextDocument is the document in which the command was invoked.
	TextDocument TextDocumentIdentifier `json:"textDocument"`

	// Context carrying additional information.
	Context CodeActionContext `json:"context"`

	// Range is the range for which the command was invoked.
	Range Range `json:"range"`
}

// CodeActionKind is the code action kind values the client supports. When this
// property exists the client also guarantees that it will
// handle values outside its set gracefully and falls back
// to a default value when unknown.
type CodeActionKind string

// A set of predefined code action kinds.
const (
	// QuickFix base kind for quickfix actions: 'quickfix'.
	QuickFix CodeActionKind = "quickfix"

	// Refactor base kind for refactoring actions: 'refactor'.
	Refactor CodeActionKind = "refactor"

	// RefactorExtract base kind for refactoring extraction actions: 'refactor.extract'
	//
	// Example extract actions:
	//
	// - Extract method
	// - Extract function
	// - Extract variable
	// - Extract interface from class
	// - ...
	RefactorExtract CodeActionKind = "refactor.extract"

	// RefactorInline base kind for refactoring inline actions: 'refactor.inline'
	//
	// Example inline actions:
	//
	// - Inline function
	// - Inline variable
	// - Inline constant
	// - ...
	RefactorInline CodeActionKind = "refactor.inline"

	// RefactorRewrite base kind for refactoring rewrite actions: 'refactor.rewrite'
	//
	// Example rewrite actions:
	//
	// - Convert JavaScript function to class
	// - Add or remove parameter
	// - Encapsulate field
	// - Make method static
	// - Move method to base class
	// - ...
	RefactorRewrite CodeActionKind = "refactor.rewrite"

	// Source base kind for source actions: `source`
	//
	// Source code actions apply to the entire file.
	Source CodeActionKind = "source"

	// SourceOrganizeImports base kind for an organize imports source action: `source.organizeImports`.
	SourceOrganizeImports CodeActionKind = "source.organizeImports"
)

// CodeActionContext contains additional diagnostic information about the context in which
// a code action is run.
type CodeActionContext struct {
	// Diagnostics is an array of diagnostics.
	Diagnostics []Diagnostic `json:"diagnostics"`

	// Only requested kind of actions to return.
	//
	// Actions not of this kind are filtered out by the client before being shown. So servers
	// can omit computing them.
	Only []CodeActionKind `json:"only,omitempty"`
}

// CodeAction capabilities specific to the `textDocument/codeAction`.
type CodeAction struct {
	// Title is a short, human-readable, title for this code action.
	Title string `json:"title"`

	// Kind is the kind of the code action.
	//
	// Used to filter code actions.
	Kind CodeActionKind `json:"kind,omitempty"`

	// Diagnostics is the diagnostics that this code action resolves.
	Diagnostics []Diagnostic `json:"diagnostics,omitempty"`

	// IsPreferred marks this as a preferred action. Preferred actions are used by the `auto fix` command and can be targeted
	// by keybindings.
	//
	// A quick fix should be marked preferred if it properly addresses the underlying error.
	// A refactoring should be marked preferred if it is the most reasonable choice of actions to take.
	//
	// @since 3.15.0.
	IsPreferred bool `json:"isPreferred,omitempty"`

	// Disabled marks that the code action cannot currently be applied.
	//
	// Clients should follow the following guidelines regarding disabled code
	// actions:
	//
	//  - Disabled code actions are not shown in automatic lightbulbs code
	//    action menus.
	//
	//  - Disabled actions are shown as faded out in the code action menu when
	//    the user request a more specific type of code action, such as
	//    refactorings.
	//
	//  - If the user has a keybinding that auto applies a code action and only
	//    a disabled code actions are returned, the client should show the user
	//    an error message with `reason` in the editor.
	//
	// @since 3.16.0.
	Disabled *CodeActionDisable `json:"disabled,omitempty"`

	// Edit is the workspace edit this code action performs.
	Edit *WorkspaceEdit `json:"edit,omitempty"`

	// Command is a command this code action executes. If a code action
	// provides an edit and a command, first the edit is
	// executed and then the command.
	Command *Command `json:"command,omitempty"`

	// Data is a data entry field that is preserved on a code action between
	// a "textDocument/codeAction" and a "codeAction/resolve" request.
	//
	// @since 3.16.0.
	Data interface{} `json:"data,omitempty"`
}

// CodeActionDisable Disable in CodeAction.
//
// @since 3.16.0.
type CodeActionDisable struct {
	// Reason human readable description of why the code action is currently
	// disabled.
	//
	// This is displayed in the code actions UI.
	Reason string `json:"reason"`
}

// CodeActionRegistrationOptions CodeAction Registrationi options.
type CodeActionRegistrationOptions struct {
	TextDocumentRegistrationOptions

	CodeActionOptions
}

// CodeLensParams params of Code Lens request.
type CodeLensParams struct {
	WorkDoneProgressParams
	PartialResultParams

	// TextDocument is the document to request code lens for.
	TextDocument TextDocumentIdentifier `json:"textDocument"`
}

// CodeLens is a code lens represents a command that should be shown along with
// source text, like the number of references, a way to run tests, etc.
//
// A code lens is _unresolved_ when no command is associated to it. For performance
// reasons the creation of a code lens and resolving should be done in two stages.
type CodeLens struct {
	// Range is the range in which this code lens is valid. Should only span a single line.
	Range Range `json:"range"`

	// Command is the command this code lens represents.
	Command *Command `json:"command,omitempty"`

	// Data is a data entry field that is preserved on a code lens item between
	// a code lens and a code lens resolve request.
	Data interface{} `json:"data,omitempty"`
}

// CodeLensRegistrationOptions CodeLens Registration options.
type CodeLensRegistrationOptions struct {
	TextDocumentRegistrationOptions

	// ResolveProvider code lens has a resolve provider as well.
	ResolveProvider bool `json:"resolveProvider,omitempty"`
}

// DocumentLinkParams params of Document Link request.
type DocumentLinkParams struct {
	WorkDoneProgressParams
	PartialResultParams

	// TextDocument is the document to provide document links for.
	TextDocument TextDocumentIdentifier `json:"textDocument"`
}

// DocumentLink is a document link is a range in a text document that links to an internal or external resource, like another
// text document or a web site.
type DocumentLink struct {
	// Range is the range this link applies to.
	Range Range `json:"range"`

	// Target is the uri this link points to. If missing a resolve request is sent later.
	Target DocumentURI `json:"target,omitempty"`

	// Tooltip is the tooltip text when you hover over this link.
	//
	// If a tooltip is provided, is will be displayed in a string that includes instructions on how to
	// trigger the link, such as `{0} (ctrl + click)`. The specific instructions vary depending on OS,
	// user settings, and localization.
	//
	// @since 3.15.0.
	Tooltip string `json:"tooltip,omitempty"`

	// Data is a data entry field that is preserved on a document link between a
	// DocumentLinkRequest and a DocumentLinkResolveRequest.
	Data interface{} `json:"data,omitempty"`
}

// DocumentColorParams params of Document Color request.
type DocumentColorParams struct {
	WorkDoneProgressParams
	PartialResultParams

	// TextDocument is the document to format.
	TextDocument TextDocumentIdentifier `json:"textDocument"`
}

// ColorInformation response of Document Color request.
type ColorInformation struct {
	// Range is the range in the document where this color appears.
	Range Range `json:"range"`

	// Color is the actual color value for this color range.
	Color Color `json:"color"`
}

// Color represents a color in RGBA space.
type Color struct {
	// Alpha is the alpha component of this color in the range [0-1].
	Alpha float64 `json:"alpha"`

	// Blue is the blue component of this color in the range [0-1].
	Blue float64 `json:"blue"`

	// Green is the green component of this color in the range [0-1].
	Green float64 `json:"green"`

	// Red is the red component of this color in the range [0-1].
	Red float64 `json:"red"`
}

// ColorPresentationParams params of Color Presentation request.
type ColorPresentationParams struct {
	WorkDoneProgressParams
	PartialResultParams

	// TextDocument is the text document.
	TextDocument TextDocumentIdentifier `json:"textDocument"`

	// Color is the color information to request presentations for.
	Color Color `json:"color"`

	// Range is the range where the color would be inserted. Serves as a context.
	Range Range `json:"range"`
}

// ColorPresentation response of Color Presentation request.
type ColorPresentation struct {
	// Label is the label of this color presentation. It will be shown on the color
	// picker header. By default this is also the text that is inserted when selecting
	// this color presentation.
	Label string `json:"label"`

	// TextEdit an edit which is applied to a document when selecting
	// this presentation for the color.  When `falsy` the label is used.
	TextEdit *TextEdit `json:"textEdit,omitempty"`

	// AdditionalTextEdits an optional array of additional [text edits](#TextEdit) that are applied when
	// selecting this color presentation. Edits must not overlap with the main [edit](#ColorPresentation.textEdit) nor with themselves.
	AdditionalTextEdits []TextEdit `json:"additionalTextEdits,omitempty"`
}

// DocumentFormattingParams params of Document Formatting request.
type DocumentFormattingParams struct {
	WorkDoneProgressParams

	// Options is the format options.
	Options FormattingOptions `json:"options"`

	// TextDocument is the document to format.
	TextDocument TextDocumentIdentifier `json:"textDocument"`
}

// FormattingOptions value-object describing what options formatting should use.
type FormattingOptions struct {
	// InsertSpaces prefer spaces over tabs.
	InsertSpaces bool `json:"insertSpaces"`

	// TabSize size of a tab in spaces.
	TabSize uint32 `json:"tabSize"`

	// TrimTrailingWhitespace trim trailing whitespaces on a line.
	//
	// @since 3.15.0.
	TrimTrailingWhitespace bool `json:"trimTrailingWhitespace,omitempty"`

	// InsertFinalNewlines insert a newline character at the end of the file if one does not exist.
	//
	// @since 3.15.0.
	InsertFinalNewline bool `json:"insertFinalNewline,omitempty"`

	// TrimFinalNewlines trim all newlines after the final newline at the end of the file.
	//
	// @since 3.15.0.
	TrimFinalNewlines bool `json:"trimFinalNewlines,omitempty"`

	// Key is the signature for further properties.
	Key map[string]interface{} `json:"key,omitempty"` // bool | int32 | string
}

// DocumentRangeFormattingParams params of Document Range Formatting request.
type DocumentRangeFormattingParams struct {
	WorkDoneProgressParams

	// TextDocument is the document to format.
	TextDocument TextDocumentIdentifier `json:"textDocument"`

	// Range is the range to format
	Range Range `json:"range"`

	// Options is the format options.
	Options FormattingOptions `json:"options"`
}

// DocumentOnTypeFormattingParams params of Document on Type Formatting request.
type DocumentOnTypeFormattingParams struct {
	// TextDocument is the document to format.
	TextDocument TextDocumentIdentifier `json:"textDocument"`

	// Position is the position at which this request was sent.
	Position Position `json:"position"`

	// Ch is the character that has been typed.
	Ch string `json:"ch"`

	// Options is the format options.
	Options FormattingOptions `json:"options"`
}

// DocumentOnTypeFormattingRegistrationOptions DocumentOnTypeFormatting Registration options.
type DocumentOnTypeFormattingRegistrationOptions struct {
	TextDocumentRegistrationOptions

	// FirstTriggerCharacter a character on which formatting should be triggered, like `}`.
	FirstTriggerCharacter string `json:"firstTriggerCharacter"`

	// MoreTriggerCharacter a More trigger characters.
	MoreTriggerCharacter []string `json:"moreTriggerCharacter"`
}

// RenameParams params of Rename request.
type RenameParams struct {
	TextDocumentPositionParams
	PartialResultParams

	// NewName is the new name of the symbol. If the given name is not valid the
	// request must return a [ResponseError](#ResponseError) with an
	// appropriate message set.
	NewName string `json:"newName"`
}

// RenameRegistrationOptions Rename Registration options.
type RenameRegistrationOptions struct {
	TextDocumentRegistrationOptions

	// PrepareProvider is the renames should be checked and tested for validity before being executed.
	PrepareProvider bool `json:"prepareProvider,omitempty"`
}

// PrepareRenameParams params of PrepareRenameParams request.
//
// @since 3.15.0.
type PrepareRenameParams struct {
	TextDocumentPositionParams
}

// FoldingRangeParams params of Folding Range request.
type FoldingRangeParams struct {
	TextDocumentPositionParams
	PartialResultParams
}

// FoldingRangeKind is the enum of known range kinds.
type FoldingRangeKind string

const (
	// CommentFoldingRange is the folding range for a comment.
	CommentFoldingRange FoldingRangeKind = "comment"

	// ImportsFoldingRange is the folding range for a imports or includes.
	ImportsFoldingRange FoldingRangeKind = "imports"

	// RegionFoldingRange is the folding range for a region (e.g. `#region`).
	RegionFoldingRange FoldingRangeKind = "region"
)

// FoldingRange capabilities specific to `textDocument/foldingRange` requests.
//
// @since 3.10.0.
type FoldingRange struct {
	// StartLine is the zero-based line number from where the folded range starts.
	StartLine uint32 `json:"startLine"`

	// StartCharacter is the zero-based character offset from where the folded range starts. If not defined, defaults to the length of the start line.
	StartCharacter uint32 `json:"startCharacter,omitempty"`

	// EndLine is the zero-based line number where the folded range ends.
	EndLine uint32 `json:"endLine"`

	// EndCharacter is the zero-based character offset before the folded range ends. If not defined, defaults to the length of the end line.
	EndCharacter uint32 `json:"endCharacter,omitempty"`

	// Kind describes the kind of the folding range such as `comment' or 'region'. The kind
	// is used to categorize folding ranges and used by commands like 'Fold all comments'.
	// See FoldingRangeKind for an enumeration of standardized kinds.
	Kind FoldingRangeKind `json:"kind,omitempty"`
}
