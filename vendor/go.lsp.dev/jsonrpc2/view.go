// Copyright 2026 The Go Language Server Authors. All rights reserved.
// SPDX-License-Identifier: BSD-3-Clause

package jsonrpc2

// FrameView is a view of one framed JSON-RPC body.
//
// A FrameView returned by [ScanFrameView] borrows the frame slice passed to it:
// Bytes and every byte slice reachable from MessageView alias that slice. The
// borrowed view is valid only until the caller mutates/reuses the frame, the
// stream performs the next read into the same storage, or a callback that
// received the view returns. Call Clone before retaining a FrameView beyond that
// lifetime.
//
// A FrameView returned by [NewFrameView] or [FrameView.Clone] owns a private
// copy of the frame, so its borrowed spans remain valid for the FrameView's
// lifetime.
type FrameView struct {
	frame []byte
	msg   MessageView
	owns  bool
}

// ScanFrameView scans frame as one borrowed JSON-RPC message frame.
//
// The returned view aliases frame. Mutating or reusing frame mutates or
// invalidates all byte slices reachable from the returned FrameView.
func ScanFrameView(frame []byte) (FrameView, error) {
	msg, err := ScanMessageView(frame)
	if err != nil {
		return FrameView{}, err
	}
	return FrameView{frame: frame, msg: msg}, nil
}

// NewFrameView copies frame, scans the copy, and returns an owning FrameView.
//
// Use NewFrameView at explicit lifetime boundaries where a view must outlive the
// read buffer or callback that produced the original frame.
func NewFrameView(frame []byte) (FrameView, error) {
	owned := cloneBytes(frame)
	msg, err := ScanMessageView(owned)
	if err != nil {
		return FrameView{}, err
	}
	return FrameView{frame: owned, msg: msg, owns: true}, nil
}

// Bytes returns the raw JSON frame bytes backing this view.
//
// For a borrowed FrameView the returned slice aliases the caller-provided frame.
// For an owning FrameView it aliases the FrameView's private copy.
func (v *FrameView) Bytes() []byte { return v.frame }

// MessageView returns the parsed borrowed message view for this frame.
func (v *FrameView) MessageView() MessageView { return v.msg }

// OwnsFrame reports whether this FrameView owns a private frame copy.
func (v *FrameView) OwnsFrame() bool { return v.owns }

// Clone returns an owning FrameView with all spans retargeted into a private
// copy of the frame.
func (v *FrameView) Clone() (FrameView, error) { return NewFrameView(v.frame) }

// MessageViewKind classifies a borrowed [MessageView].
type MessageViewKind uint8

const (
	// MessageViewInvalid is the zero value and is not returned for a successful
	// scan.
	MessageViewInvalid MessageViewKind = iota
	// MessageViewCall is a request that carries a non-null identifier and expects
	// a response.
	MessageViewCall
	// MessageViewNotification is a request with no identifier or a null
	// identifier.
	MessageViewNotification
	// MessageViewResponseResult is a response carrying a result member. A JSON
	// null result is represented by the borrowed bytes "null".
	MessageViewResponseResult
	// MessageViewResponseError is a response carrying an error object.
	MessageViewResponseError
)

// MessageViewResponse is a short alias for a successful response view.
const MessageViewResponse = MessageViewResponseResult

const (
	messageViewInvalidName      = "invalid"
	messageViewCallName         = "call"
	messageViewNotificationName = "notification"
	messageViewResponseName     = "response"
	messageViewErrorName        = "error"
)

// String returns a stable diagnostic name for k.
func (k MessageViewKind) String() string {
	switch k {
	case MessageViewInvalid:
		return messageViewInvalidName
	case MessageViewCall:
		return messageViewCallName
	case MessageViewNotification:
		return messageViewNotificationName
	case MessageViewResponseResult:
		return messageViewResponseName
	case MessageViewResponseError:
		return messageViewErrorName
	default:
		return messageViewInvalidName
	}
}

