//go:build !noasm && gc && arm64 && !amd64
// +build !noasm,gc,arm64,!amd64

package sha1cd

import (
	shared "github.com/pjbgf/sha1cd/internal"
	"github.com/pjbgf/sha1cd/internal/cpu"
	"github.com/pjbgf/sha1cd/ubc"
)

var hasSHA1 = cpu.ARM64.HasSHA1

// blockARM64 hashes a single chunk of p into the current state in h.
// p must hold at least one whole chunk. Anything beyond the first chunk is
// ignored, as the collision detection the caller runs afterwards inspects m1
// and cs for one chunk only.
// Both m1 and cs are used to store intermediate results which are used by the collision detection logic.
//
//go:noescape
func blockARM64(h []uint32, p []byte, m1 []uint32, cs [][5]uint32)

func block(dig *digest, p []byte) {
	if forceGeneric || !hasSHA1 {
		blockGeneric(dig, p)
		return
	}

	m1 := [shared.Rounds]uint32{}
	cs := [shared.PreStepState][shared.WordBuffers]uint32{}

	for len(p) >= shared.Chunk {
		// The assembly code only supports processing a block at a time,
		// so adjust the chunk accordingly.
		chunk := p[:shared.Chunk]

		blockARM64(dig.h[:], chunk, m1[:], cs[:])

		// Assembly states need repair only when a disturbance vector survives.
		if mask := ubc.CalculateDvMask(&m1); mask != 0 {
			rectifyCompressionState(&m1, &cs)
			if checkCollision(&m1, &cs, &dig.h, mask) {
				dig.col = true

				blockARM64(dig.h[:], chunk, m1[:], cs[:])
				blockARM64(dig.h[:], chunk, m1[:], cs[:])
			}
		}

		p = p[shared.Chunk:]
	}
}
