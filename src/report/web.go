package main

import (
	"io/fs"
	"net/http"
	"os"
	"strings"
)

func handler(output string, photos, results fs.FS, examples []string) http.Handler {
	known := make(map[string]bool, len(examples))
	for _, name := range examples {
		known[name] = true
	}
	files := map[string]string{
		"/":             "index.html",
		"/index.html":   "index.html",
		"/app.js":       "app.js",
		"/quality.js":   "quality.js",
		"/style.css":    "style.css",
		"/data.json":    "data.json",
		"/quality.json": "quality.json",
	}
	images := map[string]bool{"lsd.png": true, "ransac.png": true, "tsp.png": true}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			w.Header().Set("Allow", "GET, HEAD")
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		w.Header().Set("X-Content-Type-Options", "nosniff")
		if file, ok := files[r.URL.Path]; ok {
			w.Header().Set("Cache-Control", "no-cache")
			http.ServeFileFS(w, r, os.DirFS(output), file)
			return
		}
		parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/"), "/")
		if len(parts) != 3 || parts[0] != "images" || !known[parts[1]] {
			http.NotFound(w, r)
			return
		}
		name, image := parts[1], parts[2]
		if image == "original.jpg" {
			http.ServeFileFS(w, r, photos, name+"/"+name+".jpg")
		} else if images[image] {
			http.ServeFileFS(w, r, results, name+"/"+image)
		} else {
			http.NotFound(w, r)
		}
	})
}
