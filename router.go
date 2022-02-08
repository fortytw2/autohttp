package autohttp

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"io/ioutil"
	"net/http"

	"github.com/fortytw2/httpz/internal/httpsnoop"
	"github.com/fortytw2/lounge"
)

type embeddedAssets struct {
	indexBytes []byte
	staticDir  fs.FS
}

func newEmbeddedAssets(assets fs.FS, htmlPath string, staticDir string) (*embeddedAssets, error) {
	file, err := assets.Open(htmlPath)
	if err != nil {
		return nil, err
	}

	indexHTMLBytes, err := ioutil.ReadAll(file)
	if err != nil {
		return nil, err
	}

	staticFS, err := fs.Sub(assets, staticDir)
	if err != nil {
		return nil, err
	}

	return &embeddedAssets{
		indexBytes: indexHTMLBytes,
		staticDir:  staticFS,
	}, nil
}

type Router struct {
	Routes map[string]map[string]*Handler

	embeddedAssets *embeddedAssets

	log lounge.Log

	enableHSTS         bool
	enableRouteMetrics bool

	defaultEncoder      Encoder
	defaultDecoder      Decoder
	defaultErrorHandler ErrorHandler
}

type RouterOption func(r *Router) error

func EnableHSTS(r *Router) error {
	r.enableHSTS = true
	return nil
}

func EnableRouteMetrics(r *Router) error {
	r.enableRouteMetrics = true
	return nil
}

func WithEmbeddedAssets(assets fs.FS, indexHTMLPath string, staticDirPath string) func(r *Router) error {
	return func(r *Router) error {
		ea, err := newEmbeddedAssets(assets, indexHTMLPath, staticDirPath)
		if err != nil {
			return err
		}

		r.embeddedAssets = ea
		return nil
	}
}

func WithDefaultEncoder(e Encoder) func(r *Router) error {
	return func(r *Router) error {
		r.defaultEncoder = e
		return nil
	}
}

func WithDefaultDecoder(d Decoder) func(r *Router) error {
	return func(r *Router) error {
		r.defaultDecoder = d
		return nil
	}
}

func WithDefaultErrorHandler(eh ErrorHandler) func(r *Router) error {
	return func(r *Router) error {
		r.defaultErrorHandler = eh
		return nil
	}
}

var DefaultOptions = []RouterOption{
	WithDefaultDecoder(NewJSONDecoder()),
	WithDefaultEncoder(&JSONEncoder{}),
}

func NewRouter(log lounge.Log, routerOptions ...RouterOption) (*Router, error) {
	r := &Router{log: log, Routes: make(map[string]map[string]*Handler)}
	for _, ro := range append(DefaultOptions, routerOptions...) {
		err := ro(r)
		if err != nil {
			return nil, err
		}
	}

	return r, nil

}

var validMethods = map[string]bool{
	http.MethodGet:    true,
	http.MethodDelete: true,
	http.MethodPatch:  true,
	http.MethodPost:   true,
	http.MethodPut:    true,
}

func (r *Router) Register(method string, path string, fn interface{}) error {
	if ok := validMethods[method]; !ok {
		return fmt.Errorf("invalid http method: %s", method)
	}

	_, ok := r.Routes[method]
	if !ok {
		r.Routes[method] = make(map[string]*Handler)
	}

	_, ok = r.Routes[method][path]
	if ok {
		return errors.New("route already registered")
	}

	h, err := NewHandler(r.log, r.defaultDecoder, r.defaultEncoder, r.defaultErrorHandler, fn)
	if err != nil {
		return err
	}

	r.Routes[method][path] = h

	return nil
}

func (r *Router) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	if r.enableRouteMetrics {
		m := httpsnoop.CaptureMetrics(http.HandlerFunc(r.internalServeHTTP), w, req)
		r.log.Debugf("served %d bytes for %s %s in %s with code %d", m.Written, req.Method, req.URL.Path, m.Duration, m.Code)

		return
	}

	r.internalServeHTTP(w, req)
}

func (r *Router) internalServeHTTP(w http.ResponseWriter, req *http.Request) {
	if req.Method == http.MethodOptions {
		return
	}

	routes, ok := r.Routes[req.URL.Path]
	if !ok {
		r.serveNotFound(w, req)
		r.cleanLeftovers(req)
		return
	}

	route, ok := routes[req.Method]
	if !ok {
		if req.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		r.serveNotFound(w, req)
		r.cleanLeftovers(req)
		return
	}

	route.ServeHTTP(w, req)
	r.cleanLeftovers(req)
}

// this is a bit of weirdness from production on Heroku
// some reverse proxies get really upset if you don't read
// the entire request body, and sometimes that happens to us here
func (r *Router) cleanLeftovers(req *http.Request) {
	if req.Body == nil || req.Body == http.NoBody {
		// do nothing
	} else {
		// chew up the rest of the body
		var buf bytes.Buffer
		buf.ReadFrom(req.Body)
		req.Body.Close()
	}
}
