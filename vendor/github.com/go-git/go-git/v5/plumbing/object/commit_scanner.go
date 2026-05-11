package object

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"strings"

	"github.com/go-git/go-git/v5/plumbing"
)

// commitScanner holds the working state of the commit decoder driven by the
// stateFn loop in (*Commit).Decode. Each commitState reads one or more lines
// from r, updates the in-progress *Commit and the scanner's bookkeeping, and
// returns the state that should run next (or nil to stop).
type commitScanner struct {
	r      *bufio.Reader
	c      *Commit
	msgbuf bytes.Buffer

	// pending holds a line that was read but the current state decided to
	// hand back to the next state, paired with the io.EOF flag that was
	// returned when the line was originally read.
	pending    []byte
	pendingErr error

	// First-occurrence tracking: once the corresponding field has been
	// decoded, subsequent occurrences are silently dropped (matches
	// upstream's find_commit_header / first-wins semantics).
	//
	// gpgsig is not tracked here: upstream's parse_buffer_signed_by_header
	// (commit.c:1186) accumulates every occurrence into one signature buffer,
	// so we do the same on the scanner side to keep verification payloads
	// byte-aligned. gpgsig-sha256 is recognized and skipped without exposing a
	// new field in v5.
	sawTree, sawAuthor, sawCommitter bool
	sawEncoding, sawMergetag         bool

	// extra is the multi-line ExtraHeader currently being assembled.
	extra *ExtraHeader
}

// commitState is one step of the decoder state machine. Each function reads
// the lines it needs, mutates *Commit via s.c, and returns the next state to
// run (or nil to terminate the loop).
type commitState func(*commitScanner) (commitState, error)

// readLine returns the next line from the buffer, transparently consuming any
// line that was previously pushed back by a state that decided not to handle
// it.
func (s *commitScanner) readLine() ([]byte, error) {
	if s.pending != nil {
		line, err := s.pending, s.pendingErr
		s.pending, s.pendingErr = nil, nil
		return line, err
	}
	line, err := s.r.ReadBytes('\n')
	if err != nil && err != io.EOF {
		return line, err
	}
	return line, err
}

// pushBack stashes an unconsumed line so the next state's readLine call sees
// it. Only one line can be pushed back at a time.
func (s *commitScanner) pushBack(line []byte, err error) {
	s.pending = line
	s.pendingErr = err
}

// scanTree expects the first non-empty header to be `tree HASH`. Anything
// else (or an empty buffer) is rejected with ErrMalformedCommit, matching
// upstream's `bogus commit object` check.
func scanTree(s *commitScanner) (commitState, error) {
	line, err := s.readLine()
	if err != nil && err != io.EOF {
		return nil, err
	}
	if len(line) == 0 || isBlankLine(line) {
		return nil, fmt.Errorf("%w: missing tree header", ErrMalformedCommit)
	}

	key, data := splitHeader(line)
	if key != "tree" {
		return nil, fmt.Errorf("%w: tree header must be first", ErrMalformedCommit)
	}
	h, herr := parseObjectIDHex(data, ErrMalformedCommit, "tree")
	if herr != nil {
		return nil, herr
	}
	s.c.TreeHash = h
	s.sawTree = true
	if err == io.EOF {
		return nil, nil
	}
	return scanParents, nil
}

// scanParents consumes contiguous `parent HASH` lines. The first non-parent
// line ends the parent block and is handed off to scanAuthor; any later
// `parent` line is silently dropped (matches upstream's parse_commit_buffer
// exiting its parent loop at the first non-parent line and
// read_commit_extra_header_lines filtering `parent` out of extras).
func scanParents(s *commitScanner) (commitState, error) {
	line, err := s.readLine()
	if err != nil && err != io.EOF {
		return nil, err
	}
	if len(line) == 0 {
		return nil, nil
	}
	if isBlankLine(line) {
		return scanMessage, nil
	}

	key, data := splitHeader(line)
	if key == "parent" {
		h, herr := parseObjectIDHex(data, ErrMalformedCommit, "parent")
		if herr != nil {
			return nil, herr
		}
		s.c.ParentHashes = append(s.c.ParentHashes, h)
		if err == io.EOF {
			return nil, nil
		}
		return scanParents, nil
	}
	s.pushBack(line, err)
	return scanAuthor, nil
}

