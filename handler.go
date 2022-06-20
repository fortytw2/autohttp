package autohttp

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"reflect"

	"github.com/fortytw2/lounge"
)

var (
	ErrTooManyInputArgs    = errors.New("autohttp: too many input args")
	ErrTooManyReturnValues = errors.New("autohttp: too many return values")
)

// 65536
var DefaultMaxBytesToRead int64 = 2 << 15

type Header map[string]string

type HeaderWriter func(key, val string)

type Decoder interface {
	ValidateType(x interface{}) error
	// Decode returns the reflect values needed to call the fn
	// from the *http.Request
	Decode(fn interface{}, r *http.Request) ([]reflect.Value, error)
}

type Encoder interface {
	ValidateType(x interface{}) error
	// Encode cannot write a status code, this is reserved for autohttp to control
	// preventing duplicate WriteHeader calls
	Encode(values interface{}, hw HeaderWriter) (int, io.Reader, error)
}

type ErrorHandler func(w http.ResponseWriter, err error)

func DefaultErrorHandler(w http.ResponseWriter, err error) {
	ewc, ok := err.(ErrorWithCode)
	if ok {
		w.WriteHeader(ewc.StatusCode)
		json.NewEncoder(w).Encode(map[string]string{
			"error": err.Error(),
		})

		return
	}

	w.WriteHeader(http.StatusInternalServerError)
	json.NewEncoder(w).Encode(map[string]string{
		"error": err.Error(),
	})
}

// An Handler is an http.Handler generated from any function
type Handler struct {
	fn  interface{}
	log lounge.Log

	encoder      Encoder
	decoder      Decoder
	errorHandler ErrorHandler
	middlewares  []Middleware

	hideFromIntrospectors bool
}

func NewHandler(
	log lounge.Log,
	decoder Decoder,
	encoder Encoder,
	middlewares []Middleware,
	errorHandler ErrorHandler,
	fn interface{},
) (*Handler, error) {
	if decoder == nil || encoder == nil {
		return nil, errors.New("a decoder and encoder must be supplied. use httpz.NoOpDecoder")
	}

	err := decoder.ValidateType(fn)
	if err != nil {
		return nil, err
	}

	err = encoder.ValidateType(fn)
	if err != nil {
		return nil, err
	}

	// extra autoroute rule
	if reflect.ValueOf(fn).Type().NumOut() > 2 {
		return nil, errors.New("a function can only have up to 2 return values")
	}

	return &Handler{
		fn:                    fn,
		encoder:               encoder,
		decoder:               decoder,
		errorHandler:          DefaultErrorHandler,
		middlewares:           middlewares,
		hideFromIntrospectors: false,
	}, nil
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// handle panics
	defer func() {
		if r := recover(); r != nil {
			fmt.Fprintf(w, "panic in route execution: %v", r)
		}
	}()

	for _, mw := range h.middlewares {
		err := mw.Before(r, h)
		if err != nil {
			h.errorHandler(w, err)
			return
		}
	}

	callValues, err := h.decoder.Decode(h.fn, r)
	if err != nil {
		// encode the parsing error cleanly
		h.errorHandler(w, err)
		return
	}

	// call the handler function using reflection
	returnValues := reflect.ValueOf(h.fn).Call(callValues)

	// split out the error value and the return value
	var encodableValue interface{} = nil
	for _, rv := range returnValues {
		if isErrorType(rv.Type()) && !rv.IsNil() && !rv.IsZero() {
			err = rv.Interface().(error)
			// encode the parsing error cleanly
			h.errorHandler(w, err)
			return
		} else if !isErrorType(rv.Type()) {
			encodableValue = rv.Interface()
		}
	}

	responseCode, body, err := h.encoder.Encode(encodableValue, w.Header().Set)
	if err != nil {
		h.errorHandler(w, err)
	} else {
		w.WriteHeader(responseCode)
		_, err = io.Copy(w, body)
		if err != nil {
			h.log.Errorf("error copying response body to writer: %s", err)
		}
	}
}
