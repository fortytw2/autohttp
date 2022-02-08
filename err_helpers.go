package autohttp

import (
	"errors"
	"reflect"
)

var ErrDuplicateType = errors.New("httpz: duplicate type in input args")

func errTypeInvalidAtIndex(idx int, t reflect.Type) error {
	return errors.New("type is invalid at input idx")
}

type ErrorWithCode struct {
	Err        error
	StatusCode int
}

func NewErrorWithCode(err error, code int) error {
	return &ErrorWithCode{
		Err:        err,
		StatusCode: code,
	}
}

func (ewc ErrorWithCode) Error() string {
	return ewc.Err.Error()
}