// scanAuthor accepts an `author` line at its canonical position immediately
// after the parent block. Any other header here is pushed back for
// scanCommitter; an out-of-place author is therefore silently dropped.
// Mirrors upstream's parse_commit_date func.
func scanAuthor(s *commitScanner) (commitState, error) {
	line, err := s.readLine()
	if err != nil && err != io.EOF {
		return nil, err
	}
	if len(line) == 0 {
		return nil, nil
	}
	if isBlankLine(line) {
		return scanMessage, nil
	}

	key, data := splitHeader(line)
	if key == "author" {
		s.c.Author.Decode(data)
		s.sawAuthor = true
		if err == io.EOF {
			return nil, nil
		}
		return scanCommitter, nil
	}
	s.pushBack(line, err)
	return scanCommitter, nil
}

// scanCommitter accepts a `committer` line at its canonical position
// immediately after the author. Any other header is pushed back for
// scanHeaders. Same upstream rationale as scanAuthor.
func scanCommitter(s *commitScanner) (commitState, error) {
	line, err := s.readLine()
	if err != nil && err != io.EOF {
		return nil, err
	}
	if len(line) == 0 {
		return nil, nil
	}
	if isBlankLine(line) {
		return scanMessage, nil
	}

	key, data := splitHeader(line)
	if key == "committer" {
		s.c.Committer.Decode(data)
		s.sawCommitter = true
		if err == io.EOF {
			return nil, nil
		}
		return scanHeaders, nil
	}
	s.pushBack(line, err)
	return scanHeaders, nil
}

// scanHeaders dispatches one header line. Continuation-bearing headers
// (mergetag, gpgsig, gpgsig-sha256, and unknown extras whose value is
// continued on subsequent lines) hand off to a dedicated continuation state
// that handles the `<space>...` lines and then returns here.
func scanHeaders(s *commitScanner) (commitState, error) {
	line, err := s.readLine()
	if err != nil && err != io.EOF {
		return nil, err
	}
	if len(line) == 0 {
		return nil, nil
	}
	if isBlankLine(line) {
		return scanMessage, nil
	}

	originalLine := line
	key, data := splitHeader(line)

	var next commitState = scanHeaders
	switch key {
	case "tree", "parent", "author", "committer":
		// Anything reaching scanHeaders with one of these keys is out of
		// canonical position: duplicate tree, parent past the contiguous
		// block, or author/committer not at their expected slot. Drop them
		// the same way upstream's standard_header_field filter excludes
		// them from the extras list (read_commit_extra_header_lines,
		// commit.c:1520-1522).
	case headerencoding:
		if !s.sawEncoding {
			s.c.Encoding = MessageEncoding(data)
			s.sawEncoding = true
		}
	case headermergetag:
		if s.sawMergetag {
			next = scanSkipCont
		} else {
			s.c.MergeTag += string(data) + "\n"
			s.sawMergetag = true
			next = scanMergetagCont
		}
	case headerpgp:
		s.c.PGPSignature += string(data) + "\n"
		next = scanPgpCont
	case headerpgp256:
		next = scanSkipCont
	default:
		h, multiline := parseExtraHeader(originalLine)
		if multiline {
			s.extra = &h
			next = scanExtraCont
		} else {
			s.c.ExtraHeaders = append(s.c.ExtraHeaders, h)
		}
	}

	if err == io.EOF {
		return nil, nil
	}
	return next, nil
}

// scanMergetagCont accumulates continuation lines for the first mergetag
// header. Continuations strip exactly one leading space, mirroring upstream's
// `line + 1` (commit.c:1509). The first non-continuation line is pushed back
// so scanHeaders can dispatch it.
func scanMergetagCont(s *commitScanner) (commitState, error) {
	return continuationCont(s, &s.c.MergeTag, scanMergetagCont)
}

