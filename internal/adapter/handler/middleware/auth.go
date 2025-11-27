package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/go-chi/render"
	"github.com/google/uuid"
	"github.com/vantutran2k1/orbit/internal/core/service"
)

type ctxKey struct{}

var TenantIDKey = ctxKey{}

func AuthMiddleware(authSvc *service.AuthService) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				authHeader = r.URL.Query().Get("api_key")
			}

			token := strings.TrimPrefix(authHeader, "Bearer ")
			token = strings.TrimSpace(token)
			if token == "" {
				render.Status(r, http.StatusUnauthorized)
				render.JSON(w, r, map[string]string{"error": "missing api key"})
				return
			}

			tenantID, err := authSvc.ValidateKey(r.Context(), token)
			if err != nil {
				render.Status(r, http.StatusUnauthorized)
				render.JSON(w, r, map[string]string{"error": "invalid api key"})
				return
			}

			ctx := context.WithValue(r.Context(), TenantIDKey, tenantID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func GetTenantID(ctx context.Context) uuid.UUID {
	id, ok := ctx.Value(TenantIDKey).(uuid.UUID)
	if !ok {
		return uuid.Nil
	}
	return id
}
