package autohttp

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/fortytw2/lounge"
)

func TestJSONDecoderValidation(t *testing.T) {
	t.Parallel()

	jsd := NewJSONDecoder()

	cases := []struct {
		Name      string
		Fn        interface{}
		ShouldErr bool
	}{
		{
			"ctx-only",
			func(ctx context.Context) {},
			false,
		},
		{
			"ctx-hdr",
			func(ctx context.Context, h Header) {},
			false,
		},
		{
			"hdr-only",
			func(h Header) {},
			false,
		},
		{
			"struct-only",
			func(in struct{ X int }) {},
			false,
		},
		{
			"ctx-struct",
			func(ctx context.Context, in struct{ X int }) {},
			false,
		},
		{
			"ctx-slice",
			func(ctx context.Context, in []int) {},
			false,
		},
		{
			"slice-only",
			func(in []int) {},
			false,
		},
		{
			"full-args",
			func(ctx context.Context, h Header, in []int) {},
			false,
		},
		{
			"full-args",
			func(ctx context.Context, h Header, in struct{ X int }) {},
			false,
		},
		{
			"duplicate args",
			func(ctx context.Context, ctx2 context.Context) {},
			true,
		},
		{
			"chan",
			func(in chan struct{}) {},
			true,
		},
		{
			"chan-ctx",
			func(ctx context.Context, in chan struct{}) {},
			true,
		},
	}

	for _, c := range cases {
		t.Run(c.Name, func(t *testing.T) {
			err := jsd.ValidateType(c.Fn)
			if err != nil && !c.ShouldErr {
				t.Errorf("case[%s] failed unexpectedly", c.Name)
			}

			if err == nil && c.ShouldErr {
				t.Errorf("case[%s] did not fail when it should have", c.Name)
			}
		})
	}
}

func TestJSONDecoder(t *testing.T) {
	var testFlag bool

	cases := []struct {
		Name         string
		Fn           interface{}
		Body         io.Reader
		ExpectStatus int
		ShouldFlag   bool
	}{

		{
			"basic",
			func(ctx context.Context, input struct {
				Name string
			}) {
				if input.Name == "test" {
					testFlag = true
				}
			},
			strings.NewReader(`{"Name": "test"}`),
			http.StatusNoContent,
			true,
		},
		{
			"basic-bad-args",
			func(ctx context.Context, input struct {
				Name string
			}) {
				if input.Name == "test" {
					testFlag = true
				}
			},
			strings.NewReader(`{"Name": "guten tag"}`),
			http.StatusNoContent,
			false,
		},
		{
			"invalid-json",
			func(ctx context.Context, input struct {
				Name string
			}) {
				if input.Name == "test" {
					testFlag = true
				}
			},
			strings.NewReader(`{"Name": fuk"}`),
			http.StatusBadRequest,
			false,
		},
		{
			"too-big-body",
			func(ctx context.Context, input struct {
				Name string
			}) {
				if input.Name == "test" {
					testFlag = true
				}
			},
			newTooLargeReader(),
			http.StatusRequestEntityTooLarge,
			false,
		},
	}

	for _, c := range cases {
		// reset flag
		testFlag = false

		t.Run(c.Name, func(t *testing.T) {
			ar, err := NewHandler(
				lounge.NewDefaultLog(lounge.WithOutput(os.Stderr)),
				NewJSONDecoder(),
				NoOpEncoder{},
				[]Middleware{},
				DefaultErrorHandler,
				c.Fn)
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

			if c.ShouldFlag {
				if !testFlag {
					t.Error("should have flipped the flag but it did not")
				}
			} else {
				if testFlag {
					t.Error("should not have flipped the flag but it did")
				}
			}
		})
	}
}

func newTooLargeReader() io.Reader {
	var str string
	for i := int64(0); i < DefaultMaxBytesToRead*3; i++ {
		str += "A"
	}

	str += "A"

	return strings.NewReader(`{"Name":"` + str + `"}`)
}
