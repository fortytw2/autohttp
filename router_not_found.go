package autohttp

import (
	"errors"
	"io/fs"
	"net/http"
)

// https://www.alexedwards.net/blog/disable-http-fileserver-directory-listings#using-a-custom-filesystem
type indexOnNotFoundFS struct {
	fs fs.FS
}

func (nfs indexOnNotFoundFS) Open(path string) (fs.File, error) {
	f, err := nfs.fs.Open(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nfs.fs.Open("index.html")
		}

		return nil, err
	}

	return f, nil
}

func (r *Router) serveNotFound(w http.ResponseWriter, req *http.Request) {
	if r.embeddedAssets == nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	// Handling the FileServer Code Snippet was taken from here:
	// https://golang.org/pkg/embed/#hdr-File_Systems
	nfs := indexOnNotFoundFS{fs: r.embeddedAssets.staticDir}

	embeddedFileServer := http.FileServer(http.FS(nfs))
	embeddedFileServer.ServeHTTP(w, req)
}
