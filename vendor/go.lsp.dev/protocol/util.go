// SPDX-FileCopyrightText: 2019 The Go Language Server Authors
// SPDX-License-Identifier: BSD-3-Clause

package protocol

// NewVersion returns the int32 pointer converted i.
func NewVersion(i int32) *int32 {
	return &i
}
