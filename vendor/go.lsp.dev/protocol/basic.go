// SPDX-FileCopyrightText: 2019 The Go Language Server Authors
// SPDX-License-Identifier: BSD-3-Clause

package protocol

import (
	"go.lsp.dev/uri"
)

// DocumentURI represents the URI of a document.
//
// Many of the interfaces contain fields that correspond to the URI of a document.
// For clarity, the type of such a field is declared as a DocumentURI.
// Over the wire, it will still be transferred as a string, but this guarantees
// that the contents of that string can be parsed as a valid URI.
type DocumentURI = uri.URI

// URI a tagging interface for normal non document URIs.
//
// @since 3.16.0.
type URI = uri.URI

// EOL denotes represents the character offset.
var EOL = []string{"\n", "\r\n", "\r"}

// Position represents a text document expressed as zero-based line and zero-based character offset.
//
// The offsets are based on a UTF-16 string representation.
// So a string of the form "a𐐀b" the character offset of the character "a" is 0,
// the character offset of "𐐀" is 1 and the character offset of "b" is 3 since 𐐀 is represented using two code
// units in UTF-16.
//
// Positions are line end character agnostic. So you can not specify a position that
// denotes "\r|\n" or "\n|" where "|" represents the character offset.
//
// Position is between two characters like an "insert" cursor in a editor.
// Special values like for example "-1" to denote the end of a line are not supported.
type Position struct {
	// Line position in a document (zero-based).
	//
	// If a line number is greater than the number of lines in a document, it defaults back to the number of lines in
	// the document.
	// If a line number is negative, it defaults to 0.
	Line uint32 `json:"line"`

	// Character offset on a line in a document (zero-based).
	//
	// Assuming that the line is represented as a string, the Character value represents the gap between the
	// "character" and "character + 1".
	//
	// If the character value is greater than the line length it defaults back to the line length.
	// If a line number is negative, it defaults to 0.
	Character uint32 `json:"character"`
}

// Range represents a text document expressed as (zero-based) start and end positions.
//
// A range is comparable to a selection in an editor. Therefore the end position is exclusive.
// If you want to specify a range that contains a line including the line ending character(s) then use an end position
// denoting the start of the next line.
type Range struct {
	// Start is the range's start position.
	Start Position `json:"start"`

	// End is the range's end position.
	End Position `json:"end"`
}

// Location represents a location inside a resource, such as a line inside a text file.
type Location struct {
	URI   DocumentURI `json:"uri"`
	Range Range       `json:"range"`
}

// LocationLink represents a link between a source and a target location.
type LocationLink struct {
	// OriginSelectionRange span of the origin of this link.
	//
	// Used as the underlined span for mouse interaction. Defaults to the word range at the mouse position.
	OriginSelectionRange *Range `json:"originSelectionRange,omitempty"`

	// TargetURI is the target resource identifier of this link.
	TargetURI DocumentURI `json:"targetUri"`

	// TargetRange is the full target range of this link.
	//
	// If the target for example is a symbol then target range is the range enclosing this symbol not including
	// leading/trailing whitespace but everything else like comments.
	//
	// This information is typically used to highlight the range in the editor.
	TargetRange Range `json:"targetRange"`

	// TargetSelectionRange is the range that should be selected and revealed when this link is being followed,
	// e.g the name of a function.
	//
	// Must be contained by the the TargetRange. See also DocumentSymbol#range
	TargetSelectionRange Range `json:"targetSelectionRange"`
}

// Command represents a reference to a command. Provides a title which will be used to represent a command in the UI.
//
// Commands are identified by a string identifier.
// The recommended way to handle commands is to implement their execution on the server side if the client and
// server provides the corresponding capabilities.
//
// Alternatively the tool extension code could handle the command. The protocol currently doesn't specify
// a set of well-known commands.
type Command struct {
	// Title of the command, like `save`.
	Title string `json:"title"`

	// Command is the identifier of the actual command handler.
	Command string `json:"command"`

	// Arguments that the command handler should be invoked with.
	Arguments []interface{} `json:"arguments,omitempty"`
}

