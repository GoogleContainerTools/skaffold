// Copyright 2026 The Go Language Server Authors. All rights reserved.
// SPDX-License-Identifier: BSD-3-Clause

package jsonrpc2

// fields holds the byte spans of the recognized top-level members of a single
// JSON-RPC object, together with presence flags. A span points into the scanned
// input and is valid only until that input is reused; presence is tracked
// separately so that a member that is present with a null value can be
// distinguished from a member that is absent.
type fields struct {
	id      []byte
	method  []byte
	params  []byte
	result  []byte
	errobj  []byte
	jsonrpc []byte

	hasID     bool
	hasMethod bool
	hasParams bool
	hasResult bool
	hasError  bool
}

// scanObject scans a single JSON object value, recording the spans of the six
// recognized JSON-RPC members. data must begin at the opening '{' (after
// trimming) and the object must be the only value in data (aside from trailing
// whitespace). Duplicate keys follow last-wins semantics, matching encoding/json.
//
// It returns the number of input bytes consumed up to and including the closing
// '}', and whether data held a well-formed object.
func scanObject(data []byte, f *fields) (end int, ok bool) {
	i := skipSpace(data, 0)
	if i >= len(data) || data[i] != '{' {
		return 0, false
	}
	i++ // consume '{'

	i = skipSpace(data, i)
	if i < len(data) && data[i] == '}' {
		return i + 1, true // empty object
	}

	for {
		i = skipSpace(data, i)
		if i >= len(data) || data[i] != '"' {
			return 0, false
		}
		keyStart := i
		keyEnd, kok := scanString(data, i)
		if !kok {
			return 0, false
		}
		key := data[keyStart:keyEnd]
		i = keyEnd

		i = skipSpace(data, i)
		if i >= len(data) || data[i] != ':' {
			return 0, false
		}
		i++ // consume ':'

		i = skipSpace(data, i)
		valStart := i
		valEnd, vok := scanValue(data, i)
		if !vok {
			return 0, false
		}
		val := data[valStart:valEnd]
		i = valEnd

		assignField(f, key, val)

		i = skipSpace(data, i)
		if i >= len(data) {
			return 0, false
		}
		switch data[i] {
		case ',':
			i++
			continue
		case '}':
			return i + 1, true
		default:
			return 0, false
		}
	}
}

// assignField records the span val under the recognized member named by key
// (the key span includes its surrounding quotes). Unrecognized members are
// ignored. Recognition uses the raw quoted key bytes; recognized member names
// contain no escapes, so a key span carrying a backslash cannot match.
func assignField(f *fields, key, val []byte) {
	switch jsonrpcField(key) {
	case fieldJSONRPC:
		f.jsonrpc = val
	case fieldID:
		f.id = val
		f.hasID = true
	case fieldMethod:
		f.method = val
		f.hasMethod = true
	case fieldParams:
		f.params = val
		f.hasParams = true
	case fieldResult:
		f.result = val
		f.hasResult = true
	case fieldError:
		f.errobj = val
		f.hasError = true
	}
}

type fieldKind uint8

const (
	fieldUnknown fieldKind = iota
	fieldJSONRPC
	fieldID
	fieldMethod
	fieldParams
	fieldResult
	fieldError
)

func jsonrpcField(key []byte) fieldKind {
	switch len(key) {
	case len(`"id"`):
		if key[0] == '"' && key[1] == 'i' && key[2] == 'd' && key[3] == '"' {
			return fieldID
		}
	case len(`"error"`):
		if key[0] == '"' && key[1] == 'e' && key[2] == 'r' && key[3] == 'r' &&
			key[4] == 'o' && key[5] == 'r' && key[6] == '"' {
			return fieldError
		}
	case len(`"method"`):
		if key[0] != '"' || key[7] != '"' {
			return fieldUnknown
		}
		switch key[1] {
		case 'm':
			if key[2] == 'e' && key[3] == 't' && key[4] == 'h' &&
				key[5] == 'o' && key[6] == 'd' {
				return fieldMethod
			}
		case 'p':
			if key[2] == 'a' && key[3] == 'r' && key[4] == 'a' &&
				key[5] == 'm' && key[6] == 's' {
				return fieldParams
			}
		case 'r':
			if key[2] == 'e' && key[3] == 's' && key[4] == 'u' &&
				key[5] == 'l' && key[6] == 't' {
				return fieldResult
			}
		}
	case len(`"jsonrpc"`):
		if key[0] == '"' && key[1] == 'j' && key[2] == 's' && key[3] == 'o' &&
			key[4] == 'n' && key[5] == 'r' && key[6] == 'p' && key[7] == 'c' &&
			key[8] == '"' {
			return fieldJSONRPC
		}
	}
	return fieldUnknown
}