// MessageView is a zero-copy view of a single JSON-RPC message.
//
// Every byte slice in MessageView aliases the frame passed to [ScanMessageView]
// or [ScanFrameView]. The view is valid only while that frame remains valid and
// unmodified. Callback-scoped APIs may make this lifetime even shorter: when a
// view is delivered to a callback, callers must treat it as invalid after the
// callback returns unless they first call Clone or Owned.
//
// The view deliberately exposes borrowed []byte spans rather than [RawMessage]
// to avoid making borrowed bytes look like the package's ordinary owned decoded
// message payloads.
type MessageView struct {
	Error ErrorView

	// JSONRPC is the borrowed raw value span of the "jsonrpc" member.
	JSONRPC []byte

	// IDRaw is the borrowed raw value span of the "id" member. ID is the parsed
	// view of the same span. For notifications and null IDs, ID is invalid.
	IDRaw []byte

	// MethodRaw is the borrowed raw string value span, including quotes.
	// MethodBytes is the borrowed unquoted string body. When MethodEscaped is
	// true, MethodBytes contains raw escaped body bytes; use MethodString or
	// Owned at explicit slow/owning boundaries.
	MethodRaw   []byte
	MethodBytes []byte

	// Params and Result are borrowed raw JSON value spans. Params is nil when the
	// request omits params or explicitly sets params to null. Result preserves a
	// null result as the borrowed bytes "null".
	Params []byte
	Result []byte

	// ErrorRaw is the borrowed raw value span of the "error" member. Error is a
	// parsed borrowed view of the same error object when Kind is
	// MessageViewResponseError.
	ErrorRaw []byte
	ID       IDView

	Kind MessageViewKind

	MethodEscaped bool
}

// ParsedMessageView is the borrowed-view form of one request parsed from a
// single-or-batch input.
//
// When Err is nil, View holds a borrowed call or notification view. When Err is
// non-nil, View is invalid and Err describes the per-member parse/request
// problem. Batch reports whether the member came from a JSON array. All byte
// slices reachable from View alias the input passed to [ScanRequestViews] or
// [AppendRequestViews].
type ParsedMessageView struct {
	Err   *Error
	View  MessageView
	Batch bool
}

// ScanRequestViews parses a single request or a batch of requests into borrowed
// views.
//
// It mirrors [ParseRequests] but yields flat span views instead of boxed
// [Call] or [Notification] messages (whose method and params likewise borrow
// data). The returned views alias data and must not outlive the current
// frame/callback lifetime unless cloned or converted to owned messages.
func ScanRequestViews(data []byte) ([]ParsedMessageView, error) {
	return AppendRequestViews(nil, data)
}

// AppendRequestViews appends borrowed request views parsed from data to dst.
//
// The top-level error behavior matches [ParseRequests]: malformed arrays return
// an error for the whole input, while malformed single requests or malformed
// batch members are represented by per-entry Err values.
func AppendRequestViews(dst []ParsedMessageView, data []byte) ([]ParsedMessageView, error) {
	i := skipSpace(data, 0)
	if i >= len(data) {
		return nil, ErrParse
	}
	if data[i] != '[' {
		return append(dst, parseOneRequestView(data, false)), nil
	}

	origLen := len(dst)
	i++
	i = skipSpace(data, i)
	if i < len(data) && data[i] == ']' {
		if skipSpace(data, i+1) != len(data) {
			return dst[:origLen], ErrInvalidRequest
		}
		return dst, nil
	}
	for {
		i = skipSpace(data, i)
		valStart := i
		valEnd, ok := scanValue(data, i)
		if !ok {
			return dst[:origLen], ErrInvalidRequest
		}
		dst = append(dst, parseOneRequestView(data[valStart:valEnd], true))
		i = skipSpace(data, valEnd)
		if i >= len(data) {
			return dst[:origLen], ErrInvalidRequest
		}
		switch data[i] {
		case ',':
			i++
			continue
		case ']':
			if skipSpace(data, i+1) != len(data) {
				return dst[:origLen], ErrInvalidRequest
			}
			return dst, nil
		default:
			return dst[:origLen], ErrInvalidRequest
		}
	}
}

func parseOneRequestView(data []byte, batch bool) ParsedMessageView {
	var f fields
	end, ok := scanObject(data, &f)
	if !ok || skipSpace(data, end) != len(data) {
		return ParsedMessageView{Err: ErrParse, Batch: batch}
	}
	if !f.hasMethod {
		return ParsedMessageView{Err: ErrInvalidRequest, Batch: batch}
	}
	view, err := f.toRequestView()
	if err != nil {
		return ParsedMessageView{Err: ErrInvalidRequest, Batch: batch}
	}
	return ParsedMessageView{View: view, Batch: batch}
}

