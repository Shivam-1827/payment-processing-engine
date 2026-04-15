package transport

import (
	"context"
	"net/http"

	"github.com/Shivam-1827/payment-engine/services/api-gateway/internal/auth"
)

type contextKey string

const merchantCtxKey contextKey = "merchant"


func RequireAuth(authenticator auth.Authenticator) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			apiKey := r.Header.Get("Authorization")
			if apiKey == "" {
				RespondError(w, http.StatusUnauthorized, "Missing Authorization header")
				return
			}

			merchant, err := authenticator.Authenticate(r.Context(), apiKey)
			if err != nil {
				RespondError(w, http.StatusUnauthorized, "Invalid API Key")
				return
			}

			ctx := context.WithValue(r.Context(), merchantCtxKey, merchant)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func EnforceIdempotencyKey(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			key := r.Header.Get("Idempotency-Key")
			if key == "" || len(key) > 255 {
				RespondError(w, http.StatusBadRequest, "Missing or invalid Idempotency-Key header")
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}