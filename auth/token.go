package auth

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var secretKey = []byte("secret-key")

// CreateToken generates the JWT token and the expiration date
func CreateToken(username, merchantID string) (string, string, error) {
	// Define expiration time
	expirationTime := time.Now().Add(time.Hour * 24) // 24 hours from now
	token := jwt.NewWithClaims(jwt.SigningMethodHS256,
		jwt.MapClaims{
			"username":   username,
			"merchantID": merchantID,
			"exp":        expirationTime.Unix(),
		})

	tokenString, err := token.SignedString(secretKey)
	if err != nil {
		return "", "", err
	}

	// Return token and expiration date as a formatted string
	return tokenString, expirationTime.Format(time.RFC3339), nil
}

func VerifyToken(tokenString string) error {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		return secretKey, nil
	})

	if err != nil {
		return err
	}

	if !token.Valid {
		return fmt.Errorf("invalid token")
	}

	return nil
}
