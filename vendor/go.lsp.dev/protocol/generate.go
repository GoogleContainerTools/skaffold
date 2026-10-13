// Copyright 2024 The Go Language Server Authors
// SPDX-License-Identifier: BSD-3-Clause

package protocol

// Regenerate the LSP types from the meta-model. After generation the files are
// formatted with gofumpt (see the Makefile "generate" target).

//go:generate go run go.lsp.dev/protocol/internal/genlsp/cmd/genlsp -input https://raw.githubusercontent.com/microsoft/vscode-languageserver-node/refs/tags/release/protocol/3.18.0/protocol/metaModel.json -output . -pkg protocol