// bytesEqualString reports whether b contains exactly s, without converting b
// to a string. It is used by scanner hot paths that compare against fixed JSON
// literals.
func bytesEqualString(b []byte, s string) bool {
	if len(b) != len(s) {
		return false
	}
	for i := range len(s) {
		if b[i] != s[i] {
			return false
		}
	}
	return true
}

// scanValue scans a single JSON value beginning at data[i] (which must be the
// first byte of the value, with leading whitespace already skipped). It returns
// the index just past the value and whether the value is structurally
// well-formed. The scan tracks string/escape state and brace/bracket nesting so
// that strings and nested containers are skipped without interpretation.
func scanValue(data []byte, i int) (end int, ok bool) {
	if i >= len(data) {
		return 0, false
	}
	switch c := data[i]; c {
	case '"':
		return scanString(data, i)
	case '{', '[':
		return scanContainer(data, i)
	case 't':
		return scanLiteral(data, i, "true")
	case 'f':
		return scanLiteral(data, i, "false")
	case 'n':
		return scanLiteral(data, i, "null")
	default:
		if c == '-' || (c >= '0' && c <= '9') {
			return scanNumber(data, i)
		}
		return 0, false
	}
}

// scanString scans a JSON string beginning at the opening quote data[i]. It
// returns the index just past the closing quote.
func scanString(data []byte, i int) (end int, ok bool) {
	// data[i] is '"'.
	i++
	for i < len(data) {
		switch data[i] {
		case '"':
			return i + 1, true
		case '\\':
			i += 2 // skip the escaped byte; \uXXXX hex digits are validated on decode
		default:
			i++
		}
	}
	return 0, false
}

// scanContainer scans a JSON object or array beginning at data[i] (an opening
// '{' or '['), tracking nested containers and string contents. It returns the
// index just past the matching closing bracket.
//
// It validates bracket balance and string escaping only, not the interior JSON
// grammar: a structurally malformed interior such as "{]", `{"a":1,,}`,
// `{"a"1}`, or "[1 2 3]" is accepted as an opaque, balanced span. This is a
// deliberate passthrough choice so that params/result/error-data values are
// carried verbatim without a full structural parse; the payload codec validates
// them when the user decodes the [RawMessage]. The bracket-balance guarantee is
// sufficient for the scanner to locate member boundaries correctly.
func scanContainer(data []byte, i int) (end int, ok bool) {
	depth := 0
	for i < len(data) {
		switch data[i] {
		case '"':
			j, sok := scanString(data, i)
			if !sok {
				return 0, false
			}
			i = j
			continue
		case '{', '[':
			depth++
		case '}', ']':
			depth--
			if depth == 0 {
				return i + 1, true
			}
		}
		i++
	}
	return 0, false
}

// scanNumber scans a JSON number beginning at data[i]. It accepts the JSON
// number grammar (optional sign, integer part, optional fraction, optional
// exponent) and returns the index just past the last digit. Its only caller
// (scanValue) enters on a '-' or a digit, so the integer part always consumes at
// least one byte or returns false; no "consumed nothing" guard is needed.
func scanNumber(data []byte, i int) (end int, ok bool) {
	if i < len(data) && data[i] == '-' {
		i++
	}
	// Integer part.
	if i >= len(data) {
		return 0, false
	}
	if data[i] == '0' {
		i++
	} else if data[i] >= '1' && data[i] <= '9' {
		for i < len(data) && data[i] >= '0' && data[i] <= '9' {
			i++
		}
	} else {
		return 0, false
	}
	// Fraction.
	if i < len(data) && data[i] == '.' {
		i++
		if i >= len(data) || data[i] < '0' || data[i] > '9' {
			return 0, false
		}
		for i < len(data) && data[i] >= '0' && data[i] <= '9' {
			i++
		}
	}
	// Exponent.
	if i < len(data) && (data[i] == 'e' || data[i] == 'E') {
		i++
		if i < len(data) && (data[i] == '+' || data[i] == '-') {
			i++
		}
		if i >= len(data) || data[i] < '0' || data[i] > '9' {
			return 0, false
		}
		for i < len(data) && data[i] >= '0' && data[i] <= '9' {
			i++
		}
	}
	return i, true
}

