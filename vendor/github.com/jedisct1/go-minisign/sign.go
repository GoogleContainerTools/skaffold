package minisign

import (
	"bytes"
	"crypto/ed25519"
	"crypto/subtle"
	"encoding/base64"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"golang.org/x/crypto/blake2b"
	"golang.org/x/crypto/scrypt"
)

const (
	sigAlgEd            = "Ed"
	kdfAlgScrypt        = "Sc"
	chkAlgBlake2b       = "B2"
	commentPrefix       = "untrusted comment: "
	trustedPrefix       = "trusted comment: "
	defaultSigUntrusted = "signature from minisign secret key"
	secretKeyLen        = 158
	streamLen           = 104

	// trustedCommentMaxLen mirrors TRUSTEDCOMMENTMAXBYTES in the C
	// reference: any longer line cannot be read back by minisign(1).
	trustedCommentMaxLen = 8192 - len(trustedPrefix)
)

// PrivateKey is a minisign Ed25519 secret key. It may be stored in encrypted
// form; call Decrypt with the passphrase before signing in that case.
type PrivateKey struct {
	UntrustedComment   string
	SignatureAlgorithm [2]byte
	KDFAlgorithm       [2]byte
	ChecksumAlgorithm  [2]byte
	KDFSalt            [32]byte
	KDFOpsLimit        uint64
	KDFMemLimit        uint64
	KeyId              [8]byte
	SecretKey          [64]byte
	Checksum           [32]byte
}

// IsEncrypted reports whether the key material is still encrypted under a
// passphrase. Calling Sign on an encrypted key returns an error.
func (sk *PrivateKey) IsEncrypted() bool {
	return sk.KDFAlgorithm != [2]byte{0, 0}
}

// NewPrivateKey parses a single base64-encoded private key payload.
func NewPrivateKey(in string) (PrivateKey, error) {
	var sk PrivateKey
	bin, err := base64.StdEncoding.DecodeString(trimCarriageReturn(in))
	if err != nil || len(bin) != secretKeyLen {
		return sk, errors.New("Invalid encoded secret key")
	}
	copy(sk.SignatureAlgorithm[:], bin[0:2])
	copy(sk.KDFAlgorithm[:], bin[2:4])
	copy(sk.ChecksumAlgorithm[:], bin[4:6])
	copy(sk.KDFSalt[:], bin[6:38])
	sk.KDFOpsLimit = binary.LittleEndian.Uint64(bin[38:46])
	sk.KDFMemLimit = binary.LittleEndian.Uint64(bin[46:54])
	copy(sk.KeyId[:], bin[54:62])
	copy(sk.SecretKey[:], bin[62:126])
	copy(sk.Checksum[:], bin[126:158])
	if string(sk.SignatureAlgorithm[:]) != sigAlgEd {
		return sk, errors.New("Unsupported signature algorithm")
	}
	if string(sk.ChecksumAlgorithm[:]) != chkAlgBlake2b {
		return sk, errors.New("Unsupported checksum algorithm")
	}
	return sk, nil
}

// DecodePrivateKey parses a full minisign secret key file (comment line +
// base64 payload).
func DecodePrivateKey(in string) (PrivateKey, error) {
	lines := strings.SplitN(in, "\n", 2)
	if len(lines) < 2 {
		return PrivateKey{}, errors.New("Incomplete encoded secret key")
	}
	sk, err := NewPrivateKey(lines[1])
	if err != nil {
		return sk, err
	}
	sk.UntrustedComment = trimCarriageReturn(lines[0])
	return sk, nil
}

// NewPrivateKeyFromFile reads and parses a minisign secret key file.
func NewPrivateKeyFromFile(file string) (PrivateKey, error) {
	bin, err := os.ReadFile(file)
	if err != nil {
		return PrivateKey{}, err
	}
	return DecodePrivateKey(string(bin))
}

// scryptParamsFromLimits ports libsodium's pwhash_scryptsalsa208sha256
// pickparams logic so that scrypt parameters match what the minisign C
// reference implementation uses for the same opslimit and memlimit.
func scryptParamsFromLimits(opslimit, memlimit uint64) (N, r, p int, err error) {
	if opslimit < 32768 {
		opslimit = 32768
	}
	r = 8
	pick := func(maxN uint64) int {
		ln := 1
		for ln < 63 && uint64(1)<<ln <= maxN/2 {
			ln++
		}
		return ln
	}
	var ln int
	if opslimit < memlimit/32 {
		ln = pick(opslimit / uint64(r*4))
		p = 1
	} else {
		ln = pick(memlimit / uint64(r*128))
		maxrp := (opslimit / 4) / (uint64(1) << ln)
		if maxrp > 0x3fffffff {
			maxrp = 0x3fffffff
		}
		p = int(maxrp / uint64(r))
	}
	if ln >= 63 {
		return 0, 0, 0, errors.New("Invalid scrypt parameters")
	}
	N = 1 << ln
	return N, r, p, nil
}

