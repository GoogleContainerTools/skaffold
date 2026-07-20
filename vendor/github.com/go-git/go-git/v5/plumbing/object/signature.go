package object

import (
	"bytes"
	"io"

	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/utils/ioutil"
	"github.com/go-git/go-git/v5/utils/sync"
)

const (
	signatureTypeUnknown signatureType = iota
	signatureTypeOpenPGP
	signatureTypeX509
	signatureTypeSSH
)

var (
	// openPGPSignatureFormat is the format of an OpenPGP signature.
	openPGPSignatureFormat = signatureFormat{
		[]byte("-----BEGIN PGP SIGNATURE-----"),
		[]byte("-----BEGIN PGP MESSAGE-----"),
	}
	// x509SignatureFormat is the format of an X509 signature, which is
	// a PKCS#7 (S/MIME) signature.
	x509SignatureFormat = signatureFormat{
		[]byte("-----BEGIN CERTIFICATE-----"),
		[]byte("-----BEGIN SIGNED MESSAGE-----"),
	}

	// sshSignatureFormat is the format of an SSH signature.
	sshSignatureFormat = signatureFormat{
		[]byte("-----BEGIN SSH SIGNATURE-----"),
	}
)

var (
	// knownSignatureFormats is a map of known signature formats, indexed by
	// their signatureType.
	knownSignatureFormats = map[signatureType]signatureFormat{
		signatureTypeOpenPGP: openPGPSignatureFormat,
		signatureTypeX509:    x509SignatureFormat,
		signatureTypeSSH:     sshSignatureFormat,
	}
)

// signatureType represents the type of the signature.
type signatureType int8

// signatureFormat represents the beginning of a signature.
type signatureFormat [][]byte

// typeForSignature returns the type of the signature based on its format.
func typeForSignature(b []byte) signatureType {
	for t, i := range knownSignatureFormats {
		for _, begin := range i {
			if bytes.HasPrefix(b, begin) {
				return t
			}
		}
	}
	return signatureTypeUnknown
}

// parseSignedBytes returns the position of the last signature block found in
// the given bytes. If no signature block is found, it returns -1.
//
// When multiple signature blocks are found, the position of the last one is
// returned. Any tailing bytes after this signature block start should be
// considered part of the signature.
//
// Given this, it would be safe to use the returned position to split the bytes
// into two parts: the first part containing the message, the second part
// containing the signature.
//
// Example:
//
//	message := []byte(`Message with signature
//
//	-----BEGIN SSH SIGNATURE-----
//	...`)
//
//	var signature string
//	if pos, _ := parseSignedBytes(message); pos != -1 {
//		signature = string(message[pos:])
//		message = message[:pos]
//	}
//
// This logic is on par with git's gpg-interface.c:parse_signed_buffer().
// https://github.com/git/git/blob/7c2ef319c52c4997256f5807564523dfd4acdfc7/gpg-interface.c#L668
func parseSignedBytes(b []byte) (int, signatureType) {
	var n, match = 0, -1
	var t signatureType
	for n < len(b) {
		var i = b[n:]
		if st := typeForSignature(i); st != signatureTypeUnknown {
			match = n
			t = st
		}
		if eol := bytes.IndexByte(i, '\n'); eol >= 0 {
			n += eol + 1
			continue
		}
		// If we reach this point, we've reached the end.
		break
	}
	return match, t
}

// countSignatureBlocks reports how many distinct armored signature blocks
// start at a line boundary in b. Used by verification paths to reject
// multi-signature payloads, matching upstream's check in gpg-interface.c
// where parse_gpg_output bails out the first time it sees a second
// exclusive status line (a second GOODSIG/BADSIG/etc.).
func countSignatureBlocks(b []byte) int {
	n, count := 0, 0
	for n < len(b) {
		i := b[n:]
		if typeForSignature(i) != signatureTypeUnknown {
			count++
		}
		if eol := bytes.IndexByte(i, '\n'); eol >= 0 {
			n += eol + 1
			continue
		}
		break
	}
	return count
}

// isSignatureHeader reports whether line is a canonical "gpgsig "/
// "gpgsig-sha256 " header line. Other "gpgsig"-prefixed extra headers
// are intentionally not matched.
func isSignatureHeader(line []byte) bool {
	return bytes.HasPrefix(line, []byte(headerpgp+" ")) ||
		bytes.HasPrefix(line, []byte(headerpgp256+" "))
}

// stripObjectSignatures streams src into dst, producing the byte sequence
// over which a PGP/GPG signature is computed:
//
//   - Canonical "gpgsig" and "gpgsig-sha256" headers (and their
//     continuation lines) are dropped, mirroring upstream's
//     remove_signature in commit.c.
//   - For tag objects, the inline trailing PGP signature is additionally
//     truncated, mirroring upstream's parse_signature in gpg-interface.c
//     used by gpg_verify_tag.
//
// The returned object's type is set to objType. Used by both
// Commit.EncodeWithoutSignature and Tag.EncodeWithoutSignature to
// reproduce the exact bytes the signature was computed over.
func stripObjectSignatures(dst, src plumbing.EncodedObject, objType plumbing.ObjectType) (err error) {
	dst.SetType(objType)

	r, err := src.Reader()
	if err != nil {
		return err
	}
	defer ioutil.CheckClose(r, &err)

	var input io.Reader = r
	if objType == plumbing.TagObject {
		raw, err := io.ReadAll(r)
		if err != nil {
			return err
		}
		if sm, _ := parseSignedBytes(raw); sm >= 0 {
			raw = raw[:sm]
		}
		input = bytes.NewReader(raw)
	}

	w, err := dst.Writer()
	if err != nil {
		return err
	}
	defer ioutil.CheckClose(w, &err)

	return stripHeaderSignatures(w, input)
}

// stripHeaderSignatures copies r to w, dropping canonical signature header
// lines (gpgsig and gpgsig-sha256) and their continuation lines. Lines
// past the blank line that closes the header block are copied verbatim.
func stripHeaderSignatures(w io.Writer, r io.Reader) error {
	br := sync.GetBufioReader(r)
	defer sync.PutBufioReader(br)

	var inBody, skipping bool
	for {
		line, rerr := br.ReadBytes('\n')
		if rerr != nil && rerr != io.EOF {
			return rerr
		}

		write := true
		if !inBody {
			switch {
			case skipping && len(line) > 0 && line[0] == ' ':
				write = false
			case isSignatureHeader(line):
				skipping = true
				write = false
			case len(line) == 1 && line[0] == '\n':
				skipping = false
				inBody = true
			default:
				skipping = false
			}
		}

		if write && len(line) > 0 {
			if _, werr := w.Write(line); werr != nil {
				return werr
			}
		}
		if rerr == io.EOF {
			return nil
		}
	}
}
