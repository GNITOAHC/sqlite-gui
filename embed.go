package svelte

import (
	"embed"
	"io"
	"io/fs"
	"net/http"
	"path/filepath"
	"strings"
)

//go:embed all:ui/build
var SSGFiles embed.FS

func CleanHTML(embedFS fs.FS) http.Handler {
    fileServer := http.FileServer(http.FS(embedFS))

    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        path := strings.TrimPrefix(r.URL.Path, "/")

        // 1. Try serving the file exactly as requested
        if f, err := embedFS.Open(path); err == nil {
            info, _ := f.Stat()
            http.ServeContent(w, r, path, info.ModTime(), f.(io.ReadSeeker))
            return
        }

        // 2. If no extension, try clean-URL: /foo → /foo.html
        if !strings.Contains(filepath.Base(path), ".") {
            htmlPath := path + ".html"
            if f, err := embedFS.Open(htmlPath); err == nil {
                info, _ := f.Stat()
                http.ServeContent(w, r, htmlPath, info.ModTime(), f.(io.ReadSeeker))
                return
            }
        }

        // 3. Otherwise fall back to normal file server
        fileServer.ServeHTTP(w, r)
    })
}

