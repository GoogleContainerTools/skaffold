package packfile

import (
	"bufio"
	"bytes"
	"crypto"
	"errors"
	"fmt"
	gohash "hash"
	"hash/crc32"
	"io"

	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/hash"
	"github.com/go-git/go-git/v5/utils/binary"
	"github.com/go-git/go-git/v5/utils/ioutil"
	"github.com/go-git/go-git/v5/utils/sync"
)

var (
	// ErrEmptyPackfile is returned by ReadHeader when no data is found in the packfile
	ErrEmptyPackfile = NewError("empty packfile")
	// ErrBadSignature is returned by ReadHeader when the signature in the packfile is incorrect.
	ErrBadSignature = NewError("malformed pack file signature")
	// ErrUnsupportedVersion is returned by ReadHeader when the packfile version is
	// different than VersionSupported.
	ErrUnsupportedVersion = NewError("unsupported packfile version")
	// ErrSeekNotSupported returned if seek is not support
	ErrSeekNotSupported = NewError("not seek support")
	// ErrMalformedPackFile is returned by the parser when the pack file is corrupted.
	ErrMalformedPackFile = errors.New("malformed PACK file")
	// ErrLengthOverflow is returned when a variable-length integer would not
	// fit into its accumulator because the input declares more continuation
	// bytes than the type can hold.
	ErrLengthOverflow = errors.New("variable-length integer overflow")
	// ErrInflatedSizeMismatch is returned when a packfile object inflates to
	// more bytes than the size declared in its object header. A well-formed
	// packfile never produces more data than the declared size; exceeding it
	// indicates a structurally invalid entry.
	ErrInflatedSizeMismatch = errors.New("packfile: inflated object exceeds declared size")
)

// boundedWriter passes writes through to w up to limit bytes total, then
// returns ErrInflatedSizeMismatch. It is used to enforce that a packfile
// object's inflated length does not exceed the size declared in its header.
type boundedWriter struct {
	w     io.Writer
	limit int64
	n     int64
}

// Write forwards p to the underlying writer while keeping the running total
// at or below limit. On overrun it forwards the legal prefix and reports
// the number of bytes actually consumed alongside ErrInflatedSizeMismatch,
// matching the contract in io.Writer. A write error from the underlying
// writer during overrun-handling is joined with ErrInflatedSizeMismatch so
// it is not silently dropped.
func (b *boundedWriter) Write(p []byte) (int, error) {
	if b.n+int64(len(p)) > b.limit {
		remain := int(b.limit - b.n)
		err := error(ErrInflatedSizeMismatch)
		if remain > 0 {
			n, werr := b.w.Write(p[:remain])
			b.n += int64(n)
			if werr != nil {
				err = errors.Join(ErrInflatedSizeMismatch, werr)
			}
			return n, err
		}
		return 0, err
	}
	n, err := b.w.Write(p)
	b.n += int64(n)
	return n, err
}

// boundedReadCloser wraps a ReadCloser and reports ErrInflatedSizeMismatch
// once more than limit bytes have been read. It is used by the on-demand
// object reader returned from FSObject.Reader so that a lazy Read of a
// packfile object cannot stream past its declared inflated size.
//
// The implementation builds on io.LimitedReader with the standard
// overrun-detection trick: request limit+1 bytes from the underlying so
// that the moment the sentinel byte materializes (LimitedReader.N drops
// to zero) we know the source produced more than limit bytes.
type boundedReadCloser struct {
	lr      io.LimitedReader
	closer  io.Closer
	overrun bool
}

// newBoundedReadCloser wraps rc so that the cumulative bytes returned from
// Read never exceed limit. The first call that would have returned a byte
// past limit instead returns ErrInflatedSizeMismatch; subsequent calls
// keep returning the same error. A negative limit is treated as zero, so
// the first byte produced by rc surfaces ErrInflatedSizeMismatch.
func newBoundedReadCloser(rc io.ReadCloser, limit int64) *boundedReadCloser {
	if limit < 0 {
		limit = 0
	}
	return &boundedReadCloser{
		lr:     io.LimitedReader{R: rc, N: limit + 1},
		closer: rc,
	}
}

// Read forwards Read up to the configured byte limit. When the underlying
// stream produces the limit+1 sentinel byte, the legal prefix is returned
// alongside ErrInflatedSizeMismatch; on subsequent calls only the error
// is returned.
func (b *boundedReadCloser) Read(p []byte) (int, error) {
	if b.overrun {
		return 0, ErrInflatedSizeMismatch
	}
	n, err := b.lr.Read(p)
	if b.lr.N == 0 {
		b.overrun = true
		return n - 1, ErrInflatedSizeMismatch
	}
	return n, err
}

// Close closes the underlying ReadCloser.
func (b *boundedReadCloser) Close() error { return b.closer.Close() }

