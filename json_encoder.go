package httpz

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
)

type JSONEncoder struct{}

func (jse *JSONEncoder) ValidateType(fn interface{}) error {
	return nil
}

func (jse *JSONEncoder) Encode(value interface{}, hw HeaderWriter) (int, io.Reader, error) {
	hw("Content-Type", "application/json")

	var b bytes.Buffer
	err := json.NewEncoder(&b).Encode(value)
	if err != nil {
		return http.StatusInternalServerError, nil, err
	}

	return http.StatusOK, &b, nil
}
