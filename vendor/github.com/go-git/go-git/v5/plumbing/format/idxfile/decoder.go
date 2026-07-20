package idxfile

import (
	"bytes"
	"crypto"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"io/fs"

	"github.com/go-git/go-git/v5/plumbing/hash"
	"github.com/go-git/go-git/v5/utils/binary"
)

var (
	// ErrUnsupportedVersion is returned by Decode when the idx file version
	// is not supported.
	ErrUnsupportedVersion = errors.New("unsupported version")
	// ErrMalformedIdxFile is returned by Decode when the idx file is corrupted.
	ErrMalformedIdxFile = errors.New("malformed IDX file")
)

const (
	fanout         = 256
	objectIDLength = hash.Size
)

// Byte sizes of the idx v2 layout elements, used by the size formula
// in [validateIdxV2Size]. See [gitformat-pack] for the canonical
// layout.
//
// [gitformat-pack]: https://git-scm.com/docs/gitformat-pack
const (
	headerLen     = 8          // magic + version
	fanoutLen     = fanout * 4 // uint32 per bucket
	crc32Len      = 4          // CRC32 per object
	offset32Len   = 4          // 32-bit offset per object
	offset64Len   = 8          // 64-bit overflow offset
	trailerHashes = 2          // pack checksum + idx checksum, each hashsz
)

// statInput is the optional shape the [Decoder] probes for at the
// start of [Decoder.Decode] to learn the on-disk length of the idx
// blob, which it uses to validate the canonical-Git size formula
// before any allocations driven by the fanout table. Callers that
// pass an [*os.File] or a `billy.File` backed by an `*os.File`
// (the production call sites in `storage/filesystem`) satisfy it
// directly; arbitrary [io.Reader]s do not, and decode for them
// retains the pre-existing behaviour of erroring out at the
// truncated-payload boundary instead.
//
// The interface is intentionally unexported so the public
// [NewDecoder] signature stays compatible with v5.
type statInput interface {
	Stat() (fs.FileInfo, error)
}

// Decoder reads and decodes idx files from an input stream.
type Decoder struct {
	io.Reader
	src io.Reader
	h   hash.Hash
}

// NewDecoder builds a new idx stream decoder, that reads from r.
func NewDecoder(r io.Reader) *Decoder {
	h := hash.New(crypto.SHA1)
	tr := io.TeeReader(r, h)
	return &Decoder{tr, r, h}
}

// Decode reads from the stream and decode the content into the MemoryIndex struct.
func (d *Decoder) Decode(idx *MemoryIndex) error {
	idxSize := int64(-1)
	if in, ok := d.src.(statInput); ok {
		fi, err := in.Stat()
		if err != nil {
			return fmt.Errorf("%w: stat input: %w", ErrMalformedIdxFile, err)
		}
		idxSize = fi.Size()
	}

	if err := validateHeader(d); err != nil {
		return err
	}

	headerFlow := []func(*MemoryIndex, io.Reader) error{
		readVersion,
		readFanout,
	}
	for _, f := range headerFlow {
		if err := f(idx, d); err != nil {
			return err
		}
	}

	if idxSize >= 0 {
		if err := validateIdxV2Size(idx, idxSize); err != nil {
			return err
		}
	}

	bodyFlow := []func(*MemoryIndex, io.Reader) error{
		readObjectNames,
		readCRC32,
		readOffsets,
		readPackChecksum,
	}
	for _, f := range bodyFlow {
		if err := f(idx, d); err != nil {
			return err
		}
	}

	actual := d.h.Sum(nil)
	if err := readIdxChecksum(idx, d); err != nil {
		return err
	}

	if !bytes.Equal(actual, idx.IdxChecksum[:]) {
		return fmt.Errorf("%w: checksum mismatch: %q instead of %q",
			ErrMalformedIdxFile, hex.EncodeToString(idx.IdxChecksum[:]), hex.EncodeToString(actual))
	}

	return nil
}

func validateHeader(r io.Reader) error {
	h := make([]byte, 4)
	if _, err := io.ReadFull(r, h); err != nil {
		return err
	}

	if !bytes.Equal(h, idxHeader) {
		return ErrMalformedIdxFile
	}

	return nil
}

func readVersion(idx *MemoryIndex, r io.Reader) error {
	v, err := binary.ReadUint32(r)
	if err != nil {
		return err
	}

	if v != VersionSupported {
		return fmt.Errorf("%w: v%d", ErrUnsupportedVersion, v)
	}

	idx.Version = v
	return nil
}

func readFanout(idx *MemoryIndex, r io.Reader) error {
	for k := 0; k < fanout; k++ {
		n, err := binary.ReadUint32(r)
		if err != nil {
			return err
		}

		if k > 0 && n < idx.Fanout[k-1] {
			return fmt.Errorf("%w: fanout table is not monotonically non-decreasing at entry %d", ErrMalformedIdxFile, k)
		}

		idx.Fanout[k] = n
		idx.FanoutMapping[k] = noMapping
	}

	return nil
}

