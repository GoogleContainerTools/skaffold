// SPDX-FileCopyrightText: 2019 The Go Language Server Authors
// SPDX-License-Identifier: BSD-3-Clause

//go:build gojay
// +build gojay

package protocol

import "github.com/francoispqt/gojay"

// MarshalJSONObject implements gojay.MarshalerJSONObject.
func (v *WorkspaceFolder) MarshalJSONObject(enc *gojay.Encoder) {
	enc.StringKey(keyURI, v.URI)
	enc.StringKey(keyName, v.Name)
}

// IsNil returns wether the structure is nil value or not.
func (v *WorkspaceFolder) IsNil() bool { return v == nil }

// UnmarshalJSONObject implements gojay's UnmarshalerJSONObject.
func (v *WorkspaceFolder) UnmarshalJSONObject(dec *gojay.Decoder, k string) error {
	switch k {
	case keyURI:
		return dec.String(&v.URI)
	case keyName:
		return dec.String(&v.Name)
	}
	return nil
}

// NKeys returns the number of keys to unmarshal.
func (v *WorkspaceFolder) NKeys() int { return 2 }

// compile time check whether the WorkspaceFolder implements a gojay.MarshalerJSONObject and gojay.UnmarshalerJSONObject interfaces.
var (
	_ gojay.MarshalerJSONObject   = (*WorkspaceFolder)(nil)
	_ gojay.UnmarshalerJSONObject = (*WorkspaceFolder)(nil)
)

// MarshalJSONObject implements gojay.MarshalerJSONObject.
func (v *DidChangeWorkspaceFoldersParams) MarshalJSONObject(enc *gojay.Encoder) {
	enc.ObjectKey(keyEvent, &v.Event)
}

// IsNil returns wether the structure is nil value or not.
func (v *DidChangeWorkspaceFoldersParams) IsNil() bool { return v == nil }

// UnmarshalJSONObject implements gojay's UnmarshalerJSONObject.
func (v *DidChangeWorkspaceFoldersParams) UnmarshalJSONObject(dec *gojay.Decoder, k string) error {
	if k == keyEvent {
		return dec.Object(&v.Event)
	}
	return nil
}

// NKeys returns the number of keys to unmarshal.
func (v *DidChangeWorkspaceFoldersParams) NKeys() int { return 1 }

// compile time check whether the DidChangeWorkspaceFoldersParams implements a gojay.MarshalerJSONObject and gojay.UnmarshalerJSONObject interfaces.
var (
	_ gojay.MarshalerJSONObject   = (*DidChangeWorkspaceFoldersParams)(nil)
	_ gojay.UnmarshalerJSONObject = (*DidChangeWorkspaceFoldersParams)(nil)
)

// MarshalJSONObject implements gojay.MarshalerJSONObject.
func (v *WorkspaceFoldersChangeEvent) MarshalJSONObject(enc *gojay.Encoder) {
	enc.ArrayKey(keyAdded, (*WorkspaceFolders)(&v.Added))
	enc.ArrayKey(keyRemoved, (*WorkspaceFolders)(&v.Removed))
}

// IsNil returns wether the structure is nil value or not.
func (v *WorkspaceFoldersChangeEvent) IsNil() bool { return v == nil }

// UnmarshalJSONObject implements gojay's UnmarshalerJSONObject.
func (v *WorkspaceFoldersChangeEvent) UnmarshalJSONObject(dec *gojay.Decoder, k string) error {
	switch k {
	case keyAdded:
		return dec.Array((*WorkspaceFolders)(&v.Added))
	case keyRemoved:
		return dec.Array((*WorkspaceFolders)(&v.Removed))
	}
	return nil
}

// NKeys returns the number of keys to unmarshal.
func (v *WorkspaceFoldersChangeEvent) NKeys() int { return 2 }

