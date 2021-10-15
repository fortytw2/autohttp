package httpz

import (
	"errors"
	"fmt"
	"io/fs"
	"net/http"
	"net/url"

	"github.com/fortytw2/httpz/internal/csp"
	"github.com/fortytw2/lounge"
)

type Router struct {
	Routes map[string]map[string]*Autoroute

	StaticFiles fs.FS

	log    lounge.Log
	origin *url.URL

	enableHSTS    bool
	redirectHTTPS bool

	contentSecurityPolicy *csp.Policy
}

type RouterOption func(r *Router) error

func EnableHSTS(r *Router) error {
	r.enableHSTS = true
	return nil
}

var DefaultOptions = []RouterOption{}

func NewRouter(log lounge.Log, staticFiles fs.FS, defaultOptions ...RouterOption) (*Router, error) {
	return &Router{log: log, StaticFiles: staticFiles, Routes: make(map[string]map[string]*Autoroute)}, nil
}

var validMethods = map[string]bool{
	http.MethodGet:    true,
	http.MethodDelete: true,
	http.MethodPatch:  true,
	http.MethodPost:   true,
	http.MethodPut:    true,
}

func (r *Router) Register(method string, path string, fn interface{}, options ...RouterOption) error {
	if ok := validMethods[method]; !ok {
		return fmt.Errorf("invalid http method: %s", method)
	}

	_, ok := r.Routes[method]
	if !ok {
		r.Routes[method] = make(map[string]*Autoroute)
	}

	_, ok = r.Routes[method][path]
	if ok {
		return errors.New("route already registered")
	}

	// NewAutoroute(r.log)

	// r.Routes[method][path] = fn

	return nil
}

func (r *Router) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	if req.Method == http.MethodOptions {
		return
	}

	routes, ok := r.Routes[req.URL.Path]
	if !ok {
		r.serveNotFound(w, req)
		return
	}

	route, ok := routes[req.Method]
	if !ok {
		if req.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		r.serveNotFound(w, req)
		return
	}

	route.ServeHTTP(w, req)
}