// scanLiteral matches the fixed literal lit (true, false, or null) at data[i].
func scanLiteral(data []byte, i int, lit string) (end int, ok bool) {
	if i+len(lit) > len(data) {
		return 0, false
	}
	if !bytesEqualString(data[i:i+len(lit)], lit) {
		return 0, false
	}
	return i + len(lit), true
}

// skipSpace returns the index of the first non-whitespace byte at or after i.
func skipSpace(data []byte, i int) int {
	for i < len(data) {
		switch data[i] {
		case ' ', '\t', '\n', '\r':
			i++
		default:
			return i
		}
	}
	return i
}

// scanErrorObject scans the nested error object span and records the spans of
// its code, message, and data members. It is used by [decodeError]. It returns
// false when span is not a single well-formed object.
func scanErrorObject(span []byte, codeSpan, msgSpan, dataSpan *[]byte) (ok bool) {
	i := skipSpace(span, 0)
	if i >= len(span) || span[i] != '{' {
		return false
	}
	i++
	i = skipSpace(span, i)
	if i < len(span) && span[i] == '}' {
		return true
	}
	for {
		i = skipSpace(span, i)
		if i >= len(span) || span[i] != '"' {
			return false
		}
		keyStart := i
		keyEnd, kok := scanString(span, i)
		if !kok {
			return false
		}
		key := span[keyStart:keyEnd]
		i = skipSpace(span, keyEnd)
		if i >= len(span) || span[i] != ':' {
			return false
		}
		i++
		i = skipSpace(span, i)
		valStart := i
		valEnd, vok := scanValue(span, i)
		if !vok {
			return false
		}
		val := span[valStart:valEnd]
		i = valEnd

		switch {
		case bytesEqualString(key, `"code"`):
			*codeSpan = val
		case bytesEqualString(key, `"message"`):
			*msgSpan = val
		case bytesEqualString(key, `"data"`):
			if !isNullLiteral(val) {
				*dataSpan = val
			}
		}

		i = skipSpace(span, i)
		if i >= len(span) {
			return false
		}
		switch span[i] {
		case ',':
			i++
			continue
		case '}':
			return true
		default:
			return false
		}
	}
}

// DecodeMessage decodes a single JSON-RPC message from data into a [Message].
//
// The decoder uses a hand-written span scanner: it locates the recognized
// top-level members without building a map or decoding into a reflection
// struct and builds the message from the recorded spans.
//
// Lifetime contract (v2): a decoded request's method string and params
// [RawMessage] BORROW data — they are valid only until data is reused or
// mutated. Inside a connection the borrow is bounded by "until the handler
// returns"; a caller that needs the request afterward must copy what it
// retains (see [Request.Clone] for the concrete request shape). Response
// results and error members are still copied into owned allocations.
//
// A batch (a value whose first non-whitespace byte is '[') is rejected; use
// [ParseRequests] to handle single-or-batch request input.
//
// String contents (method names, string identifiers, and error messages) are
// decoded with JSON escape handling but are otherwise preserved verbatim and are
// not UTF-8-validated; invalid UTF-8 bytes are passed through unchanged rather
// than replaced.
func DecodeMessage(data []byte) (Message, error) {
	i := skipSpace(data, 0)
	if i < len(data) && data[i] == '[' {
		return nil, ErrInvalidRequest
	}

	var f fields
	end, ok := scanObject(data, &f)
	if !ok {
		return nil, ErrParse
	}
	if skipSpace(data, end) != len(data) {
		// Trailing content after the object is not a single message.
		return nil, ErrParse
	}

	return f.toMessage()
}

// toMessage classifies a scanned object as a request (call/notification) or a
// response and builds the corresponding [Message]. The role rules are: a present
// "method" member (and no result/error) marks a request; an object with no
// "method" is a response. Within a request, an absent or null "id" marks a
// notification.
func (f *fields) toMessage() (Message, error) {
	switch {
	case f.hasMethod && !f.hasResult && !f.hasError:
		return f.toRequest()
	case !f.hasMethod:
		return f.toResponse()
	default:
		// A "method" alongside a "result"/"error" is a malformed mixed message.
		return nil, ErrInvalidRequest
	}
}

