package autohttp

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"strings"
)

const (
	maxJSONDecoderInputArgs = 3
	// unknown index
	uIdx = -1
)

type JSONDecoder struct {
	MaxBytesToRead        int64
	DisallowUnknownFields bool
}

func NewJSONDecoder() *JSONDecoder {
	return &JSONDecoder{
		MaxBytesToRead:        DefaultMaxBytesToRead,
		DisallowUnknownFields: true,
	}
}

func (jsd *JSONDecoder) ValidateType(fn interface{}) error {
	_, _, _, err := jsd.inputsAtIndices(fn)
	return err
}

func (jsd *JSONDecoder) inputsAtIndices(fn interface{}) (int, int, int, error) {
	reflectFn := reflect.ValueOf(fn)

	inputArgCount := reflectFn.Type().NumIn()
	if inputArgCount > maxJSONDecoderInputArgs {
		return uIdx, uIdx, uIdx, ErrTooManyInputArgs
	}

	foundCtxIdx := uIdx
	foundHeaderIdx := uIdx
	foundDecodeTargetIdx := uIdx
	for i := 0; i < inputArgCount; i++ {
		typeAtInputIdx := reflectFn.Type().In(i)

		if isContextType(typeAtInputIdx) {
			if foundCtxIdx != uIdx {
				return uIdx, uIdx, uIdx, ErrDuplicateType
			}

			if i != 0 {
				return uIdx, uIdx, uIdx, errTypeInvalidAtIndex(i, typeAtInputIdx)
			}

			foundCtxIdx = i
		}

		if isHeaderType(typeAtInputIdx) {
			if foundHeaderIdx != uIdx {
				return uIdx, uIdx, uIdx, ErrDuplicateType
			}

			// header info is only valid as the first or second argument
			if !(i == 0 || i == 1) {
				return uIdx, uIdx, uIdx, errTypeInvalidAtIndex(i, typeAtInputIdx)
			}

			foundHeaderIdx = i
		}

		if jsd.isJSONDecodable(typeAtInputIdx) {
			if foundDecodeTargetIdx != uIdx {
				return uIdx, uIdx, uIdx, ErrDuplicateType
			}

			foundDecodeTargetIdx = i
		}
	}

	var totalFound int
	if foundCtxIdx != uIdx {
		totalFound++
	}
	if foundHeaderIdx != uIdx {
		totalFound++
	}
	if foundDecodeTargetIdx != uIdx {
		totalFound++
	}

	if totalFound != inputArgCount {
		return uIdx, uIdx, uIdx, errors.New("invalid arguments found")
	}

	return foundCtxIdx, foundHeaderIdx, foundDecodeTargetIdx, nil
}

func (jsd *JSONDecoder) isJSONDecodable(t reflect.Type) bool {
	kind := t.Kind()

	// autoroute.Header is not JSON Decodable
	return !isHeaderType(t) && (kind == reflect.Ptr || kind == reflect.Map || kind == reflect.Slice || kind == reflect.Struct)
}

// Decode returns the reflect values needed to call the fn
// from the *http.Request
func (jsd *JSONDecoder) Decode(fn interface{}, r *http.Request) ([]reflect.Value, error) {
	if !strings.Contains(r.Header.Get("Content-Type"), "application/json") {
		return nil, ErrorWithCode{Err: errors.New("invalid mime type"), StatusCode: http.StatusUnsupportedMediaType}
	}

	if r.Method == http.MethodGet {
		return nil, ErrorWithCode{Err: errors.New("GET requests prohibited for this endpoint"), StatusCode: http.StatusMethodNotAllowed}
	}

	limitedReader := io.LimitReader(r.Body, int64(jsd.MaxBytesToRead))
	dec := json.NewDecoder(limitedReader)
	if jsd.DisallowUnknownFields {
		dec.DisallowUnknownFields()
	}

	ctxIdx, hdrIdx, decodeIdx, err := jsd.inputsAtIndices(fn)
	if err != nil {
		return nil, err
	}

	fnReflectType := reflect.ValueOf(fn).Type()
	callValues := make([]reflect.Value, fnReflectType.NumIn())

	if ctxIdx != uIdx {
		callValues[ctxIdx] = reflect.ValueOf(r.Context())
	}

	// add the httpz.Header to the call args
	if hdrIdx != uIdx {
		header := make(Header)
		for k := range r.Header {
			hVal := r.Header.Get(k)
			header[http.CanonicalHeaderKey(k)] = hVal
		}

		callValues[hdrIdx] = reflect.ValueOf(header)
	}

	// JSON decode and add to call values
	if decodeIdx != uIdx {
		inArg := fnReflectType.In(decodeIdx)

		var object reflect.Value

		switch inArg.Kind() {
		case reflect.Ptr:
			object = reflect.New(inArg.Elem())
		default:
			object = reflect.New(inArg)
		}

		oi := object.Interface()
		err = dec.Decode(&oi)
		if err != nil {
			if err == io.ErrUnexpectedEOF {
				return nil, ErrorWithCode{Err: fmt.Errorf("maximum body size exceeded (%d bytes)", jsd.MaxBytesToRead), StatusCode: http.StatusRequestEntityTooLarge}
			}
			return nil, ErrorWithCode{Err: err, StatusCode: http.StatusBadRequest}
		}

		switch inArg.Kind() {
		case reflect.Struct:
			callValues[decodeIdx] = reflect.ValueOf(oi).Elem()
		default:
			callValues[decodeIdx] = reflect.ValueOf(oi)
		}
	}

	return callValues, nil
}
