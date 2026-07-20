package object

import (
	"bufio"
	"bytes"
	"fmt"
	"io"

	"github.com/go-git/go-git/v5/plumbing"
)

// tagScanner holds the working state of the tag decoder driven by the
// stateFn loop in (*Tag).Decode. Each tagState reads one or more lines
// from r, updates the in-progress *Tag and the scanner's bookkeeping,
// and returns the state that should run next (or nil to stop).
type tagScanner struct {
	r      *bufio.Reader
	t      *Tag
	msgbuf bytes.Buffer

	// pending holds a line that was read but the current state decided to
	// hand back to the next state, paired with the io.EOF flag returned
	// when the line was originally read.
	pending    []byte
	pendingErr error

	// First-occurrence tracking: once the corresponding canonical
	// header has been decoded at its expected position, subsequent
	// occurrences (or out-of-position lines) are silently dropped,
	// matching the strict layout enforced by upstream's
	// parse_tag_buffer (tag.c:130).
	//
	// gpgsig-sha256 is recognized and skipped without exposing a new field
	// in v5.
	sawObject, sawType, sawName, sawTagger bool
}

// tagState is one step of the decoder state machine. Each function reads
// the lines it needs, mutates *Tag via s.t, and returns the next state
// to run (or nil to terminate the loop).
type tagState func(*tagScanner) (tagState, error)

// readLine returns the next line from the buffer, transparently
// consuming any line that was previously pushed back by a state that
// decided not to handle it.
func (s *tagScanner) readLine() ([]byte, error) {
	if s.pending != nil {
		line, err := s.pending, s.pendingErr
		s.pending, s.pendingErr = nil, nil
		return line, err
	}
	return s.r.ReadBytes('\n')
}

// pushBack stashes an unconsumed line so the next state's readLine call
// sees it. Only one line can be pushed back at a time.
func (s *tagScanner) pushBack(line []byte, err error) {
	s.pending = line
	s.pendingErr = err
}

// scanTagObject requires the first line to be `object HASH`, mirroring
// upstream's strict parse_tag_buffer (tag.c:151-156). Anything else
// returns ErrMalformedTag.
func scanTagObject(s *tagScanner) (tagState, error) {
	line, err := s.readLine()
	if err != nil && err != io.EOF {
		return nil, err
	}
	if len(line) == 0 || isBlankLine(line) {
		return nil, fmt.Errorf("%w: missing object header", ErrMalformedTag)
	}

	key, data := splitHeader(line)
	if key != "object" {
		return nil, fmt.Errorf("%w: object header must be first", ErrMalformedTag)
	}
	h, herr := parseObjectIDHex(data, ErrMalformedTag, "object")
	if herr != nil {
		return nil, herr
	}
	s.t.Target = h
	s.sawObject = true
	if err == io.EOF {
		return nil, nil
	}
	return scanTagType, nil
}

// scanTagType requires a `type` line immediately after the object header,
// mirroring upstream's parse_tag_buffer (tag.c:158-166).
func scanTagType(s *tagScanner) (tagState, error) {
	line, err := s.readLine()
	if err != nil && err != io.EOF {
		return nil, err
	}
	if len(line) == 0 || isBlankLine(line) {
		return nil, fmt.Errorf("%w: missing type header", ErrMalformedTag)
	}

	key, data := splitHeader(line)
	if key != "type" {
		return nil, fmt.Errorf("%w: type header must follow object", ErrMalformedTag)
	}
	ot, perr := plumbing.ParseObjectType(string(data))
	if perr != nil {
		return nil, perr
	}
	s.t.TargetType = ot
	s.sawType = true
	if err == io.EOF {
		return nil, nil
	}
	return scanTagName, nil
}

// scanTagName requires a `tag` line immediately after the type header,
// mirroring upstream's parse_tag_buffer (tag.c:186-194).
func scanTagName(s *tagScanner) (tagState, error) {
	line, err := s.readLine()
	if err != nil && err != io.EOF {
		return nil, err
	}
	if len(line) == 0 || isBlankLine(line) {
		return nil, fmt.Errorf("%w: missing tag header", ErrMalformedTag)
	}

	key, data := splitHeader(line)
	if key != "tag" {
		return nil, fmt.Errorf("%w: tag header must follow type", ErrMalformedTag)
	}
	s.t.Name = string(data)
	s.sawName = true
	if err == io.EOF {
		return nil, nil
	}
	return scanTagTagger, nil
}

// scanTagTagger accepts a `tagger` line at its canonical position. Any
// other header is pushed back for scanTagHeaders.
func scanTagTagger(s *tagScanner) (tagState, error) {
	line, err := s.readLine()
	if err != nil && err != io.EOF {
		return nil, err
	}
	if len(line) == 0 {
		return nil, nil
	}
	if isBlankLine(line) {
		return scanTagMessage, nil
	}

	key, data := splitHeader(line)
	if key == "tagger" {
		s.t.Tagger.Decode(data)
		s.sawTagger = true
		if err == io.EOF {
			return nil, nil
		}
		return scanTagHeaders, nil
	}
	s.pushBack(line, err)
	return scanTagHeaders, nil
}

// scanTagHeaders dispatches one header line. gpgsig-sha256 hands off to
// scanTagSkipCont so the continuation block can be consumed; out-of-position
// canonical fields and unknown headers are silently dropped.
func scanTagHeaders(s *tagScanner) (tagState, error) {
	line, err := s.readLine()
	if err != nil && err != io.EOF {
		return nil, err
	}
	if len(line) == 0 {
		return nil, nil
	}
	if isBlankLine(line) {
		return scanTagMessage, nil
	}

	key, _ := splitHeader(line)
	next := scanTagHeaders
	switch key {
	case "object", "type", "tag", "tagger":
		// Out-of-canonical-position duplicates are dropped, mirroring the
		// strict ordering of upstream's parse_tag_buffer.
	case headerpgp256:
		next = scanTagSkipCont
	default:
		// Unknown header: silently dropped (the Tag struct does not
		// expose ExtraHeaders).
	}

	if err == io.EOF {
		return nil, nil
	}
	return next, nil
}

// scanTagSkipCont discards continuation lines for a header scanTagHeaders chose
// to drop. The first non-continuation line is pushed back so scanTagHeaders can
// dispatch it.
func scanTagSkipCont(s *tagScanner) (tagState, error) {
	line, err := s.readLine()
	if err != nil && err != io.EOF {
		return nil, err
	}
	if len(line) > 0 && line[0] == ' ' {
		if err == io.EOF {
			return nil, nil
		}
		return scanTagSkipCont, nil
	}
	if len(line) > 0 {
		s.pushBack(line, err)
	}
	return scanTagHeaders, nil
}

// scanTagMessage drains the remaining bytes into the message buffer.
// (*Tag).Decode then runs parseSignedBytes over those bytes to peel off
// the optional inline trailing PGP signature.
func scanTagMessage(s *tagScanner) (tagState, error) {
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
