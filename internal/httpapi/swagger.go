package httpapi

import (
	"embed"
	"io/fs"
	"net/http"
	"os"
)

//go:embed static/*
var staticFiles embed.FS

func serveSwagger() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/docs" || r.URL.Path == "/docs/" {
			http.Redirect(w, r, "/docs/index.html", http.StatusMovedPermanently)
			return
		}

		// Try to serve openapi.yaml from root directory first
		if r.URL.Path == "/docs/openapi.yaml" {
			data, err := os.ReadFile("openapi.yaml")
			if err == nil {
				w.Header().Set("Content-Type", "application/yaml")
				w.Write(data)
				return
			}
		}

		// Serve embedded static files
		fsys, err := fs.Sub(staticFiles, "static")
		if err != nil {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		http.StripPrefix("/docs/", http.FileServer(http.FS(fsys))).ServeHTTP(w, r)
	})
}
