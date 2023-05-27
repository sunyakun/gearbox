package storage

import (
	"fmt"
	"net/http"

	"github.com/sunyakun/gearbox/pkg/apis"
)

const (
	ReasonNotFound           = "NotFound"
	ReasonAlreadyExist       = "AlreadyExist"
	ReasonConcurrentConflict = "ConfurrentConflict"
)

type StatusError struct {
	ErrStatus apis.Status
}

func (s StatusError) Error() string {
	return s.ErrStatus.Message
}

func NewNotFoundError(typeName, key string) StatusError {
	return StatusError{
		ErrStatus: apis.Status{
			ObjectMeta: apis.ObjectMeta{Kind: "Status"},
			Status:     apis.StatusFailure,
			Code:       http.StatusNotFound,
			Reason:     ReasonNotFound,
			Message:    fmt.Sprintf("%s %q not found", typeName, key),
		},
	}
}

func IsNotFoundError(err error) bool {
	if e, ok := err.(StatusError); !ok {
		return false
	} else if e.ErrStatus.Reason == ReasonNotFound {
		return true
	}
	return false
}

func NewAlreadyExistError(typeName, key string) StatusError {
	return StatusError{
		ErrStatus: apis.Status{
			ObjectMeta: apis.ObjectMeta{Kind: "Status"},
			Status:     apis.StatusFailure,
			Code:       http.StatusConflict,
			Reason:     ReasonAlreadyExist,
			Message:    fmt.Sprintf("%s %q already exist", typeName, key),
		},
	}
}

func IsAlreadyExistError(err error) bool {
	if e, ok := err.(StatusError); !ok {
		return false
	} else if e.ErrStatus.Reason == ReasonAlreadyExist {
		return true
	}
	return false
}

func NewConcurrentConclictError() StatusError {
	return StatusError{
		ErrStatus: apis.Status{
			ObjectMeta: apis.ObjectMeta{Kind: "Status"},
			Status:     apis.StatusFailure,
			Code:       http.StatusForbidden,
			Reason:     ReasonConcurrentConflict,
			Message:    "the objectVersion in request not equals to the storage version, maybe concurrent conflict",
		},
	}
}

func IsConcurrentConclictError(err error) bool {
	if e, ok := err.(StatusError); !ok {
		return false
	} else if e.ErrStatus.Reason == ReasonConcurrentConflict {
		return true
	}
	return false
}