// ObjectHeader contains the information related to the object, this information
// is collected from the previous bytes to the content of the object.
type ObjectHeader struct {
	Type            plumbing.ObjectType
	Offset          int64
	Length          int64
	Reference       plumbing.Hash
	OffsetReference int64
}

type Scanner struct {
	r          *scannerReader
	crc        gohash.Hash32
	packHasher hash.Hash

	// pendingObject is used to detect if an object has been read, or still
	// is waiting to be read
	pendingObject    *ObjectHeader
	version, objects uint32

	// lsSeekable says if this scanner can do Seek or not, to have a Scanner
	// seekable a r implementing io.Seeker is required
	IsSeekable bool
}

// NewScanner returns a new Scanner based on a reader, if the given reader
// implements io.ReadSeeker the Scanner will be also Seekable
func NewScanner(r io.Reader) *Scanner {
	_, ok := r.(io.ReadSeeker)

	crc := crc32.NewIEEE()
	hasher := hash.New(crypto.SHA1)
	return &Scanner{
		r:          newScannerReader(r, io.MultiWriter(crc, hasher)),
		crc:        crc,
		IsSeekable: ok,
		packHasher: hasher,
	}
}

func (s *Scanner) Reset(r io.Reader) {
	_, ok := r.(io.ReadSeeker)

	s.r.Reset(r)
	s.crc.Reset()
	s.packHasher.Reset()
	s.IsSeekable = ok
	s.pendingObject = nil
	s.version = 0
	s.objects = 0
}

// Header reads the whole packfile header (signature, version and object count).
// It returns the version and the object count and performs checks on the
// validity of the signature and the version fields.
func (s *Scanner) Header() (version, objects uint32, err error) {
	if s.version != 0 {
		return s.version, s.objects, nil
	}

	sig, err := s.readSignature()
	if err != nil {
		if err == io.EOF {
			err = ErrEmptyPackfile
		}

		return
	}

	if !s.isValidSignature(sig) {
		err = ErrBadSignature
		return
	}

	version, err = s.readVersion()
	s.version = version
	if err != nil {
		return
	}

	if !s.isSupportedVersion(version) {
		err = ErrUnsupportedVersion.AddDetails("%d", version)
		return
	}

	objects, err = s.readCount()
	s.objects = objects
	return
}

// readSignature reads a returns the signature field in the packfile.
func (s *Scanner) readSignature() ([]byte, error) {
	sig := make([]byte, 4)
	if _, err := io.ReadFull(s.r, sig); err != nil {
		return []byte{}, err
	}

	return sig, nil
}

// isValidSignature returns if sig is a valid packfile signature.
func (s *Scanner) isValidSignature(sig []byte) bool {
	return bytes.Equal(sig, signature)
}

// readVersion reads and returns the version field of a packfile.
func (s *Scanner) readVersion() (uint32, error) {
	return binary.ReadUint32(s.r)
}

// isSupportedVersion returns whether version v is supported by the parser.
// The current supported version is VersionSupported, defined above.
func (s *Scanner) isSupportedVersion(v uint32) bool {
	return v == VersionSupported
}

// readCount reads and returns the count of objects field of a packfile.
func (s *Scanner) readCount() (uint32, error) {
	return binary.ReadUint32(s.r)
}

// SeekObjectHeader seeks to specified offset and returns the ObjectHeader
// for the next object in the reader
func (s *Scanner) SeekObjectHeader(offset int64) (*ObjectHeader, error) {
	// if seeking we assume that you are not interested in the header
	if s.version == 0 {
		s.version = VersionSupported
	}

	if _, err := s.r.Seek(offset, io.SeekStart); err != nil {
		return nil, err
	}

	h, err := s.nextObjectHeader()
	if err != nil {
		return nil, err
	}

	h.Offset = offset
	return h, nil
}

// NextObjectHeader returns the ObjectHeader for the next object in the reader
func (s *Scanner) NextObjectHeader() (*ObjectHeader, error) {
	if err := s.doPending(); err != nil {
		return nil, err
	}

	offset, err := s.r.Seek(0, io.SeekCurrent)
	if err != nil {
		return nil, err
	}

	h, err := s.nextObjectHeader()
	if err != nil {
		return nil, err
	}

	h.Offset = offset
	return h, nil
}

