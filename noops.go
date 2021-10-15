package httpz

import (
	"errors"
	"io"
	"net/http"
	"reflect"
)

type NoOpDecoder struct{}

func (noop NoOpDecoder) ValidateType(fn interface{}) error {
	if reflect.ValueOf(fn).Type().NumIn() != 0 {
		return errors.New("noop decoder only works for functions with no inputs")
	}

	return nil
}

func (noop NoOpDecoder) Decode(fn interface{}, r *http.Request) ([]reflect.Value, error) {
	return nil, nil
}

type NoOpEncoder struct{}

func (noop NoOpEncoder) ValidateType(fn interface{}) error {
	if reflect.ValueOf(fn).Type().NumOut() != 0 {
		return errors.New("noop encoder only works for functions with no return values")
	}

	return nil
}

func (noop NoOpEncoder) Encode(value interface{}, hw HeaderWriter) (int, io.Reader, error) {
	return http.StatusNoContent, nil, nil
}