// TextEdit is a textual edit applicable to a text document.
type TextEdit struct {
	// Range is the range of the text document to be manipulated.
	//
	// To insert text into a document create a range where start == end.
	Range Range `json:"range"`

	// NewText is the string to be inserted. For delete operations use an
	// empty string.
	NewText string `json:"newText"`
}

// ChangeAnnotation is the additional information that describes document changes.
//
// @since 3.16.0.
type ChangeAnnotation struct {
	// Label a human-readable string describing the actual change.
	// The string is rendered prominent in the user interface.
	Label string `json:"label"`

	// NeedsConfirmation is a flag which indicates that user confirmation is needed
	// before applying the change.
	NeedsConfirmation bool `json:"needsConfirmation,omitempty"`

	// Description is a human-readable string which is rendered less prominent in
	// the user interface.
	Description string `json:"description,omitempty"`
}

// ChangeAnnotationIdentifier an identifier referring to a change annotation managed by a workspace
// edit.
//
// @since 3.16.0.
type ChangeAnnotationIdentifier string

// AnnotatedTextEdit is a special text edit with an additional change annotation.
//
// @since 3.16.0.
type AnnotatedTextEdit struct {
	TextEdit

	// AnnotationID is the actual annotation identifier.
	AnnotationID ChangeAnnotationIdentifier `json:"annotationId"`
}

// TextDocumentEdit describes textual changes on a single text document.
//
// The TextDocument is referred to as a OptionalVersionedTextDocumentIdentifier to allow clients to check the
// text document version before an edit is applied.
//
// TextDocumentEdit describes all changes on a version "Si" and after they are applied move the document to
// version "Si+1".
// So the creator of a TextDocumentEdit doesn't need to sort the array or do any kind of ordering. However the
// edits must be non overlapping.
type TextDocumentEdit struct {
	// TextDocument is the text document to change.
	TextDocument OptionalVersionedTextDocumentIdentifier `json:"textDocument"`

	// Edits is the edits to be applied.
	//
	// @since 3.16.0 - support for AnnotatedTextEdit.
	// This is guarded by the client capability Workspace.WorkspaceEdit.ChangeAnnotationSupport.
	Edits []TextEdit `json:"edits"` // []TextEdit | []AnnotatedTextEdit
}

// ResourceOperationKind is the file event type.
type ResourceOperationKind string

const (
	// CreateResourceOperation supports creating new files and folders.
	CreateResourceOperation ResourceOperationKind = "create"

	// RenameResourceOperation supports renaming existing files and folders.
	RenameResourceOperation ResourceOperationKind = "rename"

	// DeleteResourceOperation supports deleting existing files and folders.
	DeleteResourceOperation ResourceOperationKind = "delete"
)

// CreateFileOptions represents an options to create a file.
type CreateFileOptions struct {
	// Overwrite existing file. Overwrite wins over `ignoreIfExists`.
	Overwrite bool `json:"overwrite,omitempty"`

	// IgnoreIfExists ignore if exists.
	IgnoreIfExists bool `json:"ignoreIfExists,omitempty"`
}

// CreateFile represents a create file operation.
type CreateFile struct {
	// Kind a create.
	Kind ResourceOperationKind `json:"kind"` // should be `create`

	// URI is the resource to create.
	URI DocumentURI `json:"uri"`

	// Options additional options.
	Options *CreateFileOptions `json:"options,omitempty"`

	// AnnotationID an optional annotation identifier describing the operation.
	//
	// @since 3.16.0.
	AnnotationID ChangeAnnotationIdentifier `json:"annotationId,omitempty"`
}

// RenameFileOptions represents a rename file options.
type RenameFileOptions struct {
	// Overwrite target if existing. Overwrite wins over `ignoreIfExists`.
	Overwrite bool `json:"overwrite,omitempty"`

	// IgnoreIfExists ignores if target exists.
	IgnoreIfExists bool `json:"ignoreIfExists,omitempty"`
}

// RenameFile represents a rename file operation.
type RenameFile struct {
	// Kind a rename.
	Kind ResourceOperationKind `json:"kind"` // should be `rename`

	// OldURI is the old (existing) location.
	OldURI DocumentURI `json:"oldUri"`

	// NewURI is the new location.
	NewURI DocumentURI `json:"newUri"`

	// Options rename options.
	Options *RenameFileOptions `json:"options,omitempty"`

	// AnnotationID an optional annotation identifier describing the operation.
	//
	// @since 3.16.0.
	AnnotationID ChangeAnnotationIdentifier `json:"annotationId,omitempty"`
}