// compile time check whether the WorkspaceFoldersChangeEvent implements a gojay.MarshalerJSONObject and gojay.UnmarshalerJSONObject interfaces.
var (
	_ gojay.MarshalerJSONObject   = (*WorkspaceFoldersChangeEvent)(nil)
	_ gojay.UnmarshalerJSONObject = (*WorkspaceFoldersChangeEvent)(nil)
)

// MarshalJSONObject implements gojay.MarshalerJSONObject.
func (v *DidChangeConfigurationParams) MarshalJSONObject(enc *gojay.Encoder) {
	enc.AddInterfaceKeyOmitEmpty(keySettings, v.Settings)
}

// IsNil returns wether the structure is nil value or not.
func (v *DidChangeConfigurationParams) IsNil() bool { return v == nil }

// UnmarshalJSONObject implements gojay's UnmarshalerJSONObject.
func (v *DidChangeConfigurationParams) UnmarshalJSONObject(dec *gojay.Decoder, k string) error {
	if k == keySettings {
		return dec.Interface(&v.Settings)
	}
	return nil
}

// NKeys returns the number of keys to unmarshal.
func (v *DidChangeConfigurationParams) NKeys() int { return 1 }

// compile time check whether the DidChangeConfigurationParams implements a gojay.MarshalerJSONObject and gojay.UnmarshalerJSONObject interfaces.
var (
	_ gojay.MarshalerJSONObject   = (*DidChangeConfigurationParams)(nil)
	_ gojay.UnmarshalerJSONObject = (*DidChangeConfigurationParams)(nil)
)

// ConfigurationItems represents a slice of ConfigurationItem.
type ConfigurationItems []ConfigurationItem

// MarshalJSONArray implements gojay.MarshalerJSONArray.
func (v ConfigurationItems) MarshalJSONArray(enc *gojay.Encoder) {
	for i := range v {
		enc.ObjectOmitEmpty(&v[i])
	}
}

// IsNil implements gojay.MarshalerJSONArray.
func (v ConfigurationItems) IsNil() bool { return len(v) == 0 }

// UnmarshalJSONArray implements gojay.UnmarshalerJSONArray.
func (v *ConfigurationItems) UnmarshalJSONArray(dec *gojay.Decoder) error {
	t := ConfigurationItem{}
	if err := dec.Object(&t); err != nil {
		return err
	}
	*v = append(*v, t)
	return nil
}

// compile time check whether the configurationItem implements a gojay.MarshalerJSONArray and gojay.UnmarshalerJSONArray interfaces.
var (
	_ gojay.MarshalerJSONArray   = (*ConfigurationItems)(nil)
	_ gojay.UnmarshalerJSONArray = (*ConfigurationItems)(nil)
)

// MarshalJSONObject implements gojay.MarshalerJSONObject.
func (v *ConfigurationParams) MarshalJSONObject(enc *gojay.Encoder) {
	enc.ArrayKey(keyItems, (*ConfigurationItems)(&v.Items))
}

// IsNil returns wether the structure is nil value or not.
func (v *ConfigurationParams) IsNil() bool { return v == nil }

// UnmarshalJSONObject implements gojay's UnmarshalerJSONObject.
func (v *ConfigurationParams) UnmarshalJSONObject(dec *gojay.Decoder, k string) error {
	if k == keyItems {
		if v.Items == nil {
			v.Items = []ConfigurationItem{}
		}
		return dec.Array((*ConfigurationItems)(&v.Items))
	}
	return nil
}

// NKeys returns the number of keys to unmarshal.
func (v *ConfigurationParams) NKeys() int { return 1 }

// compile time check whether the ConfigurationParams implements a gojay.MarshalerJSONObject and gojay.UnmarshalerJSONObject interfaces.
var (
	_ gojay.MarshalerJSONObject   = (*ConfigurationParams)(nil)
	_ gojay.UnmarshalerJSONObject = (*ConfigurationParams)(nil)
)

