package flutterwave

import (
	"bytes"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"
)

const (
	flutterwaveAPIKey = "FLWSECK-bd2da1b48751990a85a47313a55c7b15-19a05a00a20vt-X"
	flutterwaveBaseURL = "https://api.flutterwave.com/v3/charges"
)

// GetFlutterwaveURL returns the appropriate Flutterwave URL based on country/currency
func GetFlutterwaveURL(country, currency string) string {
	// Map currency/country to Flutterwave charge type
	urlMap := map[string]string{
		"KES": "https://api.flutterwave.com/v3/charges?type=mpesa",
		"UGX": "https://api.flutterwave.com/v3/charges?type=mobile_money_uganda",
		"XAF": "https://api.flutterwave.com/v3/charges?type=mobile_money_franco",
		"ZMW": "https://api.flutterwave.com/v3/charges?type=mobile_money_zambia",
		"TZS": "https://api.flutterwave.com/v3/charges?type=mobile_money_tanzania",
		"RWF": "https://api.flutterwave.com/v3/charges?type=mobile_money_rwanda",
	}
	
	if url, ok := urlMap[currency]; ok {
		return url
	}
	
	// Default to M-Pesa if currency not found
	return "https://api.flutterwave.com/v3/charges?type=mpesa"
}

// FlutterwavePaymentRequest represents the request structure for Flutterwave M-Pesa payment
type FlutterwavePaymentRequest struct {
	TxRef       string `json:"tx_ref"`
	Amount      string `json:"amount"`
	Currency    string `json:"currency"`
	Email       string `json:"email"`
	PhoneNumber string `json:"phone_number"`
}

// FlutterwavePaymentResponse represents the response from Flutterwave API
type FlutterwavePaymentResponse struct {
	Status  string                  `json:"status"`
	Message string                  `json:"message"`
	Data    *FlutterwavePaymentData `json:"data"`
}

// FlutterwavePaymentData represents the payment data in the response
type FlutterwavePaymentData struct {
	ID                int                 `json:"id"`
	TxRef             string              `json:"tx_ref"`
	FlwRef            string              `json:"flw_ref"`
	DeviceFingerprint string              `json:"device_fingerprint"`
	Amount            int                 `json:"amount"`
	ChargedAmount     int                 `json:"charged_amount"`
	AppFee            float64             `json:"app_fee"`
	MerchantFee       int                 `json:"merchant_fee"`
	ProcessorResponse string              `json:"processor_response"`
	AuthModel         string              `json:"auth_model"`
	Currency          string              `json:"currency"`
	IP                string              `json:"ip"`
	Narration         string              `json:"narration"`
	Status            string              `json:"status"`
	AuthURL           string              `json:"auth_url"`
	PaymentType       string              `json:"payment_type"`
	FraudStatus       string              `json:"fraud_status"`
	ChargeType        string              `json:"charge_type"`
	CreatedAt         string              `json:"created_at"`
	AccountID         int                 `json:"account_id"`
	Customer          FlutterwaveCustomer `json:"customer"`
}

// FlutterwaveCustomer represents customer information
type FlutterwaveCustomer struct {
	ID          int    `json:"id"`
	PhoneNumber string `json:"phone_number"`
	Name        string `json:"name"`
	Email       string `json:"email"`
	CreatedAt   string `json:"created_at"`
}

// FlutterwaveCallbackRequest represents the callback payload from Flutterwave
type FlutterwaveCallbackRequest struct {
	Event     string                  `json:"event"`
	Data      FlutterwaveCallbackData `json:"data"`
	EventType string                  `json:"event.type"`
}

// FlutterwaveCallbackData represents the callback data
type FlutterwaveCallbackData struct {
	ID                int                 `json:"id"`
	TxRef             string              `json:"tx_ref"`
	FlwRef            string              `json:"flw_ref"`
	DeviceFingerprint string              `json:"device_fingerprint"`
	Amount            int                 `json:"amount"`
	Currency          string              `json:"currency"`
	ChargedAmount     int                 `json:"charged_amount"`
	AppFee            float64             `json:"app_fee"`
	MerchantFee       int                 `json:"merchant_fee"`
	ProcessorResponse string              `json:"processor_response"`
	AuthModel         string              `json:"auth_model"`
	IP                string              `json:"ip"`
	Narration         string              `json:"narration"`
	Status            string              `json:"status"`
	PaymentType       string              `json:"payment_type"`
	CreatedAt         string              `json:"created_at"`
	AccountID         int                 `json:"account_id"`
	Customer          FlutterwaveCustomer `json:"customer"`
}

// GenerateSecureID generates a secure random ID for transaction reference
func GenerateSecureID() string {
	b := make([]byte, 16) // Generate 16 random bytes
	_, _ = rand.Read(b)
	return base64.URLEncoding.EncodeToString(b)
}

// InitiateFlutterwavePayment initiates a mobile money payment through Flutterwave
func InitiateFlutterwavePayment(phoneNumber, email string, amount int, currency, txRef string) (*FlutterwavePaymentResponse, error) {
	// Get the appropriate URL based on currency
	url := GetFlutterwaveURL("", currency)
	
	// Prepare the request payload
	request := FlutterwavePaymentRequest{
		TxRef:       txRef,
		Amount:      fmt.Sprintf("%d", amount),
		Currency:    currency,
		Email:       email,
		PhoneNumber: phoneNumber,
	}

	// Convert to JSON
	jsonData, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create HTTP request
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+flutterwaveAPIKey)

	// Execute request
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	// Read response body
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	// Parse response
	var flutterwaveResponse FlutterwavePaymentResponse
	err = json.Unmarshal(body, &flutterwaveResponse)

	if err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	// Log response for debugging (but don't return it in API response)
	fmt.Printf("Flutterwave Payment Response: %+v\n", flutterwaveResponse)
	fmt.Printf("Flutterwave Raw Response Body: %s\n", string(body))

	return &flutterwaveResponse, nil
}

// ProcessFlutterwaveCallback processes the callback from Flutterwave
func ProcessFlutterwaveCallback(callbackData FlutterwaveCallbackRequest) error {
	fmt.Printf("Processing Flutterwave callback: %+v\n", callbackData)

	// Handle different event types
	switch callbackData.Event {
	case "charge.completed":
		if callbackData.Data.Status == "successful" {
			fmt.Printf("Payment successful for tx_ref: %s\n", callbackData.Data.TxRef)
		} else if callbackData.Data.Status == "failed" {
			fmt.Printf("Payment failed for tx_ref: %s\n", callbackData.Data.TxRef)
		}
		// Add your success/failure handling logic here
		return nil
	default:
		return fmt.Errorf("unknown event type: %s", callbackData.Event)
	}
}
