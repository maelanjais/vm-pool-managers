package middlewares

import (
	"PoolManagerVM/backend/config"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

// AuthMiddleware is a Gin middleware that validates JWT access tokens in incoming requests.
//
// Workflow:
//  1. Reads the "Authorization" header from the request.
//     - Expects the format: "Bearer <token>".
//  2. Parses and validates the JWT using HMAC and the configured secret.
//  3. If the token is missing, invalid, or has a wrong signature, it returns 401 Unauthorized and aborts the request.
//  4. If valid, extracts the "user_id" claim from the token and stores it in the Gin context for use by downstream handlers.
//  5. Calls c.Next() to continue processing the request.
//
// Usage:
//
//	router.Use(AuthMiddleware())
func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			// Fallback pour WebSocket (si token dans l’URL)
			tokenQuery := c.Query("token")
			if tokenQuery != "" {
				authHeader = "Bearer " + tokenQuery
			} else {
				c.JSON(http.StatusUnauthorized, gin.H{"error": "authorization header missing"})
				c.Abort()
				return
			}
		}
		tokenString := authHeader[len("Bearer "):]

		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (any, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, jwt.ErrSignatureInvalid
			}
			return config.JWTSecret, nil
		})

		if err != nil || !token.Valid {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
			c.Abort()
			return
		}

		if claims, ok := token.Claims.(jwt.MapClaims); ok {
			userID := fmt.Sprintf("%v", claims["user_id"])
			email := fmt.Sprintf("%v", claims["email"])

			c.Set("user_id", userID)
			c.Set("email", email)
		} else {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
			c.Abort()
			return
		}

		c.Next()
	}
}