// MarshalJSONObject implements gojay.MarshalerJSONObject.
func (v *ConfigurationItem) MarshalJSONObject(enc *gojay.Encoder) {
	enc.StringKeyOmitEmpty(keyScopeURI, string(v.ScopeURI))
	enc.StringKeyOmitEmpty(keySection, v.Section)
}

// IsNil returns wether the structure is nil value or not.
func (v *ConfigurationItem) IsNil() bool { return v == nil }

// UnmarshalJSONObject implements gojay's UnmarshalerJSONObject.
func (v *ConfigurationItem) UnmarshalJSONObject(dec *gojay.Decoder, k string) error {
	switch k {
	case keyScopeURI:
		return dec.String((*string)(&v.ScopeURI))
	case keySection:
		return dec.String(&v.Section)
	}
	return nil
}

// NKeys returns the number of keys to unmarshal.
func (v *ConfigurationItem) NKeys() int { return 2 }

// compile time check whether the ConfigurationItem implements a gojay.MarshalerJSONObject and gojay.UnmarshalerJSONObject interfaces.
var (
	_ gojay.MarshalerJSONObject   = (*ConfigurationItem)(nil)
	_ gojay.UnmarshalerJSONObject = (*ConfigurationItem)(nil)
)

// FileEvents represents a FileEvent pointer slice.
type FileEvents []*FileEvent

// MarshalJSONArray implements gojay.MarshalerJSONArray.
func (v FileEvents) MarshalJSONArray(enc *gojay.Encoder) {
	for i := range v {
		enc.ObjectOmitEmpty(v[i])
	}
}

// IsNil implements gojay.MarshalerJSONArray.
func (v FileEvents) IsNil() bool { return len(v) == 0 }

// UnmarshalJSONArray implements gojay.UnmarshalerJSONArray.
func (v *FileEvents) UnmarshalJSONArray(dec *gojay.Decoder) error {
	t := &FileEvent{}
	if err := dec.Object(t); err != nil {
		return err
	}
	*v = append(*v, t)
	return nil
}

// compile time check whether the FileEvents implements a gojay.MarshalerJSONArray and gojay.UnmarshalerJSONArray interfaces.
var (
	_ gojay.MarshalerJSONArray   = (*FileEvents)(nil)
	_ gojay.UnmarshalerJSONArray = (*FileEvents)(nil)
)

// MarshalJSONObject implements gojay.MarshalerJSONObject.
func (v *DidChangeWatchedFilesParams) MarshalJSONObject(enc *gojay.Encoder) {
	enc.ArrayKeyOmitEmpty(keyChanges, (*FileEvents)(&v.Changes))
}

// IsNil returns wether the structure is nil value or not.
func (v *DidChangeWatchedFilesParams) IsNil() bool { return v == nil }

// UnmarshalJSONObject implements gojay's UnmarshalerJSONObject.
func (v *DidChangeWatchedFilesParams) UnmarshalJSONObject(dec *gojay.Decoder, k string) error {
	if k == keyChanges {
		if v.Changes == nil {
			v.Changes = []*FileEvent{}
		}
		return dec.Array((*FileEvents)(&v.Changes))
	}
	return nil
}

// NKeys returns the number of keys to unmarshal.
func (v *DidChangeWatchedFilesParams) NKeys() int { return 1 }

// compile time check whether the DidChangeWatchedFilesParams implements a gojay.MarshalerJSONObject and gojay.UnmarshalerJSONObject interfaces.
var (
	_ gojay.MarshalerJSONObject   = (*DidChangeWatchedFilesParams)(nil)
	_ gojay.UnmarshalerJSONObject = (*DidChangeWatchedFilesParams)(nil)
)

// MarshalJSONObject implements gojay.MarshalerJSONObject.
func (v *FileEvent) MarshalJSONObject(enc *gojay.Encoder) {
	enc.Float64Key(keyType, float64(v.Type))
	enc.StringKey(keyURI, string(v.URI))
}

// IsNil returns wether the structure is nil value or not.
func (v *FileEvent) IsNil() bool { return v == nil }

