package ui

import (
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"io/fs"
	"net/http"
	"strings"
	"sync"
)

// Compress gzips text responses (HTML, CSS, JS, JSON, SVG) for clients that accept it.
// Over a slow link, compressing is the difference between a quick and a sluggish page.
func Compress(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
			next.ServeHTTP(w, r)
			return
		}
		w.Header().Add("Vary", "Accept-Encoding")
		gw := &gzipWriter{ResponseWriter: w}
		defer gw.Close()
		next.ServeHTTP(gw, r)
	})
}

type gzipWriter struct {
	http.ResponseWriter
	gz          *gzip.Writer
	wroteHeader bool
}

func compressible(contentType string) bool {
	return strings.HasPrefix(contentType, "text/") ||
		strings.Contains(contentType, "javascript") ||
		strings.Contains(contentType, "json") ||
		strings.Contains(contentType, "svg")
}

func (g *gzipWriter) WriteHeader(code int) {
	if g.wroteHeader {
		return
	}
	g.wroteHeader = true
	h := g.Header()
	// Partial and empty responses are left alone.
	if code == http.StatusOK && h.Get("Content-Encoding") == "" && compressible(h.Get("Content-Type")) {
		h.Del("Content-Length")
		h.Set("Content-Encoding", "gzip")
		g.gz = gzip.NewWriter(g.ResponseWriter)
	}
	g.ResponseWriter.WriteHeader(code)
}

func (g *gzipWriter) Write(b []byte) (int, error) {
	if !g.wroteHeader {
		if g.Header().Get("Content-Type") == "" {
			g.Header().Set("Content-Type", http.DetectContentType(b))
		}
		g.WriteHeader(http.StatusOK)
	}
	if g.gz != nil {
		return g.gz.Write(b)
	}
	return g.ResponseWriter.Write(b)
}

func (g *gzipWriter) Close() {
	if g.gz != nil {
		g.gz.Close()
	}
}

var (
	assetVersions   = map[string]string{}
	assetVersionsMu sync.Mutex
)

// assetURL returns the URL of a static file with a short content hash appended, so the
// browser can cache it for a long time and still pick up a changed file immediately.
func assetURL(name string) string {
	assetVersionsMu.Lock()
	defer assetVersionsMu.Unlock()
	v, ok := assetVersions[name]
	if !ok {
		if b, err := fs.ReadFile(assets, "static/"+name); err == nil {
			sum := sha256.Sum256(b)
			v = hex.EncodeToString(sum[:4])
		}
		assetVersions[name] = v
	}
	if v == "" {
		return "/static/" + name
	}
	return "/static/" + name + "?v=" + v
}
