package card

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
)

type TokenResponse struct {
	Error       bool   `json:"error"`
	Message     string `json:"message"`
	AccessToken string `json:"accessToken,omitempty"`
	Expires     int64  `json:"expires,omitempty"`
	ExpiresDate string `json:"expiresDate,omitempty"`
}

func generateToken() (string, error) {
	url := "https://sandbox.impalapay.com/api/"
	username := "MamlakaX3xeYGFXjDqq9D17MOulKesLpSpoke"
	password := "DlXLmGA57Cdzy5kFHX"
	encoded := base64.StdEncoding.EncodeToString([]byte(username + ":" + password))
	authHeader := fmt.Sprintf("Basic %s", encoded)

	// Create the HTTP request
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %v", err)
	}
	req.Header.Set("Authorization", authHeader)

	// Make the HTTP request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to send request: %v", err)
	}
	defer resp.Body.Close()

	// Read the response body
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response body: %v", err)
	}

	// Debug: Log the response body if JSON parsing fails
	fmt.Printf("Raw Response: %s\n", body)

	// Parse the JSON response
	var tokenResponse TokenResponse
	if err := json.Unmarshal(body, &tokenResponse); err != nil {
		// Handle the improperly formatted expiresDate field
		trimmedBody := strings.ReplaceAll(string(body), "'", "")
		if err := json.Unmarshal([]byte(trimmedBody), &tokenResponse); err != nil {
			return "", fmt.Errorf("failed to parse response after cleanup: %v", err)
		}
	}

	// Handle the response based on the error field
	if tokenResponse.Error {
		return "", fmt.Errorf("failed to generate token: %s", tokenResponse.Message)
	}
	encodedToken := base64.StdEncoding.EncodeToString([]byte(tokenResponse.AccessToken))

	return encodedToken, nil
}

type PaymentLinkResponse struct {
	Error      bool   `json:"error"`
	Message    string `json:"message"`
	ID         string `json:"id,omitempty"`
	SID        string `json:"sid,omitempty"`
	PaymentUrl string `json:"paymentUrl,omitempty"`
}

func GenerateCardPaymentLink(currency string, amount float64, externalId, callbackUrl, redirectUrl string) (string, error) {
	// Get the token
	token, err := generateToken()
	if err != nil {
		return "", fmt.Errorf("failed to generate token: %v", err)
	}

	// Prepare the request payload
	url := "https://sandbox.impalapay.com/api/?resource=merchant&action=generateCardPaymentLink"
	payload := map[string]interface{}{
		"impalaMerchantId": "MamlakaX3xeYGFXjDqq9D17MOulKesLpSpoke",
		"currency":         currency,
		"amount":           amount,
		"externalId":       externalId,
		"callbackUrl":      "https://payments.mam-laka.com/api/v1/card/callback",
		"redirectUrl":      redirectUrl,
	}
	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("failed to marshal payload: %v", err)
	}

	// Create the HTTP request
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonPayload))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

	// Make the HTTP request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to send request: %v", err)
	}
	defer resp.Body.Close()

	// Read the response body
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response body: %v", err)
	}

	// Parse the JSON response
	var paymentLinkResponse PaymentLinkResponse
	if err := json.Unmarshal(body, &paymentLinkResponse); err != nil {
		return "", fmt.Errorf("failed to parse response: %v", err)
	}

	// Handle the response based on the error field
	if paymentLinkResponse.Error {
		return "", fmt.Errorf("failed to generate payment link: %s", paymentLinkResponse.Message)
	}

	return paymentLinkResponse.PaymentUrl, nil
}

// func main() {
// 	paymentUrl, err := generateCardPaymentLink(
// 		"KES",
// 		5.0,
// 		"BoomW",
// 		"https://bc07-197-232-22-252.ngrok-free.app/mc/log.php",
// 		"",
// 	)
// 	if err != nil {
// 		fmt.Printf("Error: %v\n", err)
// 	} else {
// 		fmt.Printf("Payment URL: %s\n", paymentUrl)
// 	}
// }