// MethodString decodes the method name as a Go string.
//
// It may allocate for escaped strings and for converting unescaped bytes to a
// string, so hot dispatch should compare MethodBytes directly where possible.
func (v *MessageView) MethodString() (string, bool) {
	if len(v.MethodRaw) == 0 {
		return "", false
	}
	return unquoteJSONString(v.MethodRaw)
}

// Clone copies every borrowed byte span in v and returns a retained MessageView.
//
// The returned view no longer aliases the scanned frame, but spans that referred
// to adjacent parts of the original frame are copied independently. Use
// [FrameView.Clone] when retaining a whole frame plus retargeted spans is more
// convenient.
func (v *MessageView) Clone() MessageView {
	return MessageView{
		Kind:          v.Kind,
		JSONRPC:       cloneBytes(v.JSONRPC),
		IDRaw:         cloneBytes(v.IDRaw),
		ID:            v.ID.clone(),
		MethodRaw:     cloneBytes(v.MethodRaw),
		MethodBytes:   cloneBytes(v.MethodBytes),
		MethodEscaped: v.MethodEscaped,
		Params:        cloneBytes(v.Params),
		Result:        cloneBytes(v.Result),
		ErrorRaw:      cloneBytes(v.ErrorRaw),
		Error:         v.Error.clone(),
	}
}

// Owned converts v into the package's ordinary owned [Message] representation.
func (v *MessageView) Owned() (Message, error) {
	switch v.Kind {
	case MessageViewCall:
		method, ok := v.MethodString()
		if !ok {
			return nil, ErrInvalidRequest
		}
		id, ok := v.ID.ID()
		if !ok || !id.IsValid() {
			return nil, ErrInvalidRequest
		}
		return &Call{id: id, method: method, params: RawMessage(cloneBytes(v.Params))}, nil
	case MessageViewNotification:
		method, ok := v.MethodString()
		if !ok {
			return nil, ErrInvalidRequest
		}
		return &Notification{method: method, params: RawMessage(cloneBytes(v.Params))}, nil
	case MessageViewResponseResult:
		id, ok := v.ID.ID()
		if !ok {
			return nil, ErrInvalidRequest
		}
		return &Response{id: id, result: RawMessage(cloneBytes(v.Result))}, nil
	case MessageViewResponseError:
		id, ok := v.ID.ID()
		if !ok {
			return nil, ErrInvalidRequest
		}
		err, ok := v.Error.Owned()
		if !ok {
			return nil, ErrInvalidRequest
		}
		return &Response{id: id, err: err}, nil
	case MessageViewInvalid:
		return nil, ErrInvalidRequest
	default:
		return nil, ErrInvalidRequest
	}
}

// IDView is a zero-copy view of a JSON-RPC identifier.
//
// StringBytes exposes the borrowed string body with surrounding quotes removed.
// When StringEscaped is true, those bytes are raw escaped body bytes, not decoded
// text; decode on demand with StringValue or ID.
type IDView struct {
	raw     []byte
	str     []byte
	num     int64
	kind    idKind
	escaped bool
}

// IsValid reports whether the identifier is set (number or string).
func (id IDView) IsValid() bool { return id.kind != idNone }

// IsNumber reports whether the identifier holds an integer value.
func (id IDView) IsNumber() bool { return id.kind == idNumber }

// IsString reports whether the identifier holds a string value.
func (id IDView) IsString() bool { return id.kind == idString }

// Number returns the integer value of the identifier and whether it is a number.
func (id IDView) Number() (int64, bool) { return id.num, id.kind == idNumber }

// StringBytes returns the borrowed string body and whether the ID is a string.
//
// When StringEscaped is true, the returned bytes are raw escaped body bytes; use
// StringValue or ID for the decoded slow path.
func (id IDView) StringBytes() ([]byte, bool) { return id.str, id.kind == idString }

// StringEscaped reports whether StringBytes carries raw escaped bytes rather
// than decoded string bytes.
func (id IDView) StringEscaped() bool { return id.kind == idString && id.escaped }