// UnmarshalJSONObject implements gojay's UnmarshalerJSONObject.
func (v *FileEvent) UnmarshalJSONObject(dec *gojay.Decoder, k string) error {
	switch k {
	case keyType:
		return dec.Float64((*float64)(&v.Type))
	case keyURI:
		return dec.String((*string)(&v.URI))
	}
	return nil
}

// NKeys returns the number of keys to unmarshal.
func (v *FileEvent) NKeys() int { return 2 }

// compile time check whether the FileEvent implements a gojay.MarshalerJSONObject and gojay.UnmarshalerJSONObject interfaces.
var (
	_ gojay.MarshalerJSONObject   = (*FileEvent)(nil)
	_ gojay.UnmarshalerJSONObject = (*FileEvent)(nil)
)

// FileSystemWatchers represents a FileSystemWatcher slice.
type FileSystemWatchers []FileSystemWatcher

// MarshalJSONArray implements gojay.MarshalerJSONArray.
func (v FileSystemWatchers) MarshalJSONArray(enc *gojay.Encoder) {
	for i := range v {
		enc.ObjectOmitEmpty(&v[i])
	}
}

// IsNil implements gojay.MarshalerJSONArray.
func (v FileSystemWatchers) IsNil() bool { return len(v) == 0 }

// UnmarshalJSONArray implements gojay.UnmarshalerJSONArray.
func (v *FileSystemWatchers) UnmarshalJSONArray(dec *gojay.Decoder) error {
	t := FileSystemWatcher{}
	if err := dec.Object(&t); err != nil {
		return err
	}
	*v = append(*v, t)
	return nil
}

// compile time check whether the fileSystemWatchers implements a gojay.MarshalerJSONArray and gojay.UnmarshalerJSONArray interfaces.
var (
	_ gojay.MarshalerJSONArray   = (*FileSystemWatchers)(nil)
	_ gojay.UnmarshalerJSONArray = (*FileSystemWatchers)(nil)
)

// MarshalJSONObject implements gojay.MarshalerJSONObject.
func (v *DidChangeWatchedFilesRegistrationOptions) MarshalJSONObject(enc *gojay.Encoder) {
	enc.ArrayKey(keyWatchers, (*FileSystemWatchers)(&v.Watchers))
}

// IsNil returns wether the structure is nil value or not.
func (v *DidChangeWatchedFilesRegistrationOptions) IsNil() bool { return v == nil }

// UnmarshalJSONObject implements gojay's UnmarshalerJSONObject.
func (v *DidChangeWatchedFilesRegistrationOptions) UnmarshalJSONObject(dec *gojay.Decoder, k string) error {
	if k == keyWatchers {
		return dec.Array((*FileSystemWatchers)(&v.Watchers))
	}
	return nil
}

// NKeys returns the number of keys to unmarshal.
func (v *DidChangeWatchedFilesRegistrationOptions) NKeys() int { return 1 }

// compile time check whether the DidChangeWatchedFilesRegistrationOptions implements a gojay.MarshalerJSONObject and gojay.UnmarshalerJSONObject interfaces.
var (
	_ gojay.MarshalerJSONObject   = (*DidChangeWatchedFilesRegistrationOptions)(nil)
	_ gojay.UnmarshalerJSONObject = (*DidChangeWatchedFilesRegistrationOptions)(nil)
)

// MarshalJSONObject implements gojay.MarshalerJSONObject.
func (v *FileSystemWatcher) MarshalJSONObject(enc *gojay.Encoder) {
	enc.StringKey(keyGlobPattern, v.GlobPattern)
	enc.Float64KeyOmitEmpty(keyKind, float64(v.Kind))
}

// IsNil returns wether the structure is nil value or not.
func (v *FileSystemWatcher) IsNil() bool { return v == nil }

// UnmarshalJSONObject implements gojay's UnmarshalerJSONObject.
func (v *FileSystemWatcher) UnmarshalJSONObject(dec *gojay.Decoder, k string) error {
	switch k {
	case keyGlobPattern:
		return dec.String(&v.GlobPattern)
	case keyKind:
		return dec.Float64((*float64)(&v.Kind))
	}
	return nil
}