// Decrypt decrypts the secret key in place. It is a no-op on an unencrypted
// key. The passphrase is verified against the stored Blake2b-256 checksum.
func (sk *PrivateKey) Decrypt(password string) error {
	if !sk.IsEncrypted() {
		return nil
	}
	if string(sk.KDFAlgorithm[:]) != kdfAlgScrypt {
		return errors.New("Unsupported KDF algorithm")
	}
	N, r, p, err := scryptParamsFromLimits(sk.KDFOpsLimit, sk.KDFMemLimit)
	if err != nil {
		return err
	}
	stream, err := scrypt.Key([]byte(password), sk.KDFSalt[:], N, r, p, streamLen)
	if err != nil {
		return err
	}
	defer wipe(stream)

	var keyId [8]byte
	var secret [64]byte
	var chk [32]byte
	defer wipe(secret[:])
	defer wipe(chk[:])
	for i := range keyId {
		keyId[i] = sk.KeyId[i] ^ stream[i]
	}
	for i := range secret {
		secret[i] = sk.SecretKey[i] ^ stream[8+i]
	}
	for i := range chk {
		chk[i] = sk.Checksum[i] ^ stream[72+i]
	}

	h, _ := blake2b.New256(nil)
	h.Write(sk.SignatureAlgorithm[:])
	h.Write(keyId[:])
	h.Write(secret[:])
	expected := h.Sum(nil)
	if subtle.ConstantTimeCompare(expected, chk[:]) != 1 {
		return errors.New("Wrong password")
	}
	sk.KeyId = keyId
	sk.SecretKey = secret
	sk.Checksum = chk
	sk.KDFAlgorithm = [2]byte{0, 0}
	return nil
}

// Wipe overwrites the in-memory secret key, checksum and key id with zeros.
// Call this when the key is no longer needed; the buffer will not be usable
// for signing afterwards.
func (sk *PrivateKey) Wipe() {
	wipe(sk.SecretKey[:])
	wipe(sk.Checksum[:])
	wipe(sk.KeyId[:])
}

func wipe(b []byte) {
	for i := range b {
		b[i] = 0
	}
}

// isPrintable matches the C reference's is_printable so that trusted comments
// we emit are accepted when read back.
func isPrintable(s string) bool {
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c == '\t' {
			continue
		}
		if c < 0x20 || c >= 0x7f {
			return false
		}
	}
	return true
}

// PublicKey derives the public key from the (decrypted) secret key.
func (sk *PrivateKey) PublicKey() PublicKey {
	var pk PublicKey
	pk.SignatureAlgorithm = [2]byte{'E', 'd'}
	pk.KeyId = sk.KeyId
	copy(pk.PublicKey[:], sk.SecretKey[32:64])
	return pk
}

// SignOptions configures a signing operation.
type SignOptions struct {
	// UntrustedComment is written as the first line of the .minisig file.
	// Empty defaults to "signature from minisign secret key".
	UntrustedComment string

	// TrustedComment is signed as part of the global signature.
	// Empty defaults to "timestamp:<unix>".
	TrustedComment string

	// Hashed selects the prehashed signature variant ("ED"), in which the
	// signature is computed over a Blake2b-512 hash of the message. This is
	// the recommended mode and is required for streaming signers. When
	// false, the signature is computed directly over the message bytes
	// (legacy "Ed" mode).
	Hashed bool
}

// Sign produces a minisign signature over data. The returned Signature can be
// serialized with Encode.
func (sk *PrivateKey) Sign(data []byte, opts SignOptions) (Signature, error) {
	if opts.Hashed {
		h, _ := blake2b.New512(nil)
		h.Write(data)
		return sk.signRaw(h.Sum(nil), true, opts)
	}
	return sk.signRaw(data, false, opts)
}

