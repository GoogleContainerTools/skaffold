// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Originally from: https://github.com/go/blob/master/src/crypto/sha1/sha1block.go

package sha1cd

import (
	"math/bits"

	"github.com/pjbgf/sha1cd/ubc"
)

const (
	msize = 80

	_K0 = 0x5A827999
	_K1 = 0x6ED9EBA1
	_K2 = 0x8F1BBCDC
	_K3 = 0xCA62C1D6
)

// TODO: Implement SIMD support.
func block(dig *digest, p []byte) {
	blockGeneric(dig, p)
}

// blockGeneric is a portable, pure Go version of the SHA-1 block step.
// It's used by sha1block_generic.go and tests.
func blockGeneric(dig *digest, p []byte) {
	var w [16]uint32

	h0, h1, h2, h3, h4 := dig.h[0], dig.h[1], dig.h[2], dig.h[3], dig.h[4]
	for len(p) >= chunk {
		m1 := make([]uint32, msize)
		bcol := false

		// Can interlace the computation of w with the
		// rounds below if needed for speed.
		for i := 0; i < 16; i++ {
			j := i * 4
			w[i] = uint32(p[j])<<24 | uint32(p[j+1])<<16 | uint32(p[j+2])<<8 | uint32(p[j+3])
		}

		a, b, c, d, e := h0, h1, h2, h3, h4

		// Each of the four 20-iteration rounds
		// differs only in the computation of f and
		// the choice of K (_K0, _K1, etc).
		i := 0
		for ; i < 16; i++ {
			// Store pre-step compression state for the collision detection.
			dig.cs[i] = [5]uint32{a, b, c, d, e}

			f := b&c | (^b)&d
			t := bits.RotateLeft32(a, 5) + f + e + w[i&0xf] + _K0
			a, b, c, d, e = t, a, bits.RotateLeft32(b, 30), c, d

			// Store compression state for the collision detection.
			m1[i] = w[i&0xf]
		}
		for ; i < 20; i++ {
			// Store pre-step compression state for the collision detection.
			dig.cs[i] = [5]uint32{a, b, c, d, e}

			tmp := w[(i-3)&0xf] ^ w[(i-8)&0xf] ^ w[(i-14)&0xf] ^ w[(i)&0xf]
			w[i&0xf] = tmp<<1 | tmp>>(32-1)

			f := b&c | (^b)&d
			t := bits.RotateLeft32(a, 5) + f + e + w[i&0xf] + _K0
			a, b, c, d, e = t, a, bits.RotateLeft32(b, 30), c, d

			// Store compression state for the collision detection.
			m1[i] = w[i&0xf]
		}
		for ; i < 40; i++ {
			// Store pre-step compression state for the collision detection.
			dig.cs[i] = [5]uint32{a, b, c, d, e}

			tmp := w[(i-3)&0xf] ^ w[(i-8)&0xf] ^ w[(i-14)&0xf] ^ w[(i)&0xf]
			w[i&0xf] = tmp<<1 | tmp>>(32-1)

			f := b ^ c ^ d
			t := bits.RotateLeft32(a, 5) + f + e + w[i&0xf] + _K1
			a, b, c, d, e = t, a, bits.RotateLeft32(b, 30), c, d

			// Store compression state for the collision detection.
			m1[i] = w[i&0xf]
		}
		for ; i < 60; i++ {
			// Store pre-step compression state for the collision detection.
			dig.cs[i] = [5]uint32{a, b, c, d, e}

			tmp := w[(i-3)&0xf] ^ w[(i-8)&0xf] ^ w[(i-14)&0xf] ^ w[(i)&0xf]
			w[i&0xf] = tmp<<1 | tmp>>(32-1)

			f := ((b | c) & d) | (b & c)
			t := bits.RotateLeft32(a, 5) + f + e + w[i&0xf] + _K2
			a, b, c, d, e = t, a, bits.RotateLeft32(b, 30), c, d

			// Store compression state for the collision detection.
			m1[i] = w[i&0xf]
		}
		for ; i < 80; i++ {
			// Store pre-step compression state for the collision detection.
			dig.cs[i] = [5]uint32{a, b, c, d, e}

			tmp := w[(i-3)&0xf] ^ w[(i-8)&0xf] ^ w[(i-14)&0xf] ^ w[(i)&0xf]
			w[i&0xf] = tmp<<1 | tmp>>(32-1)

			f := b ^ c ^ d
			t := bits.RotateLeft32(a, 5) + f + e + w[i&0xf] + _K3
			a, b, c, d, e = t, a, bits.RotateLeft32(b, 30), c, d

			// Store compression state for the collision detection.
			m1[i] = w[i&0xf]
		}

		h0 += a
		h1 += b
		h2 += c
		h3 += d
		h4 += e

		if mask, err := ubc.CalculateDvMask(m1); err == nil && mask != 0 {
			dvs := ubc.SHA1_dvs()
			for i := 0; dvs[i].DvType != 0; i++ {
				if (mask & ((uint32)(1) << uint32(dvs[i].MaskB))) != 0 {
					for j := 0; j < msize; j++ {
						dig.m2[j] = m1[j] ^ dvs[i].Dm[j]
					}

					recompressionStep(dvs[i].TestT, &dig.ihv2, &dig.ihvtmp, dig.m2, dig.cs[dvs[i].TestT])

					if 0 == ((dig.ihvtmp[0] ^ h0) | (dig.ihvtmp[1] ^ h1) |
						(dig.ihvtmp[2] ^ h2) | (dig.ihvtmp[3] ^ h3) | (dig.ihvtmp[4] ^ h4)) {
						dig.col = true
						bcol = true
					}
				}
			}
		}

		// Collision attacks are thwarted by hashing a detected near-collision block 3 times.
		// Think of it as extending SHA-1 from 80-steps to 240-steps for such blocks:
		// 		The best collision attacks against SHA-1 have complexity about 2^60,
		// 		thus for 240-steps an immediate lower-bound for the best cryptanalytic attacks would be 2^180.
		// 		An attacker would be better off using a generic birthday search of complexity 2^80.
		if bcol {
			for j := 0; j < 2; j++ {
				a, b, c, d, e := h0, h1, h2, h3, h4

				i := 0
				for ; i < 20; i++ {
					f := b&c | (^b)&d
					t := bits.RotateLeft32(a, 5) + f + e + m1[i] + _K0
					a, b, c, d, e = t, a, bits.RotateLeft32(b, 30), c, d
				}
				for ; i < 40; i++ {
					f := b ^ c ^ d
					t := bits.RotateLeft32(a, 5) + f + e + m1[i] + _K1
					a, b, c, d, e = t, a, bits.RotateLeft32(b, 30), c, d
				}
				for ; i < 60; i++ {
					f := ((b | c) & d) | (b & c)
					t := bits.RotateLeft32(a, 5) + f + e + m1[i] + _K2
					a, b, c, d, e = t, a, bits.RotateLeft32(b, 30), c, d
				}
				for ; i < 80; i++ {
					f := b ^ c ^ d
					t := bits.RotateLeft32(a, 5) + f + e + m1[i] + _K3
					a, b, c, d, e = t, a, bits.RotateLeft32(b, 30), c, d
				}

				h0 += a
				h1 += b
				h2 += c
				h3 += d
				h4 += e
			}
		}

		p = p[chunk:]
	}

	dig.h[0], dig.h[1], dig.h[2], dig.h[3], dig.h[4] = h0, h1, h2, h3, h4
}

