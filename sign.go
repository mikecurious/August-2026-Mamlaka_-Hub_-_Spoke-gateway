package main

// import (
// 	"crypto/rand"
// 	"crypto/rsa"
// 	"crypto/x509"
// 	"encoding/base64"
// 	"encoding/pem"
// 	"fmt"
// 	"log"
// 	"os"
// )

// func main() {
// 	certPath := "ProductionCertificate.cer"
// 	initiatorPassword := "Sad@magara12345&"

// 	securityCredential, err := GenerateSecurityCredential(certPath, initiatorPassword)
// 	if err != nil {
// 		log.Fatal("Error generating Security Credential:", err)
// 	}

// 	fmt.Println("SecurityCredential:", securityCredential)
// }

// func GenerateSecurityCredential(certPath, password string) (string, error) {
// 	certData, err := os.ReadFile(certPath)
// 	if err != nil {
// 		return "", fmt.Errorf("failed to read certificate file: %v", err)
// 	}

// 	block, _ := pem.Decode(certData)
// 	if block == nil {
// 		cert, err := x509.ParseCertificate(certData)
// 		if err != nil {
// 			return "", fmt.Errorf("failed to parse certificate as DER: %v", err)
// 		}
// 		return encryptPassword(cert, password)
// 	}

// 	if block.Type != "CERTIFICATE" {
// 		cert, err := x509.ParseCertificate(certData)
// 		if err != nil {
// 			return "", fmt.Errorf("PEM block type is %s, expected CERTIFICATE: %v", block.Type, err)
// 		}
// 		return encryptPassword(cert, password)
// 	}

// 	cert, err := x509.ParseCertificate(block.Bytes)
// 	if err != nil {
// 		return "", fmt.Errorf("failed to parse PEM certificate: %v", err)
// 	}

// 	return encryptPassword(cert, password)
// }

// func encryptPassword(cert *x509.Certificate, password string) (string, error) {
// 	if password == "" {
// 		return "", fmt.Errorf("password cannot be empty")
// 	}

// 	pub, ok := cert.PublicKey.(*rsa.PublicKey)
// 	if !ok {
// 		return "", fmt.Errorf("certificate does not contain an RSA public key")
// 	}

// 	encryptedBytes, err := rsa.EncryptPKCS1v15(rand.Reader, pub, []byte(password))
// 	if err != nil {
// 		return "", fmt.Errorf("error encrypting password: %v", err)
// 	}

// 	encoded := base64.StdEncoding.EncodeToString(encryptedBytes)
// 	return encoded, nil
// }
