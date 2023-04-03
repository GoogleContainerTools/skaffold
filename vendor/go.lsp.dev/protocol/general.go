// SPDX-FileCopyrightText: 2019 The Go Language Server Authors
// SPDX-License-Identifier: BSD-3-Clause

package protocol

// TraceValue represents a InitializeParams Trace mode.
type TraceValue string

// list of TraceValue.
const (
	// TraceOff disable tracing.
	TraceOff TraceValue = "off"

	// TraceMessage normal tracing mode.
	TraceMessage TraceValue = "message"

	// TraceVerbose verbose tracing mode.
	TraceVerbose TraceValue = "verbose"
)

// ClientInfo information about the client.
//
// @since 3.15.0.
type ClientInfo struct {
	// Name is the name of the client as defined by the client.
	Name string `json:"name"`

	// Version is the client's version as defined by the client.
	Version string `json:"version,omitempty"`
}

// InitializeParams params of Initialize request.
type InitializeParams struct {
	WorkDoneProgressParams

	// ProcessID is the process Id of the parent process that started
	// the server. Is null if the process has not been started by another process.
	// If the parent process is not alive then the server should exit (see exit notification) its process.
	ProcessID int32 `json:"processId"`

	// ClientInfo is the information about the client.
	//
	// @since 3.15.0
	ClientInfo *ClientInfo `json:"clientInfo,omitempty"`

	// Locale is the locale the client is currently showing the user interface
	// in. This must not necessarily be the locale of the operating
	// system.
	//
	// Uses IETF language tags as the value's syntax
	// (See https://en.wikipedia.org/wiki/IETF_language_tag)
	//
	// @since 3.16.0.
	Locale string `json:"locale,omitempty"`

	// RootPath is the rootPath of the workspace. Is null
	// if no folder is open.
	//
	// Deprecated: Use RootURI instead.
	RootPath string `json:"rootPath,omitempty"`

	// RootURI is the rootUri of the workspace. Is null if no
	// folder is open. If both `rootPath` and "rootUri" are set
	// "rootUri" wins.
	//
	// Deprecated: Use WorkspaceFolders instead.
	RootURI DocumentURI `json:"rootUri,omitempty"`

	// InitializationOptions user provided initialization options.
	InitializationOptions interface{} `json:"initializationOptions,omitempty"`

	// Capabilities is the capabilities provided by the client (editor or tool)
	Capabilities ClientCapabilities `json:"capabilities"`

	// Trace is the initial trace setting. If omitted trace is disabled ('off').
	Trace TraceValue `json:"trace,omitempty"`

	// WorkspaceFolders is the workspace folders configured in the client when the server starts.
	// This property is only available if the client supports workspace folders.
	// It can be `null` if the client supports workspace folders but none are
	// configured.
	//
	// @since 3.6.0.
	WorkspaceFolders []WorkspaceFolder `json:"workspaceFolders,omitempty"`
}

// InitializeResult result of ClientCapabilities.
type InitializeResult struct {
	// Capabilities is the capabilities the language server provides.
	Capabilities ServerCapabilities `json:"capabilities"`

	// ServerInfo Information about the server.
	//
	// @since 3.15.0.
	ServerInfo *ServerInfo `json:"serverInfo,omitempty"`
}

// LogTraceParams params of LogTrace notification.
//
// @since 3.16.0.
type LogTraceParams struct {
	// Message is the message to be logged.
	Message string `json:"message"`

	// Verbose is the additional information that can be computed if the "trace" configuration
	// is set to "verbose".
	Verbose TraceValue `json:"verbose,omitempty"`
}

// SetTraceParams params of SetTrace notification.
//
// @since 3.16.0.
type SetTraceParams struct {
	// Value is the new value that should be assigned to the trace setting.
	Value TraceValue `json:"value"`
}

// FileOperationPatternKind is a pattern kind describing if a glob pattern matches a file a folder or
// both.
//
// @since 3.16.0.
type FileOperationPatternKind string

// list of FileOperationPatternKind.
const (
	// FileOperationPatternKindFile is the pattern matches a file only.
	FileOperationPatternKindFile FileOperationPatternKind = "file"

	// FileOperationPatternKindFolder is the pattern matches a folder only.
	FileOperationPatternKindFolder FileOperationPatternKind = "folder"
)

// FileOperationPatternOptions matching options for the file operation pattern.
//
// @since 3.16.0.
type FileOperationPatternOptions struct {
	// IgnoreCase is The pattern should be matched ignoring casing.
	IgnoreCase bool `json:"ignoreCase,omitempty"`
}

