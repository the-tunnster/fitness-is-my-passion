package ui

import (
	"compress/gzip"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func get(h http.Handler, target, acceptEncoding string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, target, nil)
	if acceptEncoding != "" {
		req.Header.Set("Accept-Encoding", acceptEncoding)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func TestCompressGzipsTextButNotBinary(t *testing.T) {
	h := Compress(StaticHandler())

	rec := get(h, "/static/style.css", "gzip")
	if rec.Header().Get("Content-Encoding") != "gzip" {
		t.Fatal("css should be gzipped")
	}
	zr, err := gzip.NewReader(rec.Body)
	if err != nil {
		t.Fatal(err)
	}
	body, _ := io.ReadAll(zr)
	if !strings.Contains(string(body), "#sidebar") {
		t.Fatal("decompressed css is wrong")
	}

	if rec := get(h, "/static/material-symbols-rounded.woff2", "gzip"); rec.Header().Get("Content-Encoding") != "" {
		t.Fatal("the font is already compressed and should be sent as is")
	}
	if rec := get(h, "/static/style.css", ""); rec.Header().Get("Content-Encoding") != "" {
		t.Fatal("clients that don't accept gzip must get plain data")
	}
}

func TestVersionedAssetsAreCachedLongTerm(t *testing.T) {
	url := assetURL("app.js")
	if !strings.HasPrefix(url, "/static/app.js?v=") {
		t.Fatalf("asset URL has no version: %s", url)
	}
	h := StaticHandler()
	if cc := get(h, url, "").Header().Get("Cache-Control"); !strings.Contains(cc, "immutable") {
		t.Fatalf("versioned asset should be immutable, got %q", cc)
	}
	if cc := get(h, "/static/app.js", "").Header().Get("Cache-Control"); strings.Contains(cc, "immutable") {
		t.Fatalf("unversioned asset must not be immutable, got %q", cc)
	}
}
