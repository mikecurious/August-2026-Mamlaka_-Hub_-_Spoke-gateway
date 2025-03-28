package auth

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Retrieve the Authorization header
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization header is required"})
			c.Abort()
			return
		}

		// Extract the token from the Bearer scheme
		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		if tokenString == authHeader { // Token not prefixed with "Bearer "
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid authorization token format"})
			c.Abort()
			return
		}

		// Parse and verify the token
		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			return secretKey, nil
		})

		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or expired token", "details": err.Error()})
			c.Abort()
			return
		}

		// Check if the token is valid
		if !token.Valid {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
			c.Abort()
			return
		}

		// Extract expiration from claims
		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token claims"})
			c.Abort()
			return
		}

		expirationTimestamp, ok := claims["exp"].(float64)
		merchandID, merchandOk := claims["merchantID"]
		username, usernameOk := claims["username"]

		if !ok || !merchandOk || !usernameOk {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid token claims"})
			c.Abort()
			return
		}

		// Convert expiration timestamp to time.Time
		expirationTime := time.Unix(int64(expirationTimestamp), 0)

		// Log the expiration date or include it in the response
		c.Set("token_expiration", expirationTime)
		c.Set("merchantID", merchandID)
		c.Set("username", username)

		// Proceed to the next middleware/handler
		c.Next()
	}
}