// DeleteFileOptions represents a delete file options.
type DeleteFileOptions struct {
	// Recursive delete the content recursively if a folder is denoted.
	Recursive bool `json:"recursive,omitempty"`

	// IgnoreIfNotExists ignore the operation if the file doesn't exist.
	IgnoreIfNotExists bool `json:"ignoreIfNotExists,omitempty"`
}

// DeleteFile represents a delete file operation.
type DeleteFile struct {
	// Kind is a delete.
	Kind ResourceOperationKind `json:"kind"` // should be `delete`

	// URI is the file to delete.
	URI DocumentURI `json:"uri"`

	// Options delete options.
	Options *DeleteFileOptions `json:"options,omitempty"`

	// AnnotationID an optional annotation identifier describing the operation.
	//
	// @since 3.16.0.
	AnnotationID ChangeAnnotationIdentifier `json:"annotationId,omitempty"`
}

// WorkspaceEdit represent a changes to many resources managed in the workspace.
//
// The edit should either provide changes or documentChanges.
// If the client can handle versioned document edits and if documentChanges are present, the latter are preferred over
// changes.
type WorkspaceEdit struct {
	// Changes holds changes to existing resources.
	Changes map[DocumentURI][]TextEdit `json:"changes,omitempty"`

	// DocumentChanges depending on the client capability `workspace.workspaceEdit.resourceOperations` document changes
	// are either an array of `TextDocumentEdit`s to express changes to n different text documents
	// where each text document edit addresses a specific version of a text document. Or it can contain
	// above `TextDocumentEdit`s mixed with create, rename and delete file / folder operations.
	//
	// Whether a client supports versioned document edits is expressed via
	// `workspace.workspaceEdit.documentChanges` client capability.
	//
	// If a client neither supports `documentChanges` nor `workspace.workspaceEdit.resourceOperations` then
	// only plain `TextEdit`s using the `changes` property are supported.
	DocumentChanges []TextDocumentEdit `json:"documentChanges,omitempty"`

	// ChangeAnnotations is a map of change annotations that can be referenced in
	// "AnnotatedTextEdit"s or create, rename and delete file / folder
	// operations.
	//
	// Whether clients honor this property depends on the client capability
	// "workspace.changeAnnotationSupport".
	//
	// @since 3.16.0.
	ChangeAnnotations map[ChangeAnnotationIdentifier]ChangeAnnotation `json:"changeAnnotations,omitempty"`
}

// TextDocumentIdentifier indicates the using a URI. On the protocol level, URIs are passed as strings.
type TextDocumentIdentifier struct {
	// URI is the text document's URI.
	URI DocumentURI `json:"uri"`
}

// TextDocumentItem represent an item to transfer a text document from the client to the server.
type TextDocumentItem struct {
	// URI is the text document's URI.
	URI DocumentURI `json:"uri"`

	// LanguageID is the text document's language identifier.
	LanguageID LanguageIdentifier `json:"languageId"`

	// Version is the version number of this document (it will increase after each
	// change, including undo/redo).
	Version int32 `json:"version"`

	// Text is the content of the opened text document.
	Text string `json:"text"`
}

// LanguageIdentifier represent a text document's language identifier.
type LanguageIdentifier string

