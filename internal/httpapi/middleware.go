package httpapi

import (
	"errors"
	"net/http"
	"strings"
)

// corsMiddleware is intentionally permissive while the API is used for testing:
// any origin, any requested header. Restrict before exposing the service.
func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, HEAD, OPTIONS")
		allowHeaders := "Content-Type, Authorization"
		if requested := r.Header.Get("Access-Control-Request-Headers"); requested != "" {
			allowHeaders = requested
		}
		w.Header().Set("Access-Control-Allow-Headers", allowHeaders)
		w.Header().Set("Access-Control-Expose-Headers", "*")
		w.Header().Set("Access-Control-Max-Age", "86400")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}

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
