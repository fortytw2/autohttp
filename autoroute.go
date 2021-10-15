package httpz

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
	ErrTooManyInputArgs    = errors.New("autoroute: too many input args")
	ErrTooManyReturnValues = errors.New("autoroute: too many return values")
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
	// Encode cannot write a status code, this is reserved for autoroute itself to control
	// to limit duplicate WriteHeader calls
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

// An Autoroute is an http.Handler generated from any function
type Autoroute struct {
	fn  interface{}
	log lounge.Log

	encoder      Encoder
	decoder      Decoder
	errorHandler ErrorHandler

	hideFromIntrospectors bool
}

func NewAutoroute(
	log lounge.Log,
	decoder Decoder,
	encoder Encoder,
	fn interface{},
) (*Autoroute, error) {
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

	return &Autoroute{
		fn:                    fn,
		encoder:               encoder,
		decoder:               decoder,
		errorHandler:          DefaultErrorHandler,
		hideFromIntrospectors: false,
	}, nil
}

func (ar *Autoroute) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// handle panics
	defer func() {
		if r := recover(); r != nil {
			fmt.Fprintf(w, "panic in route execution: %v", r)
		}
	}()

	callValues, err := ar.decoder.Decode(ar.fn, r)
	if err != nil {
		// encode the parsing error cleanly
		ar.errorHandler(w, err)
		return
	}

	// call the handler function using reflection
	returnValues := reflect.ValueOf(ar.fn).Call(callValues)

	// split out the error value and the return value
	var encodableValue interface{} = nil
	for _, rv := range returnValues {
		if isErrorType(rv.Type()) && !rv.IsNil() {
			err = rv.Interface().(error)
			// encode the parsing error cleanly
			ar.errorHandler(w, err)
			return
		} else {
			encodableValue = rv.Interface()
		}
	}

	responseCode, body, err := ar.encoder.Encode(encodableValue, w.Header().Set)
	if err != nil {
		ar.errorHandler(w, err)
	} else {
		w.WriteHeader(responseCode)
		_, err = io.Copy(w, body)
		if err != nil {
			ar.log.Errorf("error copying response body to writer: %s", err)
		}
	}
}
