// Package cpu detects the CPU features that sha1cd dispatches on.
//
// It covers only what the assembly implementations need, which keeps
// start up cheap and avoids an external dependency. Every flag is false
// where a feature cannot be detected safely, which selects the generic
// implementation.
package cpu

// X86 holds the features of the current amd64 CPU. All flags are false on
// other architectures.
var X86 struct {
	// HasAVX and HasAVX2 are set only when the OS also preserves the YMM
	// state, and HasAVX512F only when it preserves the ZMM and opmask state.
	HasAVX     bool
	HasAVX2    bool
	HasAVX512F bool
	HasBMI1    bool
	HasBMI2    bool
	HasSHA     bool
	HasSSSE3   bool
	HasSSE41   bool
}

// ARM64 holds the features of the current arm64 CPU. All flags are false on
// other architectures.
var ARM64 struct {
	HasSHA1 bool
}
