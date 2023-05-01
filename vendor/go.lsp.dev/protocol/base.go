// SPDX-FileCopyrightText: 2021 The Go Language Server Authors
// SPDX-License-Identifier: BSD-3-Clause

package protocol

// CancelParams params of cancelRequest.
type CancelParams struct {
	// ID is the request id to cancel.
	ID interface{} `json:"id"` // int32 | string
}

// ProgressParams params of Progress netification.
//
// @since 3.15.0.
type ProgressParams struct {
	// Token is the progress token provided by the client or server.
	Token ProgressToken `json:"token"`

	// Value is the progress data.
	Value interface{} `json:"value"`
}