func readObjectNames(idx *MemoryIndex, r io.Reader) error {
	for k := 0; k < fanout; k++ {
		var buckets uint32
		if k == 0 {
			buckets = idx.Fanout[k]
		} else {
			buckets = idx.Fanout[k] - idx.Fanout[k-1]
		}

		if buckets == 0 {
			continue
		}

		idx.FanoutMapping[k] = len(idx.Names)

		nameLen := int(buckets * objectIDLength)
		bin := make([]byte, nameLen)
		if _, err := io.ReadFull(r, bin); err != nil {
			return err
		}

		idx.Names = append(idx.Names, bin)
		idx.Offset32 = append(idx.Offset32, make([]byte, buckets*4))
		idx.CRC32 = append(idx.CRC32, make([]byte, buckets*4))
	}

	return nil
}

func readCRC32(idx *MemoryIndex, r io.Reader) error {
	for k := 0; k < fanout; k++ {
		if pos := idx.FanoutMapping[k]; pos != noMapping {
			if _, err := io.ReadFull(r, idx.CRC32[pos]); err != nil {
				return err
			}
		}
	}

	return nil
}

func readOffsets(idx *MemoryIndex, r io.Reader) error {
	var o64cnt int64
	for k := 0; k < fanout; k++ {
		if pos := idx.FanoutMapping[k]; pos != noMapping {
			if _, err := io.ReadFull(r, idx.Offset32[pos]); err != nil {
				return err
			}

			for p := 0; p < len(idx.Offset32[pos]); p += 4 {
				if idx.Offset32[pos][p]&(byte(1)<<7) > 0 {
					o64cnt++
				}
			}
		}
	}

	if o64cnt > 0 {
		idx.Offset64 = make([]byte, o64cnt*8)
		if _, err := io.ReadFull(r, idx.Offset64); err != nil {
			return err
		}
	}

	return nil
}

func readPackChecksum(idx *MemoryIndex, r io.Reader) error {
	if _, err := io.ReadFull(r, idx.PackfileChecksum[:]); err != nil {
		return err
	}

	return nil
}

func readIdxChecksum(idx *MemoryIndex, r io.Reader) error {
	if _, err := io.ReadFull(r, idx.IdxChecksum[:]); err != nil {
		return err
	}

	return nil
}

// validateIdxV2Size enforces the size formula used by canonical Git
// load_idx for idx v2 files: the on-disk length must lie within
// [minSize, maxSize] where
//
//	perObject = hashsz + crc32Len + offset32Len
//	minSize   = headerLen + fanoutLen + trailerHashes*hashsz + nr*perObject
//	maxSize   = minSize + (nr-1)*offset64Len     when nr > 0
//
// with nr taken from the last fanout entry and hashsz fixed at
// [objectIDLength] (SHA-1 in v5). Multiplications use a self-checking
// overflow guard so inputs whose claimed object count overflows the
// formula are rejected rather than wrapping into a smaller value.
func validateIdxV2Size(idx *MemoryIndex, idxSize int64) error {
	nr := int64(idx.Fanout[fanout-1])
	hashsz := int64(objectIDLength)

	minSize := minIdxV2Size(nr, hashsz)
	maxSize := maxIdxV2Size(nr, hashsz)
	if minSize < 0 || maxSize < 0 {
		return fmt.Errorf("%w: object count %d is inconsistent with file size", ErrMalformedIdxFile, nr)
	}

	if idxSize < minSize || idxSize > maxSize {
		return fmt.Errorf("%w: file size %d is inconsistent with object count %d", ErrMalformedIdxFile, idxSize, nr)
	}
	return nil
}

// minIdxV2Size returns the minimum on-disk size of an idx v2 file
// holding nr objects with the given hash size, mirroring the
// computation in canonical Git load_idx. Returns -1 when any
// intermediate multiplication or addition would overflow int64.
func minIdxV2Size(nr, hashsz int64) int64 {
	perObject := hashsz + crc32Len + offset32Len
	fixed := int64(headerLen+fanoutLen) + trailerHashes*hashsz

	objects, ok := mulInt64(nr, perObject)
	if !ok {
		return -1
	}
	sum, ok := addInt64(fixed, objects)
	if !ok {
		return -1
	}
	return sum
}

// maxIdxV2Size returns the maximum on-disk size of an idx v2 file
// holding nr objects with the given hash size, mirroring the
// computation in canonical Git load_idx. Returns -1 on overflow.
func maxIdxV2Size(nr, hashsz int64) int64 {
	minSize := minIdxV2Size(nr, hashsz)
	if minSize < 0 {
		return -1
	}
	if nr == 0 {
		return minSize
	}
	overflow, ok := mulInt64(nr-1, offset64Len)
	if !ok {
		return -1
	}
	sum, ok := addInt64(minSize, overflow)
	if !ok {
		return -1
	}
	return sum
}

// mulInt64 returns a*b and whether the result fits in an int64 without
// overflow. Negative operands or overflow yield ok=false. The overflow
// check uses the standard self-inverse identity: a*b/b == a only when
// the multiplication did not wrap.
func mulInt64(a, b int64) (int64, bool) {
	if a < 0 || b < 0 {
		return 0, false
	}
	if a == 0 || b == 0 {
		return 0, true
	}
	c := a * b
	if c/b != a {
		return 0, false
	}
	return c, true
}

// addInt64 returns a+b and whether the result fits in an int64 without
// overflow. Negative operands or overflow yield ok=false.
func addInt64(a, b int64) (int64, bool) {
	if a < 0 || b < 0 {
		return 0, false
	}
	c := a + b
	if c < a {
		return 0, false
	}
	return c, true
}