// scanPgpCont accumulates continuation lines for a signature header.
// Continuations strip exactly one leading space, mirroring upstream's
// `line + 1` (commit.c:1509). The first non-continuation line is pushed back
// so scanHeaders can dispatch it. Repeat occurrences of the same signature
// header land back here and concatenate, matching upstream's
// parse_buffer_signed_by_header (commit.c:1186).
func scanPgpCont(s *commitScanner) (commitState, error) {
	return continuationCont(s, &s.c.PGPSignature, scanPgpCont)
}

func continuationCont(s *commitScanner, dst *string, self commitState) (commitState, error) {
	line, err := s.readLine()
	if err != nil && err != io.EOF {
		return nil, err
	}
	if len(line) > 0 && line[0] == ' ' {
		*dst += string(line[1:])
		if err == io.EOF {
			return nil, nil
		}
		return self, nil
	}
	if len(line) > 0 {
		s.pushBack(line, err)
	}
	return scanHeaders, nil
}

// scanSkipCont discards continuation lines that belong to a header scanHeaders
// chose to drop.
func scanSkipCont(s *commitScanner) (commitState, error) {
	line, err := s.readLine()
	if err != nil && err != io.EOF {
		return nil, err
	}
	if len(line) > 0 && line[0] == ' ' {
		if err == io.EOF {
			return nil, nil
		}
		return scanSkipCont, nil
	}
	if len(line) > 0 {
		s.pushBack(line, err)
	}
	return scanHeaders, nil
}

// scanExtraCont accumulates continuation lines for an unknown ExtraHeader
// whose value spans multiple lines, then finalises the entry once the
// continuation block ends.
func scanExtraCont(s *commitScanner) (commitState, error) {
	line, err := s.readLine()
	if err != nil && err != io.EOF {
		return nil, err
	}
	if len(line) > 0 && line[0] == ' ' {
		s.extra.Value += string(line[1:])
		if err == io.EOF {
			s.finaliseExtra()
			return nil, nil
		}
		return scanExtraCont, nil
	}
	s.finaliseExtra()
	if len(line) > 0 {
		s.pushBack(line, err)
	}
	return scanHeaders, nil
}

func (s *commitScanner) finaliseExtra() {
	s.extra.Value = strings.TrimRight(s.extra.Value, "\n")
	s.c.ExtraHeaders = append(s.c.ExtraHeaders, *s.extra)
	s.extra = nil
}

// scanMessage drains the remaining bytes into the message buffer.
func scanMessage(s *commitScanner) (commitState, error) {
	for {
		line, err := s.readLine()
		if err != nil && err != io.EOF {
			return nil, err
		}
		if len(line) > 0 {
			s.msgbuf.Write(line)
		}
		if err == io.EOF {
			return nil, nil
		}
	}
}

// isBlankLine reports whether line is the canonical header/body separator:
// a single newline. Mirrors upstream's `*line == '\n'` test in
// read_commit_extra_header_lines (commit.c:1502).
func isBlankLine(line []byte) bool {
	return len(line) == 1 && line[0] == '\n'
}

// splitHeader returns the header keyword (everything before the first space)
// and the value (everything after, with the trailing newline stripped). If
// the header has no value the returned data is nil.
func splitHeader(line []byte) (string, []byte) {
	trimmed := bytes.TrimRight(line, "\n")
	key, value, ok := bytes.Cut(trimmed, []byte{' '})
	if !ok {
		return string(trimmed), nil
	}
	return string(key), value
}

func parseObjectIDHex(data []byte, malformedErr error, header string) (plumbing.Hash, error) {
	id := string(data)
	if !plumbing.IsHash(id) {
		return plumbing.ZeroHash, fmt.Errorf("%w: bad %s hash", malformedErr, header)
	}
	return plumbing.NewHash(id), nil
}
