// SPDX-FileCopyrightText: 2019 The Go Language Server Authors
// SPDX-License-Identifier: BSD-3-Clause

package protocol

import (
	"strconv"
)

// Diagnostic represents a diagnostic, such as a compiler error or warning.
//
// Diagnostic objects are only valid in the scope of a resource.
type Diagnostic struct {
	// Range is the range at which the message applies.
	Range Range `json:"range"`

	// Severity is the diagnostic's severity. Can be omitted. If omitted it is up to the
	// client to interpret diagnostics as error, warning, info or hint.
	Severity DiagnosticSeverity `json:"severity,omitempty"`

	// Code is the diagnostic's code, which might appear in the user interface.
	Code interface{} `json:"code,omitempty"` // int32 | string;

	// CodeDescription an optional property to describe the error code.
	//
	// @since 3.16.0.
	CodeDescription *CodeDescription `json:"codeDescription,omitempty"`

	// Source a human-readable string describing the source of this
	// diagnostic, e.g. 'typescript' or 'super lint'.
	Source string `json:"source,omitempty"`

	// Message is the diagnostic's message.
	Message string `json:"message"`

	// Tags is the additional metadata about the diagnostic.
	//
	// @since 3.15.0.
	Tags []DiagnosticTag `json:"tags,omitempty"`

	// RelatedInformation an array of related diagnostic information, e.g. when symbol-names within
	// a scope collide all definitions can be marked via this property.
	RelatedInformation []DiagnosticRelatedInformation `json:"relatedInformation,omitempty"`

	// Data is a data entry field that is preserved between a
	// "textDocument/publishDiagnostics" notification and
	// "textDocument/codeAction" request.
	//
	// @since 3.16.0.
	Data interface{} `json:"data,omitempty"`
}

// DiagnosticSeverity indicates the severity of a Diagnostic message.
type DiagnosticSeverity float64

const (
	// DiagnosticSeverityError reports an error.
	DiagnosticSeverityError DiagnosticSeverity = 1

	// DiagnosticSeverityWarning reports a warning.
	DiagnosticSeverityWarning DiagnosticSeverity = 2

	// DiagnosticSeverityInformation reports an information.
	DiagnosticSeverityInformation DiagnosticSeverity = 3

	// DiagnosticSeverityHint reports a hint.
	DiagnosticSeverityHint DiagnosticSeverity = 4
)

// String implements fmt.Stringer.
func (d DiagnosticSeverity) String() string {
	switch d {
	case DiagnosticSeverityError:
		return "Error"
	case DiagnosticSeverityWarning:
		return "Warning"
	case DiagnosticSeverityInformation:
		return "Information"
	case DiagnosticSeverityHint:
		return "Hint"
	default:
		return strconv.FormatFloat(float64(d), 'f', -10, 64)
	}
}

// CodeDescription is the structure to capture a description for an error code.
//
// @since 3.16.0.
type CodeDescription struct {
	// Href an URI to open with more information about the diagnostic error.
	Href URI `json:"href"`
}

// DiagnosticTag is the diagnostic tags.
//
// @since 3.15.0.
type DiagnosticTag float64

// list of DiagnosticTag.
const (
	// DiagnosticTagUnnecessary unused or unnecessary code.
	//
	// Clients are allowed to render diagnostics with this tag faded out instead of having
	// an error squiggle.
	DiagnosticTagUnnecessary DiagnosticTag = 1

	// DiagnosticTagDeprecated deprecated or obsolete code.
	//
	// Clients are allowed to rendered diagnostics with this tag strike through.
	DiagnosticTagDeprecated DiagnosticTag = 2
)

// String implements fmt.Stringer.
func (d DiagnosticTag) String() string {
	switch d {
	case DiagnosticTagUnnecessary:
		return "Unnecessary"
	case DiagnosticTagDeprecated:
		return "Deprecated"
	default:
		return strconv.FormatFloat(float64(d), 'f', -10, 64)
	}
}

// DiagnosticRelatedInformation represents a related message and source code location for a diagnostic.
//
// This should be used to point to code locations that cause or related to a diagnostics, e.g when duplicating
// a symbol in a scope.
type DiagnosticRelatedInformation struct {
	// Location is the location of this related diagnostic information.
	Location Location `json:"location"`

	// Message is the message of this related diagnostic information.
	Message string `json:"message"`
}

// PublishDiagnosticsParams represents a params of PublishDiagnostics notification.
type PublishDiagnosticsParams struct {
	// URI is the URI for which diagnostic information is reported.
	URI DocumentURI `json:"uri"`

	// Version optional the version number of the document the diagnostics are published for.
	//
	// @since 3.15
	Version uint32 `json:"version,omitempty"`

	// Diagnostics an array of diagnostic information items.
	Diagnostics []Diagnostic `json:"diagnostics"`
}
