package apperr

import (
	"fmt"
	"net/http"
)

// Kind 表示业务错误类型。
type Kind string

const (
	KindInvalidArgument    Kind = "invalid_argument"
	KindNotFound           Kind = "not_found"
	KindConflict           Kind = "conflict"
	KindUnauthorized       Kind = "unauthorized"
	KindForbidden          Kind = "forbidden"
	KindFailedPrecondition Kind = "failed_precondition"
	KindUnavailable        Kind = "unavailable"
	KindInternal           Kind = "internal"
)

// HTTPStatus 返回与错误类型对应的 HTTP 状态码。
func (k Kind) HTTPStatus() int {
	switch k {
	case KindInvalidArgument:
		return http.StatusBadRequest
	case KindNotFound:
		return http.StatusNotFound
	case KindConflict:
		return http.StatusConflict
	case KindUnauthorized:
		return http.StatusUnauthorized
	case KindForbidden:
		return http.StatusForbidden
	case KindFailedPrecondition:
		return http.StatusPreconditionFailed
	case KindUnavailable:
		return http.StatusServiceUnavailable
	case KindInternal:
		return http.StatusInternalServerError
	default:
		return http.StatusInternalServerError
	}
}

// Error 业务错误。
type Error struct {
	Kind  Kind
	Msg   string
	Cause error
}

// New 创建业务错误。
func New(kind Kind, msg string) *Error {
	return &Error{Kind: kind, Msg: msg}
}

// Wrap 创建带底层错误的业务错误。
func Wrap(kind Kind, msg string, cause error) *Error {
	return &Error{Kind: kind, Msg: msg, Cause: cause}
}

// Error 返回错误描述。
func (e *Error) Error() string {
	if e == nil {
		return ""
	}
	if e.Cause == nil {
		return e.Msg
	}
	if e.Msg == "" {
		return e.Cause.Error()
	}
	return fmt.Sprintf("%s: %v", e.Msg, e.Cause)
}

// Unwrap 返回底层错误。
func (e *Error) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Cause
}

// Is 只比较错误类型，方便 errors.Is 统一判断。
func (e *Error) Is(target error) bool {
	if e == nil || target == nil {
		return false
	}
	t, ok := target.(*Error)
	if !ok {
		return false
	}
	return e.Kind == t.Kind
}

// Status 返回对应的 HTTP 状态码。
func (e *Error) Status() int {
	if e == nil {
		return http.StatusInternalServerError
	}
	return e.Kind.HTTPStatus()
}

// Code 返回响应体中的业务码。
func (e *Error) Code() int {
	return e.Status()
}

// BadRequest 创建 400 错误。
func BadRequest(msg string) *Error { return New(KindInvalidArgument, msg) }

// NotFound 创建 404 错误。
func NotFound(msg string) *Error { return New(KindNotFound, msg) }

// Conflict 创建 409 错误。
func Conflict(msg string) *Error { return New(KindConflict, msg) }

// Unauthorized 创建 401 错误。
func Unauthorized(msg string) *Error { return New(KindUnauthorized, msg) }

// Forbidden 创建 403 错误。
func Forbidden(msg string) *Error { return New(KindForbidden, msg) }

// FailedPrecondition 创建 412 错误。
func FailedPrecondition(msg string) *Error { return New(KindFailedPrecondition, msg) }

// Unavailable 创建 503 错误。
func Unavailable(msg string) *Error { return New(KindUnavailable, msg) }

// Internal 创建 500 错误。
func Internal(msg string) *Error { return New(KindInternal, msg) }

var _ error = (*Error)(nil)
