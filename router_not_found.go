package httpz

import (
	"io"
	"io/fs"
	"mime"
	"net/http"
	"path/filepath"
	"strings"
)

const requiredUIDir = "ui/dist"

func (r *Router) serveNotFound(w http.ResponseWriter, req *http.Request) {
	var staticFilePath string
	if req.URL.Path == "/" {
		staticFilePath = "ui/dist/index.html"
	} else {
		staticFilePath = filepath.Join(requiredUIDir, req.URL.Path)
	}

	r.log.Debugf("serving path: %s", staticFilePath)
	staticFile, err := r.StaticFiles.Open(staticFilePath)
	if err != nil {
		if err == fs.ErrNotExist {
			staticFile, _ = r.StaticFiles.Open("ui/dist/index.html")
		} else {
			w.WriteHeader(http.StatusNotFound)
			return
		}

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
