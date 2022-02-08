package autohttp

import (
	"io"
	"io/fs"
	"mime"
	"net/http"
	"strings"
)

func (r *Router) serveNotFound(w http.ResponseWriter, req *http.Request) {
	if r.embeddedAssets == nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	r.log.Debugf("serving embedded asset path: %s", req.URL.Path)

	// hot path for index html
	if req.URL.Path == "/" {
		r.serveIndexHTML(w, req)
		return
	}

	// req.URL.Path == "/static/*"
	filePath := strings.TrimPrefix(req.URL.Path, "/static/")
	staticFile, err := r.embeddedAssets.staticDir.Open(filePath)
	if err != nil {
		if err == fs.ErrNotExist {
			r.serveIndexHTML(w, req)
			return
		}
	} else {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	extension := strings.Split(req.URL.Path, ".")
	if len(extension) > 1 {
		mimeType := mime.TypeByExtension("." + extension[len(extension)-1])
		w.Header().Set("Content-Type", mimeType)
	}

	_, err = io.Copy(w, staticFile)
	if err != nil {
		r.log.Errorf("could not serve static file: %s", err)
	}
}

func (r *Router) serveIndexHTML(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Content-Type", "application/html")
	w.WriteHeader(http.StatusOK)
	_, err := w.Write(r.embeddedAssets.indexBytes)
	if err != nil {
		r.log.Errorf("could not serve static file: %s", err)
	}
}