// FileOperationPattern a pattern to describe in which file operation requests or notifications
// the server is interested in.
//
// @since 3.16.0.
type FileOperationPattern struct {
	// The glob pattern to match. Glob patterns can have the following syntax:
	//  - `*` to match one or more characters in a path segment
	//  - `?` to match on one character in a path segment
	//  - `**` to match any number of path segments, including none
	//  - `{}` to group conditions (e.g. `**​/*.{ts,js}` matches all TypeScript
	//    and JavaScript files)
	//  - `[]` to declare a range of characters to match in a path segment
	//    (e.g., `example.[0-9]` to match on `example.0`, `example.1`, …)
	//  - `[!...]` to negate a range of characters to match in a path segment
	//    (e.g., `example.[!0-9]` to match on `example.a`, `example.b`, but
	//    not `example.0`)
	Glob string `json:"glob"`

	// Matches whether to match files or folders with this pattern.
	//
	// Matches both if undefined.
	Matches FileOperationPatternKind `json:"matches,omitempty"`

	// Options additional options used during matching.
	Options FileOperationPatternOptions `json:"options,omitempty"`
}

// FileOperationFilter is a filter to describe in which file operation requests or notifications
// the server is interested in.
//
// @since 3.16.0.
type FileOperationFilter struct {
	// Scheme is a URI like "file" or "untitled".
	Scheme string `json:"scheme,omitempty"`

	// Pattern is the actual file operation pattern.
	Pattern FileOperationPattern `json:"pattern"`
}

// CreateFilesParams is the parameters sent in notifications/requests for user-initiated creation
// of files.
//
// @since 3.16.0.
type CreateFilesParams struct {
	// Files an array of all files/folders created in this operation.
	Files []FileCreate `json:"files"`
}

// FileCreate nepresents information on a file/folder create.
//
// @since 3.16.0.
type FileCreate struct {
	// URI is a file:// URI for the location of the file/folder being created.
	URI string `json:"uri"`
}

// RenameFilesParams is the parameters sent in notifications/requests for user-initiated renames
// of files.
//
// @since 3.16.0.
type RenameFilesParams struct {
	// Files an array of all files/folders renamed in this operation. When a folder
	// is renamed, only the folder will be included, and not its children.
	Files []FileRename `json:"files"`
}

// FileRename represents information on a file/folder rename.
//
// @since 3.16.0.
type FileRename struct {
	// OldURI is a file:// URI for the original location of the file/folder being renamed.
	OldURI string `json:"oldUri"`

	// NewURI is a file:// URI for the new location of the file/folder being renamed.
	NewURI string `json:"newUri"`
}

// DeleteFilesParams is the parameters sent in notifications/requests for user-initiated deletes
// of files.
//
// @since 3.16.0.
type DeleteFilesParams struct {
	// Files an array of all files/folders deleted in this operation.
	Files []FileDelete `json:"files"`
}

// FileDelete represents information on a file/folder delete.
//
// @since 3.16.0.
type FileDelete struct {
	// URI is a file:// URI for the location of the file/folder being deleted.
	URI string `json:"uri"`
}

// DocumentHighlightParams params of DocumentHighlight request.
//
// @since 3.15.0.
type DocumentHighlightParams struct {
	TextDocumentPositionParams
	WorkDoneProgressParams
	PartialResultParams
}

// DeclarationParams params of Declaration request.
//
// @since 3.15.0.
type DeclarationParams struct {
	TextDocumentPositionParams
	WorkDoneProgressParams
	PartialResultParams
}

// DefinitionParams params of Definition request.
//
// @since 3.15.0.
type DefinitionParams struct {
	TextDocumentPositionParams
	WorkDoneProgressParams
	PartialResultParams
}

// TypeDefinitionParams params of TypeDefinition request.
//
// @since 3.15.0.
type TypeDefinitionParams struct {
	TextDocumentPositionParams
	WorkDoneProgressParams
	PartialResultParams
}

// ImplementationParams params of Implementation request.
//
// @since 3.15.0.
type ImplementationParams struct {
	TextDocumentPositionParams
	WorkDoneProgressParams
	PartialResultParams
}

// ShowDocumentParams params to show a document.
//
// @since 3.16.0.
type ShowDocumentParams struct {
	// URI is the document uri to show.
	URI URI `json:"uri"`

	// External indicates to show the resource in an external program.
	// To show for example `https://code.visualstudio.com/`
	// in the default WEB browser set `external` to `true`.
	External bool `json:"external,omitempty"`

	// TakeFocus an optional property to indicate whether the editor
	// showing the document should take focus or not.
	// Clients might ignore this property if an external
	// program is started.
	TakeFocus bool `json:"takeFocus,omitempty"`

	// Selection an optional selection range if the document is a text
	// document. Clients might ignore the property if an
	// external program is started or the file is not a text
	// file.
	Selection *Range `json:"selection,omitempty"`
}

// ShowDocumentResult is the result of an show document request.
//
// @since 3.16.0.
type ShowDocumentResult struct {
	// Success a boolean indicating if the show was successful.
	Success bool `json:"success"`
}

