package airtel

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"errors"
	"fmt"
	"os"
	"strings"
)

// EncryptPIN returns the disbursement PIN in the form Airtel expects. It prefers
// the pre-encrypted AIRTEL_ENCRYPTED_PIN (what Airtel support issues); if that is
// absent it encrypts AIRTEL_B2C_PIN locally with AIRTEL_B2C_PUBLIC_KEY
// (RSA PKCS1v15 + standard base64). A token is required (Airtel scopes the
// encryption to the authenticated session in the pre-encrypted flow).
func EncryptPIN(token string) (string, error) {
	if strings.TrimSpace(token) == "" {
		return "", errors.New("airtel: missing access token for PIN encryption")
	}

	if encrypted := strings.TrimSpace(os.Getenv("AIRTEL_ENCRYPTED_PIN")); encrypted != "" {
		return encrypted, nil
	}

	cleanPin := strings.TrimSpace(os.Getenv("AIRTEL_B2C_PIN"))
	if cleanPin == "" {
		return "", errors.New("airtel: neither AIRTEL_ENCRYPTED_PIN nor AIRTEL_B2C_PIN is configured")
	}

	publicKeyPEM := strings.TrimSpace(strings.Trim(os.Getenv("AIRTEL_B2C_PUBLIC_KEY"), `"`))
	if publicKeyPEM == "" {
		return "", errors.New("airtel: AIRTEL_B2C_PUBLIC_KEY is not configured")
	}

	block, _ := pem.Decode([]byte(publicKeyPEM))
	if block == nil {
		return "", errors.New("airtel: invalid AIRTEL_B2C_PUBLIC_KEY PEM (use real newlines, not literal \\n)")
	}
	if block.Type != "PUBLIC KEY" {
		return "", errors.New("airtel: AIRTEL_B2C_PUBLIC_KEY is not a PUBLIC KEY PEM")
	}

	parsedKey, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return "", fmt.Errorf("airtel: failed to parse public key: %w", err)
	}
	rsaPublicKey, ok := parsedKey.(*rsa.PublicKey)
	if !ok {
		return "", errors.New("airtel: AIRTEL_B2C_PUBLIC_KEY is not an RSA key")
	}

	encryptedBytes, err := rsa.EncryptPKCS1v15(rand.Reader, rsaPublicKey, []byte(cleanPin))
	if err != nil {
		return "", fmt.Errorf("airtel: RSA encryption failed: %w", err)
	}
	return base64.StdEncoding.EncodeToString(encryptedBytes), nil
}