// nextObjectHeader returns the ObjectHeader for the next object in the reader
// without the Offset field
func (s *Scanner) nextObjectHeader() (*ObjectHeader, error) {
	s.r.Flush()
	s.crc.Reset()

	h := &ObjectHeader{}
	s.pendingObject = h

	var err error
	h.Offset, err = s.r.Seek(0, io.SeekCurrent)
	if err != nil {
		return nil, err
	}

	h.Type, h.Length, err = s.readObjectTypeAndLength()
	if err != nil {
		return nil, err
	}

	switch h.Type {
	case plumbing.OFSDeltaObject:
		no, err := binary.ReadVariableWidthInt(s.r)
		if err != nil {
			return nil, err
		}

		// An OFS-delta references a base object that appears earlier
		// in the pack; the negative offset must be strictly positive
		// and not larger than the current object's offset.
		if no <= 0 || no > h.Offset {
			return nil, fmt.Errorf("%w: invalid OFS delta offset", ErrMalformedPackFile)
		}

		h.OffsetReference = h.Offset - no
	case plumbing.REFDeltaObject:
		var err error
		h.Reference, err = binary.ReadHash(s.r)
		if err != nil {
			return nil, err
		}
	}

	return h, nil
}

func (s *Scanner) doPending() error {
	if s.version == 0 {
		var err error
		s.version, s.objects, err = s.Header()
		if err != nil {
			return err
		}
	}

	return s.discardObjectIfNeeded()
}

func (s *Scanner) discardObjectIfNeeded() error {
	if s.pendingObject == nil {
		return nil
	}

	h := s.pendingObject
	n, _, err := s.NextObject(io.Discard)
	if err != nil {
		return err
	}

	if n != h.Length {
		return fmt.Errorf(
			"error discarding object, discarded %d, expected %d",
			n, h.Length,
		)
	}

	return nil
}

// ReadObjectTypeAndLength reads and returns the object type and the
// length field from an object entry in a packfile.
func (s *Scanner) readObjectTypeAndLength() (plumbing.ObjectType, int64, error) {
	t, c, err := s.readType()
	if err != nil {
		return t, 0, err
	}

	l, err := s.readLength(c)

	return t, l, err
}

func (s *Scanner) readType() (plumbing.ObjectType, byte, error) {
	var c byte
	var err error
	if c, err = s.r.ReadByte(); err != nil {
		return plumbing.ObjectType(0), 0, err
	}

	typ := parseType(c)

	return typ, c, nil
}

func parseType(b byte) plumbing.ObjectType {
	return plumbing.ObjectType((b & maskType) >> firstLengthBits)
}

// the length is codified in the last 4 bits of the first byte and in
// the last 7 bits of subsequent bytes.  Last byte has a 0 MSB.
func (s *Scanner) readLength(first byte) (int64, error) {
	length := int64(first & maskFirstLength)

	c := first
	shift := firstLengthBits
	var err error
	for c&maskContinue > 0 {
		// Mirrors unpack_object_header_buffer in canonical Git's
		// packfile.c: a continuation byte at shift > 64-7 cannot
		// contribute without overflowing an int64.
		if shift > 64-lengthBits {
			return 0, fmt.Errorf("%w: %w", ErrMalformedPackFile, ErrLengthOverflow)
		}

		if c, err = s.r.ReadByte(); err != nil {
			return 0, err
		}

		length += int64(c&maskLength) << shift
		shift += lengthBits
	}

	return length, nil
}

// NextObject writes the content of the next object into the reader, returns
// the number of bytes written, the CRC32 of the content and an error, if any.
//
// When a prior NextObjectHeader has stashed the object header in
// pendingObject, the inflated stream is bounded by the header's declared
// length and surfaces ErrInflatedSizeMismatch on overrun.
func (s *Scanner) NextObject(w io.Writer) (written int64, crc32 uint32, err error) {
	declaredSize := int64(-1)
	if s.pendingObject != nil {
		declaredSize = s.pendingObject.Length
	}
	s.pendingObject = nil
	written, err = s.copyObject(w, declaredSize)

	s.r.Flush()
	crc32 = s.crc.Sum32()
	s.crc.Reset()

	return
}

// ReadObject returns a reader for the object content and an error.
//
// When a prior NextObjectHeader has stashed the object header in
// pendingObject, the returned reader is bounded by the header's declared
// length so callers cannot stream past the declared inflated size; an
// overrun surfaces ErrInflatedSizeMismatch on the byte just past the
// limit.
func (s *Scanner) ReadObject() (io.ReadCloser, error) {
	declaredSize := int64(-1)
	if s.pendingObject != nil {
		declaredSize = s.pendingObject.Length
	}
	s.pendingObject = nil
	zr, err := sync.GetZlibReader(s.r)
	if err != nil {
		return nil, fmt.Errorf("zlib reset error: %s", err)
	}

	rc := ioutil.NewReadCloserWithCloser(zr.Reader, func() error {
		sync.PutZlibReader(zr)
		return nil
	})
	if declaredSize >= 0 {
		return newBoundedReadCloser(rc, declaredSize), nil
	}
	return rc, nil
}

