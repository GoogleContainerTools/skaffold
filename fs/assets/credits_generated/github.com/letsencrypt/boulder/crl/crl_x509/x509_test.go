// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package crl_x509

import (
	_ "crypto/sha256"
	_ "crypto/sha512"
	"encoding/base64"
)

func fromBase64(in string) []byte {
	out := make([]byte, base64.StdEncoding.DecodedLen(len(in)))
	n, err := base64.StdEncoding.Decode(out, []byte(in))
	if err != nil {
		panic("failed to base64 decode")
	}
	return out[:n]
}
