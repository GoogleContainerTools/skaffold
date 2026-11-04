//go:build !noasm && gc && amd64 && !arm64
// +build !noasm,gc,amd64,!arm64

package sha1cd

import (
	shared "github.com/pjbgf/sha1cd/internal"
	"github.com/pjbgf/sha1cd/internal/cpu"
	"github.com/pjbgf/sha1cd/ubc"
)

// hasSHANI reports whether blockAMD64 can run. It uses legacy SSE encodings
// only, so it does not need AVX and also runs on the Goldmont, Goldmont Plus
// and Tremont Atoms, which implement SHA-NI without AVX.
var hasSHANI = cpu.X86.HasSHA && cpu.X86.HasSSSE3 && cpu.X86.HasSSE41

// blockAMD64 hashes a single chunk of p into the current state in h.
// p must hold at least one whole chunk. Anything beyond the first chunk is
// ignored, as the collision detection the caller runs afterwards inspects m1
// and cs for one chunk only.
// Both m1 and cs are used to store intermediate results which are used by the collision detection logic.
//
//go:noescape
func blockAMD64(h []uint32, p []byte, m1 []uint32, cs [][5]uint32)

func block(dig *digest, p []byte) {
	switch {
	case forceGeneric:
		blockGeneric(dig, p)
	case hasSHANI:
		blockSHANI(dig, p)
	case hasAVX2:
		blockAVX2(dig, p)
	default:
		blockGeneric(dig, p)
	}
}

func blockSHANI(dig *digest, p []byte) {
	m1 := [shared.Rounds]uint32{}
	cs := [shared.PreStepState][shared.WordBuffers]uint32{}

	for len(p) >= shared.Chunk {
		// The assembly code only supports processing a block at a time,
		// so adjust the chunk accordingly.
		chunk := p[:shared.Chunk]

		blockAMD64(dig.h[:], chunk, m1[:], cs[:])
		// Assembly states need repair only when a disturbance vector survives.
		if mask := ubc.CalculateDvMask(&m1); mask != 0 {
			rectifyCompressionState(&m1, &cs)
			if checkCollision(&m1, &cs, &dig.h, mask) {
				dig.col = true

				blockAMD64(dig.h[:], chunk, m1[:], cs[:])
				blockAMD64(dig.h[:], chunk, m1[:], cs[:])
			}
		}

		p = p[shared.Chunk:]
	}
}
