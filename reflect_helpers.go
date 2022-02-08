package autohttp

import (
	"context"
	"reflect"
)

var (
	contextType = reflect.TypeOf((*context.Context)(nil)).Elem()
	errorType   = reflect.TypeOf((*error)(nil)).Elem()
	headerType  = reflect.TypeOf(make(Header))
)

func isContextType(t reflect.Type) bool {
	return t.Implements(contextType)
}

func isErrorType(t reflect.Type) bool {
	return t.Implements(errorType)
}

func isHeaderType(t reflect.Type) bool {
	return t == headerType
}
