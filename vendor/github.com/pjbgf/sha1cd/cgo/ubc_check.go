package cgo

// #include <ubc_check.h>
// #include <stdlib.h>
//
// uint32_t check(const uint32_t W[80])
// {
//	 uint32_t ubc_dv_mask[DVMASKSIZE] = {(uint32_t)(0xFFFFFFFF)};
//   ubc_check(W, ubc_dv_mask);
//   return ubc_dv_mask[0];
// }
import "C"
import (
	"fmt"
	"unsafe"
)

// CalculateDvMask takes as input an expanded message block and verifies the unavoidable
// bitconditions for all listed DVs. It returns a dvmask where each bit belonging to a DV
// is set if all unavoidable bitconditions for that DV have been met.
// Thus, one needs to do the recompression check for each DV that has its bit set.
func CalculateDvMask(W []uint32) (uint32, error) {
	if len(W) < 80 {
		return 0, fmt.Errorf("invalid input: len(W) must be 80, was %d", len(W))
	}

	return uint32(C.check((*C.uint32_t)(unsafe.Pointer(&W[0])))), nil
}
