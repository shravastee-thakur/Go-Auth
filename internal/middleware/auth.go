package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/shravastee-thakur/Go-Auth/internal/utils"
)

type contextKey string

const UserContextKet contextKey = "user"

// AuthMiddleware verifies the Bearer Access Token (like authMiddleware.js in Express)
func AuthMiddleware(accessSecret string) func(http.Handler) http.Handler {
	return func (next http.Handler)http.Handler  {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// 1. Read Authorization header: "Bearer <token>"
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer "){
				utils.WriteError(w, http.StatusUnauthorized, "Missing or invalid authorization header")
				return 
			}

			tokenString := strings.TrimPrefix(authHeader, "Bearer ")

			// 2. Validate Access Token
			claims, err := utils.ValidateToken(tokenString, accessSecret)
			if err != nil {
				utils.WriteError(w, http.StatusUnauthorized, "Invalid or expired token")
				return
			}

			// 3. Attach claims to request context (like req.user = claims)
			ctx := context.WithValue(r.Context(), UserContextKet, claims)

			// 4. Call the next handler (like next())
			next.ServeHTTP(w, r.WithContext(ctx))

		})
	}
}