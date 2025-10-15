//go:build !noasm && gc && amd64 && !arm64
// +build !noasm,gc,amd64,!arm64

package sha1cd

import (
	"runtime"

	"github.com/klauspost/cpuid/v2"
	shared "github.com/pjbgf/sha1cd/internal"
)

var hasSHANI = (runtime.GOARCH == "amd64" &&
	cpuid.CPU.Supports(cpuid.AVX) &&
	cpuid.CPU.Supports(cpuid.SHA) &&
	cpuid.CPU.Supports(cpuid.SSE3) &&
	cpuid.CPU.Supports(cpuid.SSE4))

// blockAMD64 hashes the message p into the current state in h.
// Both m1 and cs are used to store intermediate results which are used by the collision detection logic.
//
//go:noescape
func blockAMD64(h []uint32, p []byte, m1 []uint32, cs [][5]uint32)

func block(dig *digest, p []byte) {
	if forceGeneric || !hasSHANI {
		blockGeneric(dig, p)
		return
	}

	m1 := [shared.Rounds]uint32{}
	cs := [shared.PreStepState][shared.WordBuffers]uint32{}

	for len(p) >= shared.Chunk {
		// The assembly code only supports processing a block at a time,
		// so adjust the chunk accordingly.
		chunk := p[:shared.Chunk]

		blockAMD64(dig.h[:], chunk, m1[:], cs[:])
		rectifyCompressionState(m1, &cs)

		col := checkCollision(m1, cs, dig.h)
		if col {
			dig.col = true

			blockAMD64(dig.h[:], chunk, m1[:], cs[:])
			blockAMD64(dig.h[:], chunk, m1[:], cs[:])
		}

		p = p[shared.Chunk:]
	}
}