func (sk *PrivateKey) signRaw(message []byte, hashed bool, opts SignOptions) (Signature, error) {
	if sk.IsEncrypted() {
		return Signature{}, errors.New("Secret key is encrypted; call Decrypt first")
	}
	if string(sk.SignatureAlgorithm[:]) != sigAlgEd {
		return Signature{}, errors.New("Unsupported signature algorithm")
	}

	untrusted := opts.UntrustedComment
	if untrusted == "" {
		untrusted = defaultSigUntrusted
	}
	if hasNewline(untrusted) {
		return Signature{}, errors.New("Untrusted comment must fit on a single line")
	}
	trusted := opts.TrustedComment
	if trusted == "" {
		trusted = fmt.Sprintf("timestamp:%d", time.Now().Unix())
	}
	if hasNewline(trusted) {
		return Signature{}, errors.New("Trusted comment must fit on a single line")
	}
	if !isPrintable(trusted) {
		return Signature{}, errors.New("Trusted comment contains unprintable characters")
	}
	if len(trusted) > trustedCommentMaxLen {
		return Signature{}, errors.New("Trusted comment too long")
	}

	var sig Signature
	sig.KeyId = sk.KeyId
	sig.UntrustedComment = commentPrefix + untrusted
	sig.TrustedComment = trustedPrefix + trusted
	if hashed {
		sig.SignatureAlgorithm = [2]byte{'E', 'D'}
	} else {
		sig.SignatureAlgorithm = [2]byte{'E', 'd'}
	}

	edSK := ed25519.PrivateKey(sk.SecretKey[:])
	raw := ed25519.Sign(edSK, message)
	copy(sig.Signature[:], raw)

	global := make([]byte, 0, len(raw)+len(trusted))
	global = append(global, raw...)
	global = append(global, trusted...)
	copy(sig.GlobalSignature[:], ed25519.Sign(edSK, global))

	return sig, nil
}

// SignFile signs the contents of a file. In Hashed mode the file is streamed
// through Blake2b-512 with constant memory; in legacy mode the file is loaded
// into memory because Ed25519 needs the message twice.
//
// If opts.TrustedComment is empty, a default of "timestamp:<unix>\tfile:<base>"
// (with "\thashed" appended for prehashed signatures) is used — matching what
// the reference minisign(1) CLI writes.
func (sk *PrivateKey) SignFile(file string, opts SignOptions) (Signature, error) {
	if opts.TrustedComment == "" {
		opts.TrustedComment = defaultTrustedComment(filepath.Base(file), opts.Hashed)
	}
	if !opts.Hashed {
		data, err := os.ReadFile(file)
		if err != nil {
			return Signature{}, err
		}
		return sk.signRaw(data, false, opts)
	}
	f, err := os.Open(file)
	if err != nil {
		return Signature{}, err
	}
	defer f.Close()
	h, _ := blake2b.New512(nil)
	if _, err := io.Copy(h, f); err != nil {
		return Signature{}, err
	}
	return sk.signRaw(h.Sum(nil), true, opts)
}

func defaultTrustedComment(basename string, hashed bool) string {
	suffix := ""
	if hashed {
		suffix = "\thashed"
	}
	return fmt.Sprintf("timestamp:%d\tfile:%s%s", time.Now().Unix(), basename, suffix)
}

// Encode serializes the signature into the textual minisign .minisig format.
func (sig Signature) Encode() []byte {
	bin1 := make([]byte, 0, 74)
	bin1 = append(bin1, sig.SignatureAlgorithm[:]...)
	bin1 = append(bin1, sig.KeyId[:]...)
	bin1 = append(bin1, sig.Signature[:]...)

	untrusted := sig.UntrustedComment
	if untrusted == "" {
		untrusted = commentPrefix + defaultSigUntrusted
	} else if !strings.HasPrefix(untrusted, commentPrefix) {
		untrusted = commentPrefix + untrusted
	}
	trusted := sig.TrustedComment
	if !strings.HasPrefix(trusted, trustedPrefix) {
		trusted = trustedPrefix + trusted
	}

	var buf bytes.Buffer
	buf.WriteString(untrusted)
	buf.WriteByte('\n')
	buf.WriteString(base64.StdEncoding.EncodeToString(bin1))
	buf.WriteByte('\n')
	buf.WriteString(trusted)
	buf.WriteByte('\n')
	buf.WriteString(base64.StdEncoding.EncodeToString(sig.GlobalSignature[:]))
	buf.WriteByte('\n')
	return buf.Bytes()
}

// MarshalText implements encoding.TextMarshaler.
func (sig Signature) MarshalText() ([]byte, error) {
	return sig.Encode(), nil
}

func hasNewline(s string) bool {
	return strings.ContainsAny(s, "\r\n")
}
