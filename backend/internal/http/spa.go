package http

import (
	"net/http"
	"path"
	"strings"
)

type spaHandler struct {
	files  http.FileSystem
	server http.Handler
}

func newSPAHandler(dir string) http.Handler {
	files := http.Dir(dir)
	return &spaHandler{files: files, server: http.FileServer(files)}
}

func (h *spaHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	name := strings.TrimPrefix(path.Clean(r.URL.Path), "/")
	if name != "." && name != "" {
		if file, err := h.files.Open(name); err == nil {
			info, statErr := file.Stat()
			_ = file.Close()
			if statErr == nil && !info.IsDir() {
				h.server.ServeHTTP(w, r)
				return
			}
		}
	}

	request := r.Clone(r.Context())
	requestURL := *r.URL
	requestURL.Path = "/"
	request.URL = &requestURL
	h.server.ServeHTTP(w, request)
}
