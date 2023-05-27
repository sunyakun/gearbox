package errors

import (
	"fmt"
	"net/http"

	"github.com/sunyakun/gearbox/pkg/apis"
)

type APIStatus interface {
	Status() apis.Status
}

type StatusError struct {
	ErrStatus apis.Status
}

func (s StatusError) Status() apis.Status {
	return s.ErrStatus
}

func (s StatusError) Error() string {
	return s.ErrStatus.Message
}

func getErrorCodeAndReason(err error) (int, string) {
	if e, ok := err.(StatusError); ok {
		return e.Status().Code, e.Status().Reason
	}
	return 0, ""
}

func NewBadRequest(message string) StatusError {
	return StatusError{
		ErrStatus: apis.Status{
			ObjectMeta: apis.ObjectMeta{Kind: "Status"},
			Code:       http.StatusBadRequest,
			Status:     apis.StatusFailure,
			Reason:     http.StatusText(http.StatusBadRequest),
			Message:    message,
		},
	}
}

func NewNotFound(kind, key string) StatusError {
	return StatusError{
		ErrStatus: apis.Status{
			ObjectMeta: apis.ObjectMeta{Kind: "Status"},
			Code:       http.StatusNotFound,
			Status:     apis.StatusFailure,
			Reason:     http.StatusText(http.StatusNotFound),
			Message:    fmt.Sprintf("%s %q not found", kind, key),
		},
	}
}

func NewForbidden(operate, kind, key, message string) StatusError {
	return StatusError{
		ErrStatus: apis.Status{
			ObjectMeta: apis.ObjectMeta{Kind: "Status"},
			Code:       http.StatusForbidden,
			Status:     apis.StatusFailure,
			Reason:     http.StatusText(http.StatusForbidden),
			Message:    fmt.Sprintf("%s %s %q is forbidden: %s", operate, kind, key, message),
		},
	}
}

func NewConflict(err error) StatusError {
	return StatusError{
		ErrStatus: apis.Status{
			ObjectMeta: apis.ObjectMeta{Kind: "Status"},
			Code:       http.StatusConflict,
			Status:     apis.StatusFailure,
			Reason:     http.StatusText(http.StatusConflict),
			Message:    err.Error(),
		},
	}
}

func NewInternalError(err error) StatusError {
	return StatusError{
		ErrStatus: apis.Status{
			ObjectMeta: apis.ObjectMeta{Kind: "Status"},
			Code:       http.StatusInternalServerError,
			Status:     apis.StatusFailure,
			Reason:     http.StatusText(http.StatusInternalServerError),
			Message:    err.Error(),
		},
	}
}

func IsBadRequestError(err error) bool {
	if code, _ := getErrorCodeAndReason(err); code == http.StatusBadRequest {
		return true
	}
	return false
}

func IsNotFoundError(err error) bool {
	if code, _ := getErrorCodeAndReason(err); code == http.StatusNotFound {
		return true
	}
	return false
}

func IsForbiddenError(err error) bool {
	if code, _ := getErrorCodeAndReason(err); code == http.StatusForbidden {
		return true
	}
	return false
}

func IsConflictError(err error) bool {
	if code, _ := getErrorCodeAndReason(err); code == http.StatusConflict {
		return true
	}
	return false
}

func IsInternalError(err error) bool {
	if code, _ := getErrorCodeAndReason(err); code == http.StatusInternalServerError {
		return true
	}
	return false
}
