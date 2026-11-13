package trees

import (
	"errors"
	"fmt"
	"unicode"
	"unicode/utf8"
)

// MaxSize is the largest size of tree that can exist, per Section 5.2 of
// draft-ietf-plants-merkle-tree-certs-07:
//
// > Each issuance log describes an append-only sequence of at most 2^48-1 entries
//
// The max size is 2^48-1, rather than 2^48 as one might expect, because a
// subtree is represented by the half-open interval [start, end), meaning that
// `end` is one greater than the largest entry index in the subtree. If the max
// size of a tree were 2^48, the last entry in that tree would have an entry
// index of 2^48-1, and a subtree containing that last entry would be described
// by the pair [xx, 2^48). But the value `end` must fit in 48 bits, and therefore
// has a maximum value of 2^48-1. Thus we conclude that the maximum entry index
// must be 2^48-2, and the maximum tree size is 2^48-1.
const MaxSize = 1<<48 - 1

// CheckName returns an error if name does not follow the cosigner name
// construction in https://c2sp.org/tlog-cosignature#cosigners, which log
// origins also follow: a non-empty UTF-8 string of at most 255 bytes with no
// Unicode spaces, plus signs, or control characters below U+0020.
func CheckName(name string) error {
	if name == "" {
		return errors.New("empty")
	}
	if len(name) > 255 {
		return fmt.Errorf("%d bytes, want at most 255", len(name))
	}
	if !utf8.ValidString(name) {
		return errors.New("not valid UTF-8")
	}
	for _, r := range name {
		if unicode.IsSpace(r) || r == '+' || r < 0x20 {
			return fmt.Errorf("contains %q", r)
		}
	}
	return nil
}
