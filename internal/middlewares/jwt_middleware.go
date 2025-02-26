package middlewares

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// JWTMiddleware validates JWT tokens for protected routes
func JWTMiddleware(secretKey string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Get token from Authorization header
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				log.Println("Missing Authorization header")
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}

			// Check if the header has the Bearer prefix
			bearerPrefix := "Bearer "
			if !strings.HasPrefix(authHeader, bearerPrefix) {
				log.Println("Authorization header missing Bearer prefix")
				http.Error(w, "Invalid Authorization format", http.StatusUnauthorized)
				return
			}

			// Extract the token
			tokenString := strings.TrimPrefix(authHeader, bearerPrefix)
			
			// Debug logging
			log.Printf("Processing token: %s", tokenString[:10] + "...")

			// Parse and validate the token
			token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
				// Validate the signing method
				if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
					return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
				}
				return []byte(secretKey), nil
			})

			if err != nil {
				log.Printf("JWT Parse error: %v", err)
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}

			// Check if the token is valid
			if !token.Valid {
				log.Println("Invalid token")
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}

			// Extract claims
			claims, ok := token.Claims.(jwt.MapClaims)
			if !ok {
				log.Println("Failed to extract claims")
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}

			// Check if token is expired
			if exp, ok := claims["exp"].(float64); ok {
				if time.Now().Unix() > int64(exp) {
					log.Println("Token expired")
					http.Error(w, "Token expired", http.StatusUnauthorized)
					return
				}
			}

			// Extract user ID
			userID, ok := claims["sub"].(string)
			if !ok {
				log.Println("Missing user ID in token")
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}

			// Log successful authentication
			log.Printf("Authenticated user: %s", userID)

			// Add user ID to request context
			ctx := context.WithValue(r.Context(), "userID", userID)
			
			// Call the next handler with the updated context
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