// copyObject inflates a non-deltified object's zlib stream into w. When
// declaredSize is non-negative, the write sink is wrapped in a
// boundedWriter so an overrun surfaces ErrInflatedSizeMismatch instead
// of being silently appended.
func (s *Scanner) copyObject(w io.Writer, declaredSize int64) (n int64, err error) {
	zr, err := sync.GetZlibReader(s.r)
	defer sync.PutZlibReader(zr)

	if err != nil {
		return 0, fmt.Errorf("zlib reset error: %s", err)
	}

	defer ioutil.CheckClose(zr.Reader, &err)

	sink := w
	if declaredSize >= 0 {
		sink = &boundedWriter{w: w, limit: declaredSize}
	}

	buf := sync.GetByteSlice()
	n, err = io.CopyBuffer(sink, zr.Reader, *buf)
	sync.PutByteSlice(buf)
	return
}

// SeekFromStart sets a new offset from start, returns the old position before
// the change.
func (s *Scanner) SeekFromStart(offset int64) (previous int64, err error) {
	// if seeking we assume that you are not interested in the header
	if s.version == 0 {
		s.version = VersionSupported
	}

	previous, err = s.r.Seek(0, io.SeekCurrent)
	if err != nil {
		return -1, err
	}

	_, err = s.r.Seek(offset, io.SeekStart)
	return previous, err
}

// Checksum returns the checksum of the packfile
func (s *Scanner) Checksum() (plumbing.Hash, error) {
	err := s.discardObjectIfNeeded()
	if err != nil {
		return plumbing.ZeroHash, err
	}

	s.r.Flush()
	actual := plumbing.Hash(s.packHasher.Sum(nil))
	packChecksum, err := binary.ReadHash(s.r)
	if err != nil {
		return plumbing.ZeroHash, err
	}

	if actual != packChecksum {
		return plumbing.ZeroHash, fmt.Errorf("%w: checksum mismatch: %q instead of %q", ErrMalformedPackFile, packChecksum, actual)
	}

	return packChecksum, nil
}

// Close reads the reader until io.EOF
func (s *Scanner) Close() error {
	buf := sync.GetByteSlice()
	_, err := io.CopyBuffer(io.Discard, s.r, *buf)
	sync.PutByteSlice(buf)

	return err
}

// Flush is a no-op (deprecated)
func (s *Scanner) Flush() error {
	return nil
}

// scannerReader has the following characteristics:
//   - Provides an io.SeekReader impl for bufio.Reader, when the underlying
//     reader supports it.
//   - Keeps track of the current read position, for when the underlying reader
//     isn't an io.SeekReader, but we still want to know the current offset.
//   - Writes to the hash writer what it reads, with the aid of a smaller buffer.
//     The buffer helps avoid a performance penalty for performing small writes
//     to the crc32 hash writer.
type scannerReader struct {
	reader io.Reader
	writer io.Writer
	rbuf   *bufio.Reader
	wbuf   *bufio.Writer
	offset int64
}

func newScannerReader(r io.Reader, w io.Writer) *scannerReader {
	sr := &scannerReader{
		rbuf:   bufio.NewReader(nil),
		wbuf:   bufio.NewWriterSize(nil, 64),
		writer: w,
	}
	sr.Reset(r)

	return sr
}

func (r *scannerReader) Reset(reader io.Reader) {
	r.reader = reader
	r.rbuf.Reset(r.reader)
	r.wbuf.Reset(r.writer)

	r.offset = 0
	if seeker, ok := r.reader.(io.ReadSeeker); ok {
		r.offset, _ = seeker.Seek(0, io.SeekCurrent)
	}
}

func (r *scannerReader) Read(p []byte) (n int, err error) {
	n, err = r.rbuf.Read(p)

	r.offset += int64(n)
	if _, err := r.wbuf.Write(p[:n]); err != nil {
		return n, err
	}
	return
}

func (r *scannerReader) ReadByte() (b byte, err error) {
	b, err = r.rbuf.ReadByte()
	if err == nil {
		r.offset++
		return b, r.wbuf.WriteByte(b)
	}
	return
}

func (r *scannerReader) Flush() error {
	return r.wbuf.Flush()
}

// Seek seeks to a location. If the underlying reader is not an io.ReadSeeker,
// then only whence=io.SeekCurrent is supported, any other operation fails.
func (r *scannerReader) Seek(offset int64, whence int) (int64, error) {
	var err error

	if seeker, ok := r.reader.(io.ReadSeeker); !ok {
		if whence != io.SeekCurrent || offset != 0 {
			return -1, ErrSeekNotSupported
		}
	} else {
		if whence == io.SeekCurrent && offset == 0 {
			return r.offset, nil
		}

		r.offset, err = seeker.Seek(offset, whence)
		r.rbuf.Reset(r.reader)
	}

	return r.offset, err
}