const (
	// ABAPLanguage ABAP Language.
	ABAPLanguage LanguageIdentifier = "abap"

	// BatLanguage Windows Bat Language.
	BatLanguage LanguageIdentifier = "bat"

	// BibtexLanguage BibTeX Language.
	BibtexLanguage LanguageIdentifier = "bibtex"

	// ClojureLanguage Clojure Language.
	ClojureLanguage LanguageIdentifier = "clojure"

	// CoffeescriptLanguage CoffeeScript Language.
	CoffeeScriptLanguage LanguageIdentifier = "coffeescript"

	// CLanguage C Language.
	CLanguage LanguageIdentifier = "c"

	// CppLanguage C++ Language.
	CppLanguage LanguageIdentifier = "cpp"

	// CsharpLanguage C# Language.
	CsharpLanguage LanguageIdentifier = "csharp"

	// CSSLanguage CSS Language.
	CSSLanguage LanguageIdentifier = "css"

	// DiffLanguage Diff Language.
	DiffLanguage LanguageIdentifier = "diff"

	// DartLanguage Dart Language.
	DartLanguage LanguageIdentifier = "dart"

	// DockerfileLanguage Dockerfile Language.
	DockerfileLanguage LanguageIdentifier = "dockerfile"

	// ElixirLanguage Elixir Language.
	ElixirLanguage LanguageIdentifier = "elixir"

	// ErlangLanguage Erlang Language.
	ErlangLanguage LanguageIdentifier = "erlang"

	// FsharpLanguage F# Language.
	FsharpLanguage LanguageIdentifier = "fsharp"

	// GitCommitLanguage Git Language.
	GitCommitLanguage LanguageIdentifier = "git-commit"

	// GitRebaseLanguage Git Language.
	GitRebaseLanguage LanguageIdentifier = "git-rebase"

	// GoLanguage Go Language.
	GoLanguage LanguageIdentifier = "go"

	// GroovyLanguage Groovy Language.
	GroovyLanguage LanguageIdentifier = "groovy"

	// HandlebarsLanguage Handlebars Language.
	HandlebarsLanguage LanguageIdentifier = "handlebars"

	// HTMLLanguage HTML Language.
	HTMLLanguage LanguageIdentifier = "html"

	// IniLanguage Ini Language.
	IniLanguage LanguageIdentifier = "ini"

	// JavaLanguage Java Language.
	JavaLanguage LanguageIdentifier = "java"

	// JavaScriptLanguage JavaScript Language.
	JavaScriptLanguage LanguageIdentifier = "javascript"

	// JavaScriptReactLanguage JavaScript React Language.
	JavaScriptReactLanguage LanguageIdentifier = "javascriptreact"

	// JSONLanguage JSON Language.
	JSONLanguage LanguageIdentifier = "json"

	// LatexLanguage LaTeX Language.
	LatexLanguage LanguageIdentifier = "latex"

	// LessLanguage Less Language.
	LessLanguage LanguageIdentifier = "less"

	// LuaLanguage Lua Language.
	LuaLanguage LanguageIdentifier = "lua"

	// MakefileLanguage Makefile Language.
	MakefileLanguage LanguageIdentifier = "makefile"

	// MarkdownLanguage Markdown Language.
	MarkdownLanguage LanguageIdentifier = "markdown"

	// ObjectiveCLanguage Objective-C Language.
	ObjectiveCLanguage LanguageIdentifier = "objective-c"

	// ObjectiveCppLanguage Objective-C++ Language.
	ObjectiveCppLanguage LanguageIdentifier = "objective-cpp"

	// PerlLanguage Perl Language.
	PerlLanguage LanguageIdentifier = "perl"

	// Perl6Language Perl Language.
	Perl6Language LanguageIdentifier = "perl6"

	// PHPLanguage PHP Language.
	PHPLanguage LanguageIdentifier = "php"

	// PowershellLanguage Powershell Language.
	PowershellLanguage LanguageIdentifier = "powershell"

	// JadeLanguage Pug Language.
	JadeLanguage LanguageIdentifier = "jade"

	// PythonLanguage Python Language.
	PythonLanguage LanguageIdentifier = "python"

	// RLanguage R Language.
	RLanguage LanguageIdentifier = "r"

	// RazorLanguage Razor(cshtml) Language.
	RazorLanguage LanguageIdentifier = "razor"

	// RubyLanguage Ruby Language.
	RubyLanguage LanguageIdentifier = "ruby"

	// RustLanguage Rust Language.
	RustLanguage LanguageIdentifier = "rust"

	// SCSSLanguage SCSS Languages syntax using curly brackets.
	SCSSLanguage LanguageIdentifier = "scss"

	// SASSLanguage SCSS Languages indented syntax.
	SASSLanguage LanguageIdentifier = "sass"

	// ScalaLanguage Scala Language.
	ScalaLanguage LanguageIdentifier = "scala"

	// ShaderlabLanguage ShaderLab Language.
	ShaderlabLanguage LanguageIdentifier = "shaderlab"

	// ShellscriptLanguage Shell Script (Bash) Language.
	ShellscriptLanguage LanguageIdentifier = "shellscript"

	// SQLLanguage SQL Language.
	SQLLanguage LanguageIdentifier = "sql"

	// SwiftLanguage Swift Language.
	SwiftLanguage LanguageIdentifier = "swift"

	// TypeScriptLanguage TypeScript Language.
	TypeScriptLanguage LanguageIdentifier = "typescript"

	// TypeScriptReactLanguage TypeScript React Language.
	TypeScriptReactLanguage LanguageIdentifier = "typescriptreact"

	// TeXLanguage TeX Language.
	TeXLanguage LanguageIdentifier = "tex"

	// VBLanguage Visual Basic Language.
	VBLanguage LanguageIdentifier = "vb"

	// XMLLanguage XML Language.
	XMLLanguage LanguageIdentifier = "xml"

	// XslLanguage XSL Language.
	XslLanguage LanguageIdentifier = "xsl"

	// YamlLanguage YAML Language.
	YamlLanguage LanguageIdentifier = "yaml"
)

