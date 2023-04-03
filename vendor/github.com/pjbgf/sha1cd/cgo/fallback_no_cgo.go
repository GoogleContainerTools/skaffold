//go:build !cgo
// +build !cgo

package cgo

import (
	"hash"

	"github.com/pjbgf/sha1cd"
	"github.com/pjbgf/sha1cd/ubc"
)

// CalculateDvMask falls back to github.com/pjbgf/sha1cd/ubc implementation
// due to CGO being disabled at compilation time.
func CalculateDvMask(W []uint32) (uint32, error) {
	return ubc.CalculateDvMask(W)
}

// CalculateDvMask falls back to github.com/pjbgf/sha1cd implementation
// due to CGO being disabled at compilation time.
func New() hash.Hash {
	return sha1cd.New()
}

// CalculateDvMask falls back to github.com/pjbgf/sha1cd implementation
// due to CGO being disabled at compilation time.
func Sum(data []byte) ([]byte, bool) {
	d := sha1cd.New().(sha1cd.CollisionResistantHash)
	d.Write(data)

	return d.CollisionResistantSum(nil)
}
