package autohttp

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/fortytw2/lounge"
)

func TestHandler(t *testing.T) {
	mapReturnFn := func(ctx context.Context, input struct {
		Name string
	}) map[string]string {
		if input.Name == "test" {
			return map[string]string{"test": "booo"}
		}
		return map[string]string{"test": "awooo"}
	}

	mapErrorReturnFn := func(ctx context.Context, input struct {
		Name string
	}) (map[string]string, error) {
		if input.Name == "test" {
			return map[string]string{"test": "booo"}, nil
		}
		return map[string]string{"test": "awooo"}, nil
	}

	cases := []struct {
		Name         string
		Fn           interface{}
		Body         io.Reader
		ExpectStatus int
		ExpectRes    string
	}{
		{
			"only-error",
			func(ctx context.Context, input struct {
				Name string
			}) error {
				if input.Name == "test" {
					return nil
				}
				return errors.New("illegal")
			},
			strings.NewReader(`{"Name": "nah"}`),
			http.StatusInternalServerError,
			`{"error":"illegal"}`,
		},
		{
			"only-struct",
			mapReturnFn,
			strings.NewReader(`{"Name": "nah"}`),
			http.StatusOK,
			`{"test":"awooo"}`,
		},
		{
			"only-struct",
			mapReturnFn,
			strings.NewReader(`{"Name": "test"}`),
			http.StatusOK,
			`{"test":"booo"}`,
		},
		{
			"both-error-and-struct",
			mapErrorReturnFn,
			strings.NewReader(`{"Name": "test"}`),
			http.StatusOK,
			`{"test":"booo"}`,
		},
	}

	for _, c := range cases {
		t.Run(c.Name, func(t *testing.T) {
			ar, err := NewHandler(lounge.NewDefaultLog(lounge.WithOutput(os.Stderr)), NewJSONDecoder(), &JSONEncoder{}, DefaultErrorHandler, c.Fn)
			if err != nil {
				t.Fatal(err)
			}

			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodPost, "/", c.Body)
			r.Header.Set("Content-Type", "application/json")

			ar.ServeHTTP(w, r)

			if w.Code != c.ExpectStatus {
				t.Errorf("expected %d got %d", c.ExpectStatus, w.Code)
			}

			if strings.TrimSpace(w.Body.String()) != string(c.ExpectRes) {
				t.Errorf("json not equals: %q != %q", w.Body.String(), c.ExpectRes)
			}
		})
	}
}