// languageIdentifierMap map of LanguageIdentifiers.
var languageIdentifierMap = map[string]LanguageIdentifier{
	"abap":            ABAPLanguage,
	"bat":             BatLanguage,
	"bibtex":          BibtexLanguage,
	"clojure":         ClojureLanguage,
	"coffeescript":    CoffeeScriptLanguage,
	"c":               CLanguage,
	"cpp":             CppLanguage,
	"csharp":          CsharpLanguage,
	"css":             CSSLanguage,
	"diff":            DiffLanguage,
	"dart":            DartLanguage,
	"dockerfile":      DockerfileLanguage,
	"elixir":          ElixirLanguage,
	"erlang":          ErlangLanguage,
	"fsharp":          FsharpLanguage,
	"git-commit":      GitCommitLanguage,
	"git-rebase":      GitRebaseLanguage,
	"go":              GoLanguage,
	"groovy":          GroovyLanguage,
	"handlebars":      HandlebarsLanguage,
	"html":            HTMLLanguage,
	"ini":             IniLanguage,
	"java":            JavaLanguage,
	"javascript":      JavaScriptLanguage,
	"javascriptreact": JavaScriptReactLanguage,
	"json":            JSONLanguage,
	"latex":           LatexLanguage,
	"less":            LessLanguage,
	"lua":             LuaLanguage,
	"makefile":        MakefileLanguage,
	"markdown":        MarkdownLanguage,
	"objective-c":     ObjectiveCLanguage,
	"objective-cpp":   ObjectiveCppLanguage,
	"perl":            PerlLanguage,
	"perl6":           Perl6Language,
	"php":             PHPLanguage,
	"powershell":      PowershellLanguage,
	"jade":            JadeLanguage,
	"python":          PythonLanguage,
	"r":               RLanguage,
	"razor":           RazorLanguage,
	"ruby":            RubyLanguage,
	"rust":            RustLanguage,
	"scss":            SCSSLanguage,
	"sass":            SASSLanguage,
	"scala":           ScalaLanguage,
	"shaderlab":       ShaderlabLanguage,
	"shellscript":     ShellscriptLanguage,
	"sql":             SQLLanguage,
	"swift":           SwiftLanguage,
	"typescript":      TypeScriptLanguage,
	"typescriptreact": TypeScriptReactLanguage,
	"tex":             TeXLanguage,
	"vb":              VBLanguage,
	"xml":             XMLLanguage,
	"xsl":             XslLanguage,
	"yaml":            YamlLanguage,
}

// ToLanguageIdentifier converts ft to LanguageIdentifier.
func ToLanguageIdentifier(ft string) LanguageIdentifier {
	langID, ok := languageIdentifierMap[ft]
	if ok {
		return langID
	}

	return LanguageIdentifier(ft)
}

