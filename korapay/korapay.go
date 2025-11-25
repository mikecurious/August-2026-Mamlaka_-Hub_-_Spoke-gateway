package korapay

import (
	"bytes"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
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

// KorapayCurrency is a custom type that can unmarshal currency as either a string or an object
type KorapayCurrency string

// UnmarshalJSON handles both string and object formats for currency
func (kc *KorapayCurrency) UnmarshalJSON(data []byte) error {
	// Try to unmarshal as string first
	var str string
	if err := json.Unmarshal(data, &str); err == nil {
		*kc = KorapayCurrency(str)
		return nil
	}

	// If not a string, try to unmarshal as an object and extract the code
	var obj map[string]interface{}
	if err := json.Unmarshal(data, &obj); err == nil {
		if code, ok := obj["code"].(string); ok {
			*kc = KorapayCurrency(code)
			return nil
		}
		if currency, ok := obj["currency"].(string); ok {
			*kc = KorapayCurrency(currency)
			return nil
		}
	}

	// If all else fails, try to extract any string value from the object
	*kc = KorapayCurrency("")
	return nil
}

// String returns the currency as a string
func (kc KorapayCurrency) String() string {
	return string(kc)
}

// KorapayPaymentData represents the payment data in the response
type KorapayPaymentData struct {
	Amount               int                    `json:"amount"`
	AmountExpected       int                    `json:"amount_expected"`
	Currency             KorapayCurrency        `json:"currency"`
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

	// Use phone number as-is (should already be in local format like 0771850050)
	formattedPhone := phoneNumber

	// Validate required fields
	if amount <= 0 {
		return nil, fmt.Errorf("amount must be greater than 0")
	}
	if currency == "" {
		return nil, fmt.Errorf("currency is required")
	}
	// Fix common currency typos (X0F -> XOF, etc.)
	if currency == "X0F" {
		currency = "XOF"
		fmt.Printf("Fixed currency typo: X0F -> XOF\n")
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

	// Log raw response for debugging
	fmt.Printf("Korapay Raw Response Body: %s\n", string(body))
	fmt.Printf("Korapay Response Status: %d\n", resp.StatusCode)

	// Parse response
	var korapayResponse KorapayPaymentResponse
	err = json.Unmarshal(body, &korapayResponse)
	if err != nil {
		return nil, fmt.Errorf("failed to parse response: %w\nRaw response: %s", err, string(body))
	}

	// Log response for debugging (but don't return it in API response)
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
