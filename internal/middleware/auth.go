package middleware

import (
	"context"
	"crypto/subtle"
	"net/http"

	"fitness-tracker/internal/utils"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// BasicAuth protects every request with HTTP Basic credentials. Only use it over HTTPS.
func BasicAuth(user, pass string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		u, p, ok := r.BasicAuth()
		// Both comparisons always run so timing doesn't reveal which one failed.
		userOK := subtle.ConstantTimeCompare([]byte(u), []byte(user)) == 1
		passOK := subtle.ConstantTimeCompare([]byte(p), []byte(pass)) == 1
		if !ok || !userOK || !passOK {
			w.Header().Set("WWW-Authenticate", `Basic realm="fitness", charset="UTF-8"`)
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}

type ctxKey string

const UserIDKey ctxKey = "user_id"

// RequireUser ensures requests include a valid user_id (header or query) and
// attaches it to the request context. If missing/invalid, responds 401.
func RequireUser(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userID := r.Header.Get("X-User-ID")
		if userID == "" {
			userID = r.URL.Query().Get("user_id")
		}
		if userID == "" {
			utils.ErrorResponse(w, http.StatusUnauthorized, "Unauthorized: missing user_id")
			return
		}
		if _, err := primitive.ObjectIDFromHex(userID); err != nil {
			utils.ErrorResponse(w, http.StatusUnauthorized, "Unauthorized: invalid user_id")
			return
		}
		ctx := context.WithValue(r.Context(), UserIDKey, userID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