// VersionedTextDocumentIdentifier represents an identifier to denote a specific version of a text document.
//
// This information usually flows from the client to the server.
type VersionedTextDocumentIdentifier struct {
	TextDocumentIdentifier

	// Version is the version number of this document.
	//
	// The version number of a document will increase after each change, including
	// undo/redo. The number doesn't need to be consecutive.
	Version int32 `json:"version"`
}

// OptionalVersionedTextDocumentIdentifier represents an identifier which optionally denotes a specific version of
// a text document.
//
// This information usually flows from the server to the client.
//
// @since 3.16.0.
type OptionalVersionedTextDocumentIdentifier struct {
	TextDocumentIdentifier

	// Version is the version number of this document. If an optional versioned text document
	// identifier is sent from the server to the client and the file is not
	// open in the editor (the server has not received an open notification
	// before) the server can send `null` to indicate that the version is
	// known and the content on disk is the master (as specified with document
	// content ownership).
	//
	// The version number of a document will increase after each change,
	// including undo/redo. The number doesn't need to be consecutive.
	Version *int32 `json:"version"` // int32 | null
}

// TextDocumentPositionParams is a parameter literal used in requests to pass a text document and a position
// inside that document.
//
// It is up to the client to decide how a selection is converted into a position when issuing a request for a text
// document.
//
// The client can for example honor or ignore the selection direction to make LSP request consistent with features
// implemented internally.
type TextDocumentPositionParams struct {
	// TextDocument is the text document.
	TextDocument TextDocumentIdentifier `json:"textDocument"`

	// Position is the position inside the text document.
	Position Position `json:"position"`
}

// DocumentFilter is a document filter denotes a document through properties like language, scheme or pattern.
//
// An example is a filter that applies to TypeScript files on disk.
type DocumentFilter struct {
	// Language a language id, like `typescript`.
	Language string `json:"language,omitempty"`

	// Scheme a URI scheme, like `file` or `untitled`.
	Scheme string `json:"scheme,omitempty"`

	// Pattern a glob pattern, like `*.{ts,js}`.
	//
	// Glob patterns can have the following syntax:
	//  "*"
	// "*" to match one or more characters in a path segment
	//  "?"
	// "?" to match on one character in a path segment
	//  "**"
	// "**" to match any number of path segments, including none
	//  "{}"
	// "{}" to group conditions (e.g. `**/*.{ts,js}` matches all TypeScript and JavaScript files)
	//  "[]"
	// "[]" to declare a range of characters to match in a path segment (e.g., `example.[0-9]` to match on `example.0`, `example.1`, …)
	//  "[!...]"
	// "[!...]" to negate a range of characters to match in a path segment (e.g., `example.[!0-9]` to match on `example.a`, `example.b`, but not `example.0`)
	Pattern string `json:"pattern,omitempty"`
}

// DocumentSelector is a document selector is the combination of one or more document filters.
type DocumentSelector []*DocumentFilter

// MarkupKind describes the content type that a client supports in various
// result literals like `Hover`, `ParameterInfo` or `CompletionItem`.
//
// Please note that `MarkupKinds` must not start with a `$`. This kinds
// are reserved for internal usage.
type MarkupKind string

const (
	// PlainText is supported as a content format.
	PlainText MarkupKind = "plaintext"

	// Markdown is supported as a content format.
	Markdown MarkupKind = "markdown"
)

// MarkupContent a `MarkupContent` literal represents a string value which content is interpreted base on its
// kind flag.
//
// Currently the protocol supports `plaintext` and `markdown` as markup kinds.
//
// If the kind is `markdown` then the value can contain fenced code blocks like in GitHub issues.
// See https://help.github.com/articles/creating-and-highlighting-code-blocks/#syntax-highlighting
//
// Here is an example how such a string can be constructed using JavaScript / TypeScript:
//
//  let markdown: MarkdownContent = {
//   kind: MarkupKind.Markdown,
//    value: [
//    	'# Header',
//    	'Some text',
//    	'```typescript',
//    'someCode();',
//    '```'
//    ].join('\n')
//  };
//
// NOTE: clients might sanitize the return markdown. A client could decide to
// remove HTML from the markdown to avoid script execution.
type MarkupContent struct {
	// Kind is the type of the Markup
	Kind MarkupKind `json:"kind"`

	// Value is the content itself
	Value string `json:"value"`
}