// ServerInfo Information about the server.
//
// @since 3.15.0.
type ServerInfo struct {
	// Name is the name of the server as defined by the server.
	Name string `json:"name"`

	// Version is the server's version as defined by the server.
	Version string `json:"version,omitempty"`
}

// InitializeError known error codes for an "InitializeError".
type InitializeError struct {
	// Retry indicates whether the client execute the following retry logic:
	// (1) show the message provided by the ResponseError to the user
	// (2) user selects retry or cancel
	// (3) if user selected retry the initialize method is sent again.
	Retry bool `json:"retry,omitempty"`
}

// ReferencesOptions ReferencesProvider options.
//
// @since 3.15.0.
type ReferencesOptions struct {
	WorkDoneProgressOptions
}

// WorkDoneProgressOptions WorkDoneProgress options.
//
// @since 3.15.0.
type WorkDoneProgressOptions struct {
	WorkDoneProgress bool `json:"workDoneProgress,omitempty"`
}

// LinkedEditingRangeParams params for the LinkedEditingRange request.
//
// @since 3.16.0.
type LinkedEditingRangeParams struct {
	TextDocumentPositionParams
	WorkDoneProgressParams
}

// LinkedEditingRanges result of LinkedEditingRange request.
//
// @since 3.16.0.
type LinkedEditingRanges struct {
	// Ranges a list of ranges that can be renamed together.
	//
	// The ranges must have identical length and contain identical text content.
	//
	// The ranges cannot overlap.
	Ranges []Range `json:"ranges"`

	// WordPattern an optional word pattern (regular expression) that describes valid contents for
	// the given ranges.
	//
	// If no pattern is provided, the client configuration's word pattern will be used.
	WordPattern string `json:"wordPattern,omitempty"`
}

// MonikerParams params for the Moniker request.
//
// @since 3.16.0.
type MonikerParams struct {
	TextDocumentPositionParams
	WorkDoneProgressParams
	PartialResultParams
}

// UniquenessLevel is the Moniker uniqueness level to define scope of the moniker.
//
// @since 3.16.0.
type UniquenessLevel string

// list of UniquenessLevel.
const (
	// UniquenessLevelDocument is the moniker is only unique inside a document.
	UniquenessLevelDocument UniquenessLevel = "document"

	// UniquenessLevelProject is the moniker is unique inside a project for which a dump got created.
	UniquenessLevelProject UniquenessLevel = "project"

	// UniquenessLevelGroup is the moniker is unique inside the group to which a project belongs.
	UniquenessLevelGroup UniquenessLevel = "group"

	// UniquenessLevelScheme is the moniker is unique inside the moniker scheme.
	UniquenessLevelScheme UniquenessLevel = "scheme"

	// UniquenessLevelGlobal is the moniker is globally unique.
	UniquenessLevelGlobal UniquenessLevel = "global"
)

// MonikerKind is the moniker kind.
//
// @since 3.16.0.
type MonikerKind string

// list of MonikerKind.
const (
	// MonikerKindImport is the moniker represent a symbol that is imported into a project.
	MonikerKindImport MonikerKind = "import"

	// MonikerKindExport is the moniker represents a symbol that is exported from a project.
	MonikerKindExport MonikerKind = "export"

	// MonikerKindLocal is the moniker represents a symbol that is local to a project (e.g. a local
	// variable of a function, a class not visible outside the project, ...).
	MonikerKindLocal MonikerKind = "local"
)

// Moniker definition to match LSIF 0.5 moniker definition.
//
// @since 3.16.0.
type Moniker struct {
	// Scheme is the scheme of the moniker. For example tsc or .Net.
	Scheme string `json:"scheme"`

	// Identifier is the identifier of the moniker.
	//
	// The value is opaque in LSIF however schema owners are allowed to define the structure if they want.
	Identifier string `json:"identifier"`

	// Unique is the scope in which the moniker is unique.
	Unique UniquenessLevel `json:"unique"`

	// Kind is the moniker kind if known.
	Kind MonikerKind `json:"kind,omitempty"`
}

// StaticRegistrationOptions staticRegistration options to be returned in the initialize request.
type StaticRegistrationOptions struct {
	// ID is the id used to register the request. The id can be used to deregister
	// the request again. See also Registration#id.
	ID string `json:"id,omitempty"`
}

// DocumentLinkRegistrationOptions DocumentLinkRegistration options.
type DocumentLinkRegistrationOptions struct {
	TextDocumentRegistrationOptions

	// ResolveProvider document links have a resolve provider as well.
	ResolveProvider bool `json:"resolveProvider,omitempty"`
}

// InitializedParams params of Initialized notification.
type InitializedParams struct{}

// WorkspaceFolders represents a slice of WorkspaceFolder.
type WorkspaceFolders []WorkspaceFolder
