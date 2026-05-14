package httpapi

import (
	"errors"
	"net/http"
	"strings"
)

func contentTypeMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost || r.Method == http.MethodPut {
			// For bodyless POST/PUT endpoints, do not require Content-Type.
			if r.ContentLength == 0 {
				next.ServeHTTP(w, r)
				return
			}

			contentType := r.Header.Get("Content-Type")
			if !strings.HasPrefix(contentType, "application/json") {
				writeError(w, http.StatusUnsupportedMediaType,
					errors.New("Content-Type must be application/json"))
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}
