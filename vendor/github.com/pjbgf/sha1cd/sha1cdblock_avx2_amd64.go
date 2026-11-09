//go:build !noasm && gc && amd64

package sha1cd

import (
	shared "github.com/pjbgf/sha1cd/internal"
	"github.com/pjbgf/sha1cd/internal/cpu"
	"github.com/pjbgf/sha1cd/ubc"
)

// hasAVX2 reports whether blockAVX2 can run. It is the fallback for CPUs
// without SHA-NI, such as Intel's big cores before Ice Lake.
var hasAVX2 = cpu.X86.HasAVX2 && cpu.X86.HasBMI1 && cpu.X86.HasBMI2

// scheduleAVX2 expands the message schedules of the blocks at pa and pb, which
// may be the same, into m1a and m1b.
//
//go:noescape
func scheduleAVX2(pa, pb *byte, m1a, m1b *[shared.Rounds]uint32)

// roundsBMI2 compresses the block whose schedule is in m1 into h, and stores
// the states before steps 0, 58 and 65 into cs.
//
//go:noescape
func roundsBMI2(h *[shared.WordBuffers]uint32, m1 *[shared.Rounds]uint32,
	cs *[shared.PreStepState][shared.WordBuffers]uint32)

func blockAVX2(dig *digest, p []byte) {
	var m1 [2][shared.Rounds]uint32
	cs := [shared.PreStepState][shared.WordBuffers]uint32{}

	for len(p) >= shared.Chunk {
		// The schedules do not depend on the chaining state, so expand two
		// blocks at once. The rounds must still run one block at a time, as a
		// detected collision changes the state the next block starts from.
		n, pb := 1, p
		if len(p) >= 2*shared.Chunk {
			n, pb = 2, p[shared.Chunk:]
		}
		scheduleAVX2(&p[0], &pb[0], &m1[0], &m1[1])

		for j := 0; j < n; j++ {
			w := &m1[j]
			roundsBMI2(&dig.h, w, &cs)
			if mask := ubc.CalculateDvMask(w); mask != 0 && checkCollision(w, &cs, &dig.h, mask) {
				dig.col = true

				roundsBMI2(&dig.h, w, &cs)
				roundsBMI2(&dig.h, w, &cs)
			}
		}

		p = p[n*shared.Chunk:]
	}
}
