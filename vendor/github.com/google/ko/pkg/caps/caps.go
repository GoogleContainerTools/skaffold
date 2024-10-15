// Copyright 2024 ko Build Authors All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package caps implements a subset of Linux capabilities handling
// relevant in the context of authoring container images.
package caps

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"strconv"
	"strings"
)

// Mask captures a set of Linux capabilities
type Mask uint64

// Parse text representation of a single Linux capability.
//
// It accepts all variations recognized by Docker's --cap-add, such as
// 'chown', 'cap_chown', and 'CHOWN'. Additionally, we allow numeric
// values, e.g. '42' to support future capabilities that are not yet
// known to us.
func Parse(s string) (Mask, error) {
	if index, err := strconv.ParseUint(s, 10, 6); err == nil {
		return 1 << index, nil
	}
	name := strings.ToUpper(s)
	if name == "ALL" {
		return allKnownCaps(), nil
	}
	name = strings.TrimPrefix(name, "CAP_")
	if index, ok := nameToIndex[name]; ok {
		return 1 << index, nil
	}
	return 0, fmt.Errorf("unknown capability: %#v", s)
}

func allKnownCaps() Mask {
	var mask Mask
	for _, index := range nameToIndex {
		mask |= 1 << index
	}
	return mask
}

var nameToIndex = map[string]int{
	"CHOWN":            0,
	"DAC_OVERRIDE":     1,
	"DAC_READ_SEARCH":  2,
	"FOWNER":           3,
	"FSETID":           4,
	"KILL":             5,
	"SETGID":           6,
	"SETUID":           7,
	"SETPCAP":          8,
	"LINUX_IMMUTABLE":  9,
	"NET_BIND_SERVICE": 10,
	"NET_BROADCAST":    11,
	"NET_ADMIN":        12,
	"NET_RAW":          13,
	"IPC_LOCK":         14,
	"IPC_OWNER":        15,
	"SYS_MODULE":       16,
	"SYS_RAWIO":        17,
	"SYS_CHROOT":       18,
	"SYS_PTRACE":       19,
	"SYS_PACCT":        20,
	"SYS_ADMIN":        21,
	"SYS_BOOT":         22,
	"SYS_NICE":         23,
	"SYS_RESOURCE":     24,
	"SYS_TIME":         25,
	"SYS_TTY_CONFIG":   26,
	"MKNOD":            27,
	"LEASE":            28,
	"AUDIT_WRITE":      29,
	"AUDIT_CONTROL":    30,
	"SETFCAP":          31,

	"MAC_OVERRIDE":       32,
	"MAC_ADMIN":          33,
	"SYSLOG":             34,
	"WAKE_ALARM":         35,
	"BLOCK_SUSPEND":      36,
	"AUDIT_READ":         37,
	"PERFMON":            38,
	"BPF":                39,
	"CHECKPOINT_RESTORE": 40,
}

// Flags alter certain aspects of capabilities handling
type Flags uint32

const (
	// FlagEffective causes all of the new permitted capabilities to be
	// also raised in the effective set diring execve(2)
	FlagEffective Flags = 1
)

// XattrBytes encodes capabilities in the format of
// security.capability extended filesystem attribute. This is how Linux
// tracks file capabilities internally.
func XattrBytes(permitted, inheritable Mask, flags Flags) ([]byte, error) {
	// Underlying data layout as defined by Linux kernel (vfs_ns_cap_data)
	type vfsNsCapData struct {
		MagicEtc uint32
		Data     [2]struct {
			Permitted   uint32
			Inheritable uint32
		}
	}

	const vfsCapRevision2 = 0x02000000

	data := vfsNsCapData{MagicEtc: vfsCapRevision2 | uint32(flags)}
	data.Data[0].Permitted = uint32(permitted)
	data.Data[0].Inheritable = uint32(inheritable)
	data.Data[1].Permitted = uint32(permitted >> 32)
	data.Data[1].Inheritable = uint32(inheritable >> 32)

	buf := &bytes.Buffer{}
	if err := binary.Write(buf, binary.LittleEndian, data); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

// FileCaps encodes Linux file capabilities
type FileCaps struct {
	permitted, inheritable Mask
	flags                  Flags
}

// NewFileCaps produces file capabilities object from a list of string
// terms. A term is either a single capability name (added as permitted)
// or a cap_from_text(3) clause.
func NewFileCaps(terms ...string) (*FileCaps, error) {
	var permitted, inheritable, effective Mask
	for _, term := range terms {
		var caps, actionList string
		if index := strings.IndexAny(term, "+-="); index != -1 {
			caps, actionList = term[:index], term[index:]
		} else {
			mask, err := Parse(term)
			if err != nil {
				return nil, err
			}
			permitted |= mask
			continue
		}
		// Handling cap_from_text(3) syntax, e.g. cap1,cap2=pie
		if caps == "" && actionList[0] == '=' {
			caps = "all"
		}
		var mask, mask2 Mask
		for _, capname := range strings.Split(caps, ",") {
			m, err := Parse(capname)
			if err != nil {
				return nil, fmt.Errorf("%#v: %w", term, err)
			}
			mask |= m
		}
		for _, c := range actionList {
			switch c {
			case '+':
				mask2 = ^Mask(0)
			case '-':
				mask2 = ^mask
			case '=':
				mask2 = ^Mask(0)
				permitted &= ^mask
				inheritable &= ^mask
				effective &= ^mask
			case 'p':
				permitted = (permitted | mask) & mask2
			case 'i':
				inheritable = (inheritable | mask) & mask2
			case 'e':
				effective = (effective | mask) & mask2
			default:
				return nil, fmt.Errorf("%#v: unknown flag '%c'", term, c)
			}
		}
	}
	if permitted != 0 || inheritable != 0 {
		var flags Flags
		if effective != 0 {
			flags = FlagEffective
		}
		return &FileCaps{permitted: permitted, inheritable: inheritable, flags: flags}, nil
	}
	return nil, nil
}

// ToXattrBytes encodes capabilities in the format of
// security.capability extended filesystem attribute.
func (fc *FileCaps) ToXattrBytes() ([]byte, error) {
	return XattrBytes(fc.permitted, fc.inheritable, fc.flags)
}