// StringValue decodes the identifier string. It may allocate for escaped IDs;
// hot paths should prefer StringBytes when possible.
func (id IDView) StringValue() (string, bool) {
	if id.kind != idString {
		return "", false
	}
	return unquoteJSONString(id.raw)
}

// ID converts id into the package's ordinary owned [ID] representation.
func (id IDView) ID() (ID, bool) {
	switch id.kind {
	case idNone:
		return ID{}, true
	case idNumber:
		return NewNumberID(id.num), true
	case idString:
		s, ok := id.StringValue()
		if !ok {
			return ID{}, false
		}
		return NewStringID(s), true
	default:
		return ID{}, false
	}
}

// Raw returns the borrowed raw JSON span for the identifier.
func (id IDView) Raw() []byte { return id.raw }

func (id IDView) clone() IDView {
	return IDView{
		num:     id.num,
		raw:     cloneBytes(id.raw),
		str:     cloneBytes(id.str),
		kind:    id.kind,
		escaped: id.escaped,
	}
}

// ErrorView is a zero-copy view of a JSON-RPC error object.
type ErrorView struct {
	// Raw is the borrowed raw error-object span.
	Raw []byte

	CodeRaw []byte

	// MessageRaw is the borrowed raw string value span, including quotes.
	// MessageBytes is the borrowed unquoted string body. When MessageEscaped is
	// true, MessageBytes contains raw escaped body bytes.
	MessageRaw   []byte
	MessageBytes []byte

	// Data is the borrowed raw "data" value span, or nil when absent or null.
	Data []byte

	Code           Code
	MessageEscaped bool
}

// MessageString decodes the error message as a Go string.
//
// It may allocate for escaped messages; hot paths should prefer MessageBytes
// when MessageEscaped is false.
func (e *ErrorView) MessageString() (string, bool) {
	if len(e.MessageRaw) == 0 {
		return "", false
	}
	return unquoteJSONString(e.MessageRaw)
}

// Owned converts e into the package's ordinary owned wire error.
func (e *ErrorView) Owned() (*Error, bool) {
	out := &Error{Code: e.Code}
	if len(e.MessageRaw) > 0 {
		msg, ok := e.MessageString()
		if !ok {
			return nil, false
		}
		out.Message = msg
	}
	out.Data = RawMessage(cloneBytes(e.Data))
	return out, true
}

func (e *ErrorView) clone() ErrorView {
	return ErrorView{
		Raw:            cloneBytes(e.Raw),
		Code:           e.Code,
		CodeRaw:        cloneBytes(e.CodeRaw),
		MessageRaw:     cloneBytes(e.MessageRaw),
		MessageBytes:   cloneBytes(e.MessageBytes),
		MessageEscaped: e.MessageEscaped,
		Data:           cloneBytes(e.Data),
	}
}

// ScanMessageView scans a single JSON-RPC message and returns a borrowed view of
// its recognized fields. It performs no copies on the common no-escape path.
//
// The returned view aliases frame. Use [MessageView.Clone],
// [MessageView.Owned], or [FrameView.Clone] before retaining it beyond the
// current read/callback lifetime.
func ScanMessageView(frame []byte) (MessageView, error) {
	i := skipSpace(frame, 0)
	if i < len(frame) && frame[i] == '[' {
		return MessageView{}, ErrInvalidRequest
	}

	var f fields
	end, ok := scanObject(frame, &f)
	if !ok {
		return MessageView{}, ErrParse
	}
	if skipSpace(frame, end) != len(frame) {
		return MessageView{}, ErrParse
	}

	return f.toMessageView()
}

func (f *fields) toMessageView() (MessageView, error) {
	switch {
	case f.hasMethod && !f.hasResult && !f.hasError:
		return f.toRequestView()
	case !f.hasMethod:
		return f.toResponseView()
	default:
		return MessageView{}, ErrInvalidRequest
	}
}

