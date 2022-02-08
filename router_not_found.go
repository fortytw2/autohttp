package autohttp

import (
	"net/http"
)

func (r *Router) serveNotFound(w http.ResponseWriter, req *http.Request) {
	if r.embeddedAssets == nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	// Handling the FileServer Code Snippet was taken from here:
	// https://golang.org/pkg/embed/#hdr-File_Systems
	embeddedFileServer := http.FileServer(http.FS(r.embeddedAssets.staticDir))
	embeddedFileServer.ServeHTTP(w, req)
}