// NKeys returns the number of keys to unmarshal.
func (v *FileSystemWatcher) NKeys() int { return 2 }

// compile time check whether the FileSystemWatcher implements a gojay.MarshalerJSONObject and gojay.UnmarshalerJSONObject interfaces.
var (
	_ gojay.MarshalerJSONObject   = (*FileSystemWatcher)(nil)
	_ gojay.UnmarshalerJSONObject = (*FileSystemWatcher)(nil)
)

// MarshalJSONObject implements gojay.MarshalerJSONObject.
func (v *WorkspaceSymbolParams) MarshalJSONObject(enc *gojay.Encoder) {
	encodeProgressToken(enc, keyWorkDoneToken, v.WorkDoneToken)
	encodeProgressToken(enc, keyPartialResultToken, v.PartialResultToken)
	enc.StringKey(keyQuery, v.Query)
}

// IsNil returns wether the structure is nil value or not.
func (v *WorkspaceSymbolParams) IsNil() bool { return v == nil }

// UnmarshalJSONObject implements gojay's UnmarshalerJSONObject.
func (v *WorkspaceSymbolParams) UnmarshalJSONObject(dec *gojay.Decoder, k string) error {
	switch k {
	case keyWorkDoneToken:
		return decodeProgressToken(dec, k, keyWorkDoneToken, v.WorkDoneToken)
	case keyPartialResultToken:
		return decodeProgressToken(dec, k, keyPartialResultToken, v.PartialResultToken)
	case keyQuery:
		return dec.String(&v.Query)
	}
	return nil
}

// NKeys returns the number of keys to unmarshal.
func (v *WorkspaceSymbolParams) NKeys() int { return 3 }

// compile time check whether the WorkspaceSymbolParams implements a gojay.MarshalerJSONObject and gojay.UnmarshalerJSONObject interfaces.
var (
	_ gojay.MarshalerJSONObject   = (*WorkspaceSymbolParams)(nil)
	_ gojay.UnmarshalerJSONObject = (*WorkspaceSymbolParams)(nil)
)

// MarshalJSONObject implements gojay.MarshalerJSONObject.
func (v *ExecuteCommandParams) MarshalJSONObject(enc *gojay.Encoder) {
	encodeProgressToken(enc, keyWorkDoneToken, v.WorkDoneToken)
	enc.StringKey(keyCommand, v.Command)
	enc.ArrayKeyOmitEmpty(keyArguments, (*Interfaces)(&v.Arguments))
}

// IsNil returns wether the structure is nil value or not.
func (v *ExecuteCommandParams) IsNil() bool { return v == nil }

// UnmarshalJSONObject implements gojay's UnmarshalerJSONObject.
func (v *ExecuteCommandParams) UnmarshalJSONObject(dec *gojay.Decoder, k string) error {
	switch k {
	case keyWorkDoneToken:
		return decodeProgressToken(dec, k, keyWorkDoneToken, v.WorkDoneToken)
	case keyCommand:
		return dec.String(&v.Command)
	case keyArguments:
		return dec.Array((*Interfaces)(&v.Arguments))
	}
	return nil
}

// NKeys returns the number of keys to unmarshal.
func (v *ExecuteCommandParams) NKeys() int { return 3 }

// compile time check whether the ExecuteCommandParams implements a gojay.MarshalerJSONObject and gojay.UnmarshalerJSONObject interfaces.
var (
	_ gojay.MarshalerJSONObject   = (*ExecuteCommandParams)(nil)
	_ gojay.UnmarshalerJSONObject = (*ExecuteCommandParams)(nil)
)

// MarshalJSONObject implements gojay.MarshalerJSONObject.
func (v *ExecuteCommandRegistrationOptions) MarshalJSONObject(enc *gojay.Encoder) {
	enc.ArrayKey(keyCommands, (*Strings)(&v.Commands))
}

