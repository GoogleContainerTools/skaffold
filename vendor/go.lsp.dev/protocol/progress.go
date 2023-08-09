// SPDX-FileCopyrightText: 2021 The Go Language Server Authors
// SPDX-License-Identifier: BSD-3-Clause

package protocol

// WorkDoneProgressKind kind of WorkDoneProgress.
//
// @since 3.15.0.
type WorkDoneProgressKind string

// list of WorkDoneProgressKind.
const (
	// WorkDoneProgressKindBegin kind of WorkDoneProgressBegin.
	WorkDoneProgressKindBegin WorkDoneProgressKind = "begin"

	// WorkDoneProgressKindReport kind of WorkDoneProgressReport.
	WorkDoneProgressKindReport WorkDoneProgressKind = "report"

	// WorkDoneProgressKindEnd kind of WorkDoneProgressEnd.
	WorkDoneProgressKindEnd WorkDoneProgressKind = "end"
)

// WorkDoneProgressBegin is the to start progress reporting a "$/progress" notification.
//
// @since 3.15.0.
type WorkDoneProgressBegin struct {
	// Kind is the kind of WorkDoneProgressBegin.
	//
	// It must be WorkDoneProgressKindBegin.
	Kind WorkDoneProgressKind `json:"kind"`

	// Title mandatory title of the progress operation. Used to briefly inform about
	// the kind of operation being performed.
	//
	// Examples: "Indexing" or "Linking dependencies".
	Title string `json:"title"`

	// Cancellable controls if a cancel button should show to allow the user to cancel the
	// long running operation. Clients that don't support cancellation are allowed
	// to ignore the setting.
	Cancellable bool `json:"cancellable,omitempty"`

	// Message is optional, more detailed associated progress message. Contains
	// complementary information to the `title`.
	//
	// Examples: "3/25 files", "project/src/module2", "node_modules/some_dep".
	// If unset, the previous progress message (if any) is still valid.
	Message string `json:"message,omitempty"`

	// Percentage is optional progress percentage to display (value 100 is considered 100%).
	// If not provided infinite progress is assumed and clients are allowed
	// to ignore the `percentage` value in subsequent in report notifications.
	//
	// The value should be steadily rising. Clients are free to ignore values
	// that are not following this rule.
	Percentage uint32 `json:"percentage,omitempty"`
}

// WorkDoneProgressReport is the reporting progress is done.
//
// @since 3.15.0.
type WorkDoneProgressReport struct {
	// Kind is the kind of WorkDoneProgressReport.
	//
	// It must be WorkDoneProgressKindReport.
	Kind WorkDoneProgressKind `json:"kind"`

	// Cancellable controls enablement state of a cancel button.
	//
	// Clients that don't support cancellation or don't support controlling the button's
	// enablement state are allowed to ignore the property.
	Cancellable bool `json:"cancellable,omitempty"`

	// Message is optional, more detailed associated progress message. Contains
	// complementary information to the `title`.
	//
	// Examples: "3/25 files", "project/src/module2", "node_modules/some_dep".
	// If unset, the previous progress message (if any) is still valid.
	Message string `json:"message,omitempty"`

	// Percentage is optional progress percentage to display (value 100 is considered 100%).
	// If not provided infinite progress is assumed and clients are allowed
	// to ignore the `percentage` value in subsequent in report notifications.
	//
	// The value should be steadily rising. Clients are free to ignore values
	// that are not following this rule.
	Percentage uint32 `json:"percentage,omitempty"`
}

// WorkDoneProgressEnd is the signaling the end of a progress reporting is done.
//
// @since 3.15.0.
type WorkDoneProgressEnd struct {
	// Kind is the kind of WorkDoneProgressEnd.
	//
	// It must be WorkDoneProgressKindEnd.
	Kind WorkDoneProgressKind `json:"kind"`

	// Message is optional, a final message indicating to for example indicate the outcome
	// of the operation.
	Message string `json:"message,omitempty"`
}

// WorkDoneProgressParams is a parameter property of report work done progress.
//
// @since 3.15.0.
type WorkDoneProgressParams struct {
	// WorkDoneToken an optional token that a server can use to report work done progress.
	WorkDoneToken *ProgressToken `json:"workDoneToken,omitempty"`
}

// PartialResultParams is the parameter literal used to pass a partial result token.
//
// @since 3.15.0.
type PartialResultParams struct {
	// PartialResultToken an optional token that a server can use to report partial results
	// (for example, streaming) to the client.
	PartialResultToken *ProgressToken `json:"partialResultToken,omitempty"`
}
