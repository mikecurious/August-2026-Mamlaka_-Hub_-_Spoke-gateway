package main

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"log"
	"os"
)

func main5() {
	// Path to the downloaded certificate
	certPath := "ProductionCertificate.cer"
	initiatorPassword := "-Kali@linux003"

	securityCredential, err := GenerateSecurityCredential(certPath, initiatorPassword)
	if err != nil {
		log.Fatal("Error generating Security Credential:", err)
	}

	fmt.Println("SecurityCredential:", securityCredential)
}

func GenerateSecurityCredential(certPath, password string) (string, error) {
	certData, err := os.ReadFile(certPath)
	if err != nil {
		return "", fmt.Errorf("failed to read certificate file: %v", err)
	}

	block, _ := pem.Decode(certData)
	if block == nil {
		// If not PEM, try to parse as DER (most common format for .cer files)
		cert, err := x509.ParseCertificate(certData)
		if err != nil {
			return "", fmt.Errorf("failed to parse certificate as DER: %v", err)
		}
		return encryptPassword(cert, password)
	}

	// Validate PEM block type (should be "CERTIFICATE")
	if block.Type != "CERTIFICATE" {
		// If PEM block exists but isn't a certificate, try parsing raw bytes as DER
		cert, err := x509.ParseCertificate(certData)
		if err != nil {
			return "", fmt.Errorf("PEM block type is %s, expected CERTIFICATE: %v", block.Type, err)
		}
		return encryptPassword(cert, password)
	}

	// Parse PEM certificate
	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return "", fmt.Errorf("failed to parse PEM certificate: %v", err)
	}

	return encryptPassword(cert, password)
}

func encryptPassword(cert *x509.Certificate, password string) (string, error) {
	if password == "" {
		return "", fmt.Errorf("password cannot be empty")
	}

	// Verify that the public key is RSA
	pub, ok := cert.PublicKey.(*rsa.PublicKey)
	if !ok {
		return "", fmt.Errorf("certificate does not contain an RSA public key")
	}

	// Use RSA PKCS#1 v1.5 padding (not OAEP)
	encryptedBytes, err := rsa.EncryptPKCS1v15(rand.Reader, pub, []byte(password))
	if err != nil {
		return "", fmt.Errorf("error encrypting password: %v", err)
	}

	// Base64 encode the result
	encoded := base64.StdEncoding.EncodeToString(encryptedBytes)
	return encoded, nil
}