func (f *fields) toRequestView() (MessageView, error) {
	if !f.validVersion() {
		return MessageView{}, ErrInvalidRequest
	}

	method, methodEscaped, ok := jsonStringBody(f.method)
	if !ok {
		return MessageView{}, ErrInvalidRequest
	}

	var id IDView
	if f.hasID {
		var idok bool
		id, idok = parseIDView(f.id)
		if !idok {
			return MessageView{}, ErrInvalidRequest
		}
	}

	// "params":null is treated as absent, matching DecodeMessage.
	var params []byte
	if f.hasParams && !isNullLiteral(f.params) {
		params = f.params
	}

	kind := MessageViewNotification
	if f.hasID && id.IsValid() {
		kind = MessageViewCall
	}

	return MessageView{
		Kind:          kind,
		JSONRPC:       f.jsonrpc,
		IDRaw:         f.id,
		ID:            id,
		MethodRaw:     f.method,
		MethodBytes:   method,
		MethodEscaped: methodEscaped,
		Params:        params,
	}, nil
}

func (f *fields) toResponseView() (MessageView, error) {
	if !f.validVersion() {
		return MessageView{}, ErrInvalidRequest
	}
	if f.hasResult && f.hasError {
		return MessageView{}, ErrInvalidRequest
	}

	var id IDView
	if f.hasID {
		var idok bool
		id, idok = parseIDView(f.id)
		if !idok {
			return MessageView{}, ErrInvalidRequest
		}
	}

	if f.hasError {
		if isNullLiteral(f.errobj) {
			return MessageView{}, ErrInvalidRequest
		}
		errView, ok := parseErrorView(f.errobj)
		if !ok {
			return MessageView{}, ErrInvalidRequest
		}
		return MessageView{
			Kind:     MessageViewResponseError,
			JSONRPC:  f.jsonrpc,
			IDRaw:    f.id,
			ID:       id,
			ErrorRaw: f.errobj,
			Error:    errView,
		}, nil
	}

	if !f.hasResult {
		return MessageView{}, ErrInvalidRequest
	}
	return MessageView{
		Kind:    MessageViewResponseResult,
		JSONRPC: f.jsonrpc,
		IDRaw:   f.id,
		ID:      id,
		Result:  f.result,
	}, nil
}

func parseIDView(span []byte) (IDView, bool) {
	if len(span) == 0 {
		return IDView{}, false
	}
	switch span[0] {
	case 'n':
		if isNullLiteral(span) {
			return IDView{raw: span}, true
		}
		return IDView{}, false
	case '"':
		body, escaped, ok := jsonStringBody(span)
		if !ok {
			return IDView{}, false
		}
		return IDView{raw: span, str: body, kind: idString, escaped: escaped}, true
	default:
		n, ok := parseInt64Bytes(span)
		if !ok {
			return IDView{}, false
		}
		return IDView{num: n, raw: span, kind: idNumber}, true
	}
}

func parseErrorView(span []byte) (ErrorView, bool) {
	var codeSpan, msgSpan, dataSpan []byte
	if !scanErrorObject(span, &codeSpan, &msgSpan, &dataSpan) {
		return ErrorView{}, false
	}

	out := ErrorView{Raw: span}
	if codeSpan != nil {
		n, ok := parseInt64Bytes(codeSpan)
		if !ok || n < -1<<31 || n >= 1<<31 {
			return ErrorView{}, false
		}
		out.Code = Code(n)
		out.CodeRaw = codeSpan
	}
	if msgSpan != nil {
		msg, escaped, ok := jsonStringBody(msgSpan)
		if !ok {
			return ErrorView{}, false
		}
		out.MessageRaw = msgSpan
		out.MessageBytes = msg
		out.MessageEscaped = escaped
	}
	out.Data = dataSpan
	return out, true
}

// jsonStringBody validates a JSON string span enough for borrowed view parsing
// and returns the unquoted body bytes. If escaped is true, the returned body is
// the raw escaped body, not decoded text.
func jsonStringBody(span []byte) (body []byte, escaped, ok bool) {
	if len(span) < 2 || span[0] != '"' || span[len(span)-1] != '"' {
		return nil, false, false
	}
	body = span[1 : len(span)-1]
	for i := 0; i < len(body); i++ {
		switch body[i] {
		case '"':
			return nil, false, false
		case '\\':
			escaped = true
			i++
			if i >= len(body) {
				return nil, false, false
			}
			switch body[i] {
			case '"', '\\', '/', 'b', 'f', 'n', 'r', 't':
			case 'u':
				if _, ok := readHex4(body[i:]); !ok {
					return nil, false, false
				}
				i += 4
			default:
				return nil, false, false
			}
		}
	}
	return body, escaped, true
}