func recompressionStep(step int, ihvin, ihvout *[5]uint32, m2 [msize]uint32, state [5]uint32) {
	a, b, c, d, e := state[0], state[1], state[2], state[3], state[4]

	// Walk backwards from current step to undo previous compression.
	for i := 79; i >= 60; i-- {
		a, b, c, d, e = b, c, d, e, a
		if step > i {
			b = bits.RotateLeft32(b, -30)
			f := b ^ c ^ d
			e -= bits.RotateLeft32(a, 5) + f + _K3 + m2[i]
		}
	}
	for i := 59; i >= 40; i-- {
		a, b, c, d, e = b, c, d, e, a
		if step > i {
			b = bits.RotateLeft32(b, -30)
			f := ((b | c) & d) | (b & c)
			e -= bits.RotateLeft32(a, 5) + f + _K2 + m2[i]
		}
	}
	for i := 39; i >= 20; i-- {
		a, b, c, d, e = b, c, d, e, a
		if step > i {
			b = bits.RotateLeft32(b, -30)
			f := b ^ c ^ d
			e -= bits.RotateLeft32(a, 5) + f + _K1 + m2[i]
		}
	}
	for i := 19; i >= 0; i-- {
		a, b, c, d, e = b, c, d, e, a
		if step > i {
			b = bits.RotateLeft32(b, -30)
			f := b&c | (^b)&d
			e -= bits.RotateLeft32(a, 5) + f + _K0 + m2[i]
		}
	}

	ihvin[0] = a
	ihvin[1] = b
	ihvin[2] = c
	ihvin[3] = d
	ihvin[4] = e
	a = state[0]
	b = state[1]
	c = state[2]
	d = state[3]
	e = state[4]

	// Recompress blocks based on the current step.
	for i := 0; i < 20; i++ {
		if step <= i {
			f := b&c | (^b)&d
			t := bits.RotateLeft32(a, 5) + f + e + _K0 + m2[i]
			a, b, c, d, e = t, a, bits.RotateLeft32(b, 30), c, d
		}
	}
	for i := 20; i < 40; i++ {
		if step <= i {
			f := b ^ c ^ d
			t := bits.RotateLeft32(a, 5) + f + e + _K1 + m2[i]
			a, b, c, d, e = t, a, bits.RotateLeft32(b, 30), c, d
		}
	}
	for i := 40; i < 60; i++ {
		if step <= i {
			f := ((b | c) & d) | (b & c)
			t := bits.RotateLeft32(a, 5) + f + e + _K2 + m2[i]
			a, b, c, d, e = t, a, bits.RotateLeft32(b, 30), c, d
		}
	}
	for i := 60; i < 80; i++ {
		if step <= i {
			f := b ^ c ^ d
			t := bits.RotateLeft32(a, 5) + f + e + _K3 + m2[i]
			a, b, c, d, e = t, a, bits.RotateLeft32(b, 30), c, d
		}
	}

	ihvout[0] = ihvin[0] + a
	ihvout[1] = ihvin[1] + b
	ihvout[2] = ihvin[2] + c
	ihvout[3] = ihvin[3] + d
	ihvout[4] = ihvin[4] + e
}
