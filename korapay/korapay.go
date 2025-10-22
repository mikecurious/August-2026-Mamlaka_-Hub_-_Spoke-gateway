package korapay

import (
	"bytes"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"time"
)

const (
	korapayAPIKey = "sk_live_xiax7gn8X3iQQHtHp1orupbZqdberoNnrVvJs6Mf"
	korapayURL    = "https://api.korapay.com/merchant/api/v1/charges/mobile-money"
)

// KorapayPaymentRequest represents the request structure for Korapay mobile money payment
type KorapayPaymentRequest struct {
	Amount            int                `json:"amount"`
	Currency          string             `json:"currency"`
	Reference         string             `json:"reference"`
	Description       string             `json:"description"`
	NotificationURL   string             `json:"notification_url"`
	RedirectURL       string             `json:"redirect_url"`
	Customer          KorapayCustomer    `json:"customer"`
	MerchantBearsCost bool               `json:"merchant_bears_cost"`
	MobileMoney       KorapayMobileMoney `json:"mobile_money"`
}

// KorapayCustomer represents customer information
type KorapayCustomer struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}

// KorapayMobileMoney represents mobile money details
type KorapayMobileMoney struct {
	Number string `json:"number"`
}

// KorapayPaymentResponse represents the response from Korapay API
type KorapayPaymentResponse struct {
	Status  bool                `json:"status"`
	Message string              `json:"message"`
	Data    *KorapayPaymentData `json:"data"`
}

// KorapayPaymentData represents the payment data in the response
type KorapayPaymentData struct {
	Amount               int                    `json:"amount"`
	AmountExpected       int                    `json:"amount_expected"`
	Currency             string                 `json:"currency"`
	Fee                  float64                `json:"fee"`
	AuthModel            string                 `json:"auth_model"`
	TransactionReference string                 `json:"transaction_reference"`
	PaymentReference     string                 `json:"payment_reference"`
	Status               string                 `json:"status"`
	Narration            string                 `json:"narration"`
	Message              string                 `json:"message"`
	Authorization        map[string]interface{} `json:"authorization"`
	Metadata             KorapayMetadata        `json:"metadata"`
	MobileMoney          KorapayMobileMoney     `json:"mobile_money"`
	Customer             KorapayCustomer        `json:"customer"`
}

// KorapayMetadata represents metadata in the response
type KorapayMetadata struct {
	CanResendStk bool `json:"can_resend_stk"`
}

// KorapayCallbackRequest represents the callback payload from Korapay
type KorapayCallbackRequest struct {
	Event string              `json:"event"`
	Data  KorapayCallbackData `json:"data"`
}

// KorapayCallbackData represents the callback data
type KorapayCallbackData struct {
	Reference        string  `json:"reference"`
	PaymentReference string  `json:"payment_reference"`
	Currency         string  `json:"currency"`
	Amount           int     `json:"amount"`
	Fee              float64 `json:"fee"`
	PaymentMethod    string  `json:"payment_method"`
	Status           string  `json:"status"`
}

// GenerateSecureID generates a secure random ID for transaction reference
func GenerateSecureID() string {
	const charset = "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, 12) // Generate 12 characters
	
	// Use crypto/rand for better randomness
	randomBytes := make([]byte, 12)
	_, _ = rand.Read(randomBytes)
	
	for i := range b {
		b[i] = charset[randomBytes[i]%byte(len(charset))]
	}
	return string(b)
}

// InitiateKorapayPayment initiates a mobile money payment through Korapay
func InitiateKorapayPayment(phoneNumber, customerName, customerEmail string, amount int, currency, description, callbackURL, redirectURL string) (*KorapayPaymentResponse, error) {
	// Generate secure reference
	reference := GenerateSecureID()
	fmt.Printf("Generated Korapay reference: %s\n", reference)

	// Format phone number with + if not already present
	formattedPhone := phoneNumber
	if !strings.HasPrefix(phoneNumber, "+") {
		formattedPhone = "+" + phoneNumber
	}

	// Validate required fields
	if amount <= 0 {
		return nil, fmt.Errorf("amount must be greater than 0")
	}
	if currency == "" {
		return nil, fmt.Errorf("currency is required")
	}
	if formattedPhone == "" {
		return nil, fmt.Errorf("phone number is required")
	}
	if customerEmail == "" {
		return nil, fmt.Errorf("customer email is required")
	}

	// Prepare the request payload
	request := KorapayPaymentRequest{
		Amount:          amount,
		Currency:        currency,
		Reference:       reference,
		Description:     description,
		NotificationURL: callbackURL,
		RedirectURL:     redirectURL,
		Customer: KorapayCustomer{
			Name:  customerName,
			Email: customerEmail,
		},
		MerchantBearsCost: true,
		MobileMoney: KorapayMobileMoney{
			Number: formattedPhone,
		},
	}

	// Convert to JSON
	jsonData, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Log the request for debugging
	fmt.Printf("Korapay Request Payload: %s\n", string(jsonData))

	// Create HTTP request
	req, err := http.NewRequest("POST", korapayURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+korapayAPIKey)

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

	// Log the response for debugging
	fmt.Printf("Korapay Response Status: %d\n", resp.StatusCode)
	fmt.Printf("Korapay Response Body: %s\n", string(body))

	// Parse response
	var korapayResponse KorapayPaymentResponse
	err = json.Unmarshal(body, &korapayResponse)
	if err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	// Log response for debugging
	fmt.Printf("Korapay Payment Response: %+v\n", korapayResponse)

	return &korapayResponse, nil
}

// ProcessKorapayCallback processes the callback from Korapay
func ProcessKorapayCallback(callbackData KorapayCallbackRequest) error {
	fmt.Printf("Processing Korapay callback: %+v\n", callbackData)

	// Handle different event types
	switch callbackData.Event {
	case "charge.success":
		fmt.Printf("Payment successful for reference: %s\n", callbackData.Data.Reference)
		// Add your success handling logic here
		return nil
	case "charge.failed":
		fmt.Printf("Payment failed for reference: %s\n", callbackData.Data.Reference)
		// Add your failure handling logic here
		return nil
	default:
		return fmt.Errorf("unknown event type: %s", callbackData.Event)
	}
}
