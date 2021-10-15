package httpz

import (
	"context"
	"errors"
	"reflect"
	"testing"
)

func TestIsContextType(t *testing.T) {
	t.Parallel()

	ctxType := reflect.ValueOf(context.Background()).Type()
	if !isContextType(ctxType) {
		t.Fatal("context.Background() is not context type")
	}
}

func TestIsErrorType(t *testing.T) {
	t.Parallel()

	errType := reflect.ValueOf(errors.New("test")).Type()
	if !isErrorType(errType) {
		t.Fatal("errors.New() is not error type")
	}

	errType2 := reflect.ValueOf(ErrorWithCode{Err: errors.New("test")}).Type()
	if !isErrorType(errType2) {
		t.Fatal("errors.New() is not error type")
	}
}

func TestIsHeaderType(t *testing.T) {
	t.Parallel()

	headerType := reflect.ValueOf(make(Header)).Type()
	if !isHeaderType(headerType) {
		t.Fatal("make(Header) is not header type")
	}
}
