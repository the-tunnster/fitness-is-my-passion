package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestBasicAuth(t *testing.T) {
	h := BasicAuth("me", "secret", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	cases := []struct {
		name, user, pass string
		set              bool
		want             int
	}{
		{"no credentials", "", "", false, http.StatusUnauthorized},
		{"wrong password", "me", "nope", true, http.StatusUnauthorized},
		{"wrong user", "you", "secret", true, http.StatusUnauthorized},
		{"correct", "me", "secret", true, http.StatusOK},
	}
	for _, c := range cases {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		if c.set {
			req.SetBasicAuth(c.user, c.pass)
		}
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		if rec.Code != c.want {
			t.Errorf("%s: got %d, want %d", c.name, rec.Code, c.want)
		}
		if c.want == http.StatusUnauthorized && rec.Header().Get("WWW-Authenticate") == "" {
			t.Errorf("%s: missing WWW-Authenticate header", c.name)
		}
	}
}