// validVersion reports whether the scanned "jsonrpc" member is exactly the
// string "2.0". The span carries its surrounding quotes (scanString spans
// quote-to-quote), so the comparison is against the 5-byte literal `"2.0"`. An
// absent member yields a nil span and fails the check, as do a non-string value
// (a number such as 2.0 has no quotes), the null literal, and any other version
// string such as "1.0".
func (f *fields) validVersion() bool {
	return bytesEqualString(f.jsonrpc, `"2.0"`)
}

// toRequest builds a [*Call] or [*Notification] from a request object, copying
// the method and params spans into one allocation owned by the message. It
// rejects any object whose "jsonrpc" member is not exactly "2.0", which gates
// both the single-message and batch request paths.
func (f *fields) toRequest() (Message, error) {
	if !f.validVersion() {
		return nil, ErrInvalidRequest
	}

	// The method string and params span BORROW the scanned input rather than
	// copying it; inside a connection the borrow is valid until the handler
	// returns (the read loop does not reuse the frame before then), and
	// retention requires a clone. This is the package's documented decode
	// contract (see DecodeMessage and Handler).
	method, mok := borrowJSONString(f.method)
	if !mok {
		return nil, ErrInvalidRequest
	}

	// "params":null is treated as absent, matching the convention that null
	// parameters mean "no parameters".
	var params []byte
	if f.hasParams && !isNullLiteral(f.params) {
		params = f.params
	}

	// An absent or null id marks a notification; otherwise it is a call.
	if !f.hasID || isNullLiteral(f.id) {
		return &Notification{method: method, params: RawMessage(params)}, nil
	}

	id, idok := decodeID(f.id)
	if !idok {
		return nil, ErrInvalidRequest
	}
	return &Call{id: id, method: method, params: RawMessage(params)}, nil
}

// scanRequest scans frame as a single JSON-RPC request object directly into
// a [Request] value, allocating no message box. ok=false routes the frame to
// the general decode path: responses, batches, and malformed objects all fall
// through so their error semantics stay identical to [DecodeMessage].
func scanRequest(frame []byte) (rv Request, ok bool) {
	var f fields
	end, sok := scanObject(frame, &f)
	if !sok || skipSpace(frame, end) != len(frame) {
		return Request{}, false
	}
	if !f.hasMethod || f.hasResult || f.hasError {
		return Request{}, false
	}
	if f.fillRequest(&rv) != nil {
		return Request{}, false
	}
	return rv, true
}

// fillRequest validates a scanned request object and fills r with borrowed
// method and params spans. It mirrors toRequest without the message-box and
// payload allocations.
func (f *fields) fillRequest(r *Request) error {
	if !f.validVersion() {
		return ErrInvalidRequest
	}

	method, mok := borrowJSONString(f.method)
	if !mok {
		return ErrInvalidRequest
	}

	var params []byte
	if f.hasParams && !isNullLiteral(f.params) {
		params = f.params
	}

	if !f.hasID || isNullLiteral(f.id) {
		*r = Request{method: method, params: RawMessage(params)}
		return nil
	}

	id, idok := decodeID(f.id)
	if !idok {
		return ErrInvalidRequest
	}
	*r = Request{id: id, method: method, params: RawMessage(params), isCall: true}
	return nil
}

// toResponse builds a [*Response] from a response object. A present "error"
// member yields an error response; otherwise a present "result" member (even the
// JSON null literal) yields a success response. The id may be null or absent, in
// which case the response carries the unset (kind none) identifier.
//
// It rejects any object whose "jsonrpc" member is not exactly "2.0", and any
// object that carries both "result" and "error", which the specification forbids
// (a response holds exactly one of the two).
func (f *fields) toResponse() (Message, error) {
	if !f.validVersion() {
		return nil, ErrInvalidRequest
	}

	if f.hasResult && f.hasError {
		// A response must carry exactly one of result or error; both present is
		// malformed and must not silently drop one member.
		return nil, ErrInvalidRequest
	}

	var id ID
	if f.hasID {
		var idok bool
		if id, idok = decodeID(f.id); !idok {
			return nil, ErrInvalidRequest
		}
	}

	if f.hasError {
		if isNullLiteral(f.errobj) {
			return nil, ErrInvalidRequest
		}
		e, eok := decodeError(f.errobj)
		if !eok {
			return nil, ErrInvalidRequest
		}
		return &Response{id: id, err: e}, nil
	}

	if !f.hasResult {
		// Neither result nor error is present: not a valid response.
		return nil, ErrInvalidRequest
	}
	// "result":null is a valid success result and is preserved verbatim.
	buf := cloneBytes(f.result)
	return &Response{id: id, result: RawMessage(buf)}, nil
}