// IsNil returns wether the structure is nil value or not.
func (v *ExecuteCommandRegistrationOptions) IsNil() bool { return v == nil }

// UnmarshalJSONObject implements gojay's UnmarshalerJSONObject.
func (v *ExecuteCommandRegistrationOptions) UnmarshalJSONObject(dec *gojay.Decoder, k string) error {
	if k == keyCommands {
		return dec.Array((*Strings)(&v.Commands))
	}
	return nil
}

// NKeys returns the number of keys to unmarshal.
func (v *ExecuteCommandRegistrationOptions) NKeys() int { return 1 }

// compile time check whether the ExecuteCommandRegistrationOptions implements a gojay.MarshalerJSONObject and gojay.UnmarshalerJSONObject interfaces.
var (
	_ gojay.MarshalerJSONObject   = (*ExecuteCommandRegistrationOptions)(nil)
	_ gojay.UnmarshalerJSONObject = (*ExecuteCommandRegistrationOptions)(nil)
)

// MarshalJSONObject implements gojay.MarshalerJSONObject.
func (v *ApplyWorkspaceEditParams) MarshalJSONObject(enc *gojay.Encoder) {
	enc.StringKeyOmitEmpty(keyLabel, v.Label)
	enc.ObjectKey(keyEdit, &v.Edit)
}

// IsNil returns wether the structure is nil value or not.
func (v *ApplyWorkspaceEditParams) IsNil() bool { return v == nil }

// UnmarshalJSONObject implements gojay's UnmarshalerJSONObject.
func (v *ApplyWorkspaceEditParams) UnmarshalJSONObject(dec *gojay.Decoder, k string) error {
	switch k {
	case keyLabel:
		return dec.String(&v.Label)
	case keyEdit:
		return dec.Object(&v.Edit)
	}
	return nil
}

// NKeys returns the number of keys to unmarshal.
func (v *ApplyWorkspaceEditParams) NKeys() int { return 2 }

// compile time check whether the ApplyWorkspaceEditParams implements a gojay.MarshalerJSONObject and gojay.UnmarshalerJSONObject interfaces.
var (
	_ gojay.MarshalerJSONObject   = (*ApplyWorkspaceEditParams)(nil)
	_ gojay.UnmarshalerJSONObject = (*ApplyWorkspaceEditParams)(nil)
)

// MarshalJSONObject implements gojay.MarshalerJSONObject.
func (v *ApplyWorkspaceEditResponse) MarshalJSONObject(enc *gojay.Encoder) {
	enc.BoolKey(keyApplied, v.Applied)
	enc.StringKeyOmitEmpty(keyFailureReason, v.FailureReason)
	enc.Uint32KeyOmitEmpty(keyFailedChange, v.FailedChange)
}

// IsNil returns wether the structure is nil value or not.
func (v *ApplyWorkspaceEditResponse) IsNil() bool { return v == nil }

// UnmarshalJSONObject implements gojay's UnmarshalerJSONObject.
func (v *ApplyWorkspaceEditResponse) UnmarshalJSONObject(dec *gojay.Decoder, k string) error {
	switch k {
	case keyApplied:
		return dec.Bool(&v.Applied)
	case keyFailureReason:
		return dec.String(&v.FailureReason)
	case keyFailedChange:
		return dec.Uint32(&v.FailedChange)
	}
	return nil
}

// NKeys returns the number of keys to unmarshal.
func (v *ApplyWorkspaceEditResponse) NKeys() int { return 3 }

// compile time check whether the ApplyWorkspaceEditResponse implements a gojay.MarshalerJSONObject and gojay.UnmarshalerJSONObject interfaces.
var (
	_ gojay.MarshalerJSONObject   = (*ApplyWorkspaceEditResponse)(nil)
	_ gojay.UnmarshalerJSONObject = (*ApplyWorkspaceEditResponse)(nil)
)
