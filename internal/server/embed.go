package server

import (
	"io/fs"
	"net/http"
	"strings"

	"github.com/alexandrmotologa/codekernel/web"
)

func (s *HTTPServer) handleStatic(w http.ResponseWriter, r *http.Request) {
	// If path starts with /api/, do not serve static
	if strings.HasPrefix(r.URL.Path, "/api/") {
		http.NotFound(w, r)
		return
	}

	distSub, err := fs.Sub(web.DistFS, "dist")
	if err != nil {
		http.Error(w, "Embedded frontend build missing", http.StatusInternalServerError)
		return
	}

	path := strings.TrimPrefix(r.URL.Path, "/")
	if path == "" {
		path = "index.html"
	}

	// Check if specific file exists
	f, err := distSub.Open(path)
	if err != nil {
		// Fallback to index.html for client-side SPA routing
		indexFile, err := distSub.Open("index.html")
		if err != nil {
			http.Error(w, "index.html not found in embedded dist", http.StatusNotFound)
			return
		}
		defer indexFile.Close()
		stat, _ := indexFile.Stat()
		http.ServeContent(w, r, "index.html", stat.ModTime(), indexFile.(interface {
			Read([]byte) (int, error)
			Seek(int64, int) (int64, error)
		}))
		return
	}
	defer f.Close()

	fileServer := http.FileServer(http.FS(distSub))
	fileServer.ServeHTTP(w, r)
}