// ParsedMessage is the parsed form of one request in a single-or-batch input.
//
// When the message is well-formed Err is nil and Msg holds the decoded [*Call]
// or [*Notification]. When the message is malformed Err describes why and Msg is
// nil; the surrounding batch is still reported so that a caller can answer the
// valid entries and produce error responses for the invalid ones. Batch reports
// whether the message came from a JSON array.
type ParsedMessage struct {
	// Msg holds the decoded [*Call] or [*Notification] when the message is
	// well-formed, and is nil when Err is set.
	Msg RequestMessage

	// Err describes why a malformed message could not be decoded, and is nil for
	// a well-formed message.
	Err *Error

	// Batch reports whether the message came from a top-level JSON array.
	Batch bool
}

// ParseRequests parses a single request or a batch of requests from data.
//
// It returns an error only when data is not structurally valid JSON at the top
// level (neither a single object nor an array of objects). Per-message problems
// are reported in the Err field of the corresponding [ParsedMessage] rather than
// failing the whole parse, mirroring the lenient behavior expected of a server
// front-end.
//
// Lifetime: each parsed message's method and params BORROW data (escape-bearing
// strings are decoded into owned copies). They are valid only until data is
// reused or mutated; copy what you retain. Ids and error members are owned.
func ParseRequests(data []byte) ([]*ParsedMessage, error) {
	i := skipSpace(data, 0)
	if i >= len(data) {
		return nil, ErrParse
	}

	if data[i] != '[' {
		pm := parseOneRequest(data, false)
		return []*ParsedMessage{pm}, nil
	}

	// Batch: scan a JSON array of object spans.
	spans, ok := scanArrayElements(data, i)
	if !ok {
		return nil, ErrInvalidRequest
	}
	out := make([]*ParsedMessage, 0, len(spans))
	for _, span := range spans {
		out = append(out, parseOneRequest(span, true))
	}
	return out, nil
}

// parseOneRequest scans one request object span and classifies it as a call or
// notification. A response object (no method) is reported as an invalid request,
// since [ParseRequests] is a request-side entry point.
func parseOneRequest(data []byte, batch bool) *ParsedMessage {
	var f fields
	end, ok := scanObject(data, &f)
	if !ok || skipSpace(data, end) != len(data) {
		return &ParsedMessage{Err: ErrParse, Batch: batch}
	}
	if !f.hasMethod {
		return &ParsedMessage{Err: ErrInvalidRequest, Batch: batch}
	}
	msg, err := f.toRequest()
	if err != nil {
		return &ParsedMessage{Err: ErrInvalidRequest, Batch: batch}
	}
	return &ParsedMessage{Msg: msg.(RequestMessage), Batch: batch}
}

// scanArrayElements scans a JSON array beginning at data[i] (an opening '[') and
// returns a slice of the spans of its top-level elements. It returns false when
// the array is not well-formed.
func scanArrayElements(data []byte, i int) (spans [][]byte, ok bool) {
	// data[i] is '['.
	i++
	i = skipSpace(data, i)
	if i < len(data) && data[i] == ']' {
		return nil, true // empty batch
	}
	for {
		i = skipSpace(data, i)
		valStart := i
		valEnd, vok := scanValue(data, i)
		if !vok {
			return nil, false
		}
		spans = append(spans, data[valStart:valEnd])
		i = skipSpace(data, valEnd)
		if i >= len(data) {
			return nil, false
		}
		switch data[i] {
		case ',':
			i++
			continue
		case ']':
			if skipSpace(data, i+1) != len(data) {
				return nil, false
			}
			return spans, true
		default:
			return nil, false
		}
	}
}
