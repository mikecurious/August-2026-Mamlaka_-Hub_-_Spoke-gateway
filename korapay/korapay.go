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
	// korapayAPIKey = "sk_live_xiax7gn8X3iQQHtHp1orupbZqdberoNnrVvJs6Mf"

	korapayURL            = "https://api.korapay.com/merchant/api/v1/charges/mobile-money"
	korapayBankTransferURL = "https://api.korapay.com/merchant/api/v1/charges/bank-transfer"
	korapayDisburseURL     = "https://api.korapay.com/merchant/api/v1/transactions/disburse"
	korapayBanksURL        = "https://api.korapay.com/merchant/api/v1/misc/banks"
	korapayBanksKey        = "pk_test_Zps1MZnosrAUBY4oCMM7ZExLfyMeUEMuSEpNjzy8"
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

// ListBanks fetches banks for a country from Korapay misc/banks API.
// countryCode is the 2-letter country code (e.g. "NG").
// Returns the raw API response: { "status": true, "message": "Successful", "data": [...] }.
func ListBanks(countryCode string) (map[string]interface{}, error) {
	if countryCode == "" {
		return nil, fmt.Errorf("countryCode is required")
	}
	url := korapayBanksURL + "?countryCode=" + countryCode
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+korapayBanksKey)
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}
	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}
	return result, nil
}

// BankTransferChargeRequest is the payload for Korapay bank-transfer charge.
type BankTransferChargeRequest struct {
	AccountName string         `json:"account_name"`
	Amount      int            `json:"amount"`
	Currency    string         `json:"currency"`
	Reference   string         `json:"reference"`
	Customer    KorapayCustomer `json:"customer"`
}

// BankTransferChargeResponse is the response from Korapay bank-transfer charge.
type BankTransferChargeResponse struct {
	Status  bool                        `json:"status"`
	Message string                      `json:"message"`
	Data    *BankTransferChargeData     `json:"data"`
}

// BankTransferChargeData holds bank transfer charge response data.
type BankTransferChargeData struct {
	Currency          string                 `json:"currency"`
	Amount            int                    `json:"amount"`
	AmountExpected    int                    `json:"amount_expected"`
	Fee               float64                `json:"fee"`
	Vat               float64                `json:"vat"`
	Reference         string                 `json:"reference"`
	PaymentReference  string                 `json:"payment_reference"`
	Status            string                 `json:"status"`
	Narration         string                 `json:"narration"`
	MerchantBearsCost bool                   `json:"merchant_bears_cost"`
	BankAccount       map[string]interface{} `json:"bank_account"`
	Customer          KorapayCustomer        `json:"customer"`
}

// ChargeBankTransfer initiates a Korapay bank-transfer (payin) charge.
// reference is the unique reference (e.g. external_id) sent to Korapay; callback will use it.
// Also returns the raw response map so callers can read bank_account / bankAccount regardless of key format.
func ChargeBankTransfer(accountName string, amount int, currency, reference, customerName, customerEmail string) (*BankTransferChargeResponse, map[string]interface{}, error) {
	if reference == "" {
		return nil, nil, fmt.Errorf("reference is required")
	}
	if amount <= 0 {
		return nil, nil, fmt.Errorf("amount must be greater than 0")
	}
	if currency == "" {
		return nil, nil, fmt.Errorf("currency is required")
	}
	if customerName == "" || customerEmail == "" {
		return nil, nil, fmt.Errorf("customer name and email are required")
	}
	if accountName == "" {
		accountName = "Payment"
	}
	payload := BankTransferChargeRequest{
		AccountName: accountName,
		Amount:      amount,
		Currency:    currency,
		Reference:   reference,
		Customer: KorapayCustomer{
			Name:  customerName,
			Email: customerEmail,
		},
	}
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to marshal request: %w", err)
	}
	req, err := http.NewRequest("POST", korapayBankTransferURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+korapayAPIKey)
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, nil, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read response body: %w", err)
	}
	var rawMap map[string]interface{}
	_ = json.Unmarshal(body, &rawMap)
	var result BankTransferChargeResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, rawMap, fmt.Errorf("failed to parse response: %w", err)
	}
	// Korapay may return bank_account in snake_case or bankAccount in camelCase; ensure we have it
	if result.Data != nil && result.Data.BankAccount == nil && rawMap != nil {
		if data, _ := rawMap["data"].(map[string]interface{}); data != nil {
			if ba, ok := data["bank_account"].(map[string]interface{}); ok {
				result.Data.BankAccount = ba
			}
			if result.Data.BankAccount == nil {
				if ba, ok := data["bankAccount"].(map[string]interface{}); ok {
					result.Data.BankAccount = ba
				}
			}
		}
	}
	return &result, rawMap, nil
}

// DisburseRequest is the payload for Korapay disburse (payout).
type DisburseRequest struct {
	Reference    string              `json:"reference"`
	Destination  DisburseDestination  `json:"destination"`
}

// DisburseDestination holds destination type, amount, currency, bank_account, customer.
type DisburseDestination struct {
	Type        string               `json:"type"` // "bank_account"
	Amount      string               `json:"amount"`
	Currency    string               `json:"currency"`
	Narration   string               `json:"narration"`
	BankAccount DisburseBankAccount  `json:"bank_account"`
	Customer    KorapayCustomer      `json:"customer"`
}

// DisburseBankAccount holds bank code and account number.
type DisburseBankAccount struct {
	Bank    string `json:"bank"`
	Account string `json:"account"`
}

// DisburseResponse is the response from Korapay disburse.
type DisburseResponse struct {
	Status  bool                `json:"status"`
	Message string              `json:"message"`
	Data    *DisburseResponseData `json:"data"`
}

// DisburseResponseData holds disburse result.
type DisburseResponseData struct {
	Amount    string          `json:"amount"`
	Fee       string          `json:"fee"`
	Currency  string          `json:"currency"`
	Status    string          `json:"status"`
	Reference string          `json:"reference"`
	Narration string          `json:"narration"`
	Message   string          `json:"message"`
	Customer  KorapayCustomer `json:"customer"`
	Metadata  map[string]interface{} `json:"metadata"`
}

// Disburse initiates a payout to a bank account.
func Disburse(reference, amount, currency, narration, bankCode, accountNumber, customerName, customerEmail string) (*DisburseResponse, error) {
	if reference == "" {
		return nil, fmt.Errorf("reference is required")
	}
	if amount == "" || currency == "" || bankCode == "" || accountNumber == "" {
		return nil, fmt.Errorf("amount, currency, bank code and account number are required")
	}
	if customerName == "" || customerEmail == "" {
		return nil, fmt.Errorf("customer name and email are required")
	}
	if narration == "" {
		narration = "Payout"
	}
	payload := DisburseRequest{
		Reference: reference,
		Destination: DisburseDestination{
			Type:      "bank_account",
			Amount:    amount,
			Currency:  currency,
			Narration: narration,
			BankAccount: DisburseBankAccount{
				Bank:    bankCode,
				Account: accountNumber,
			},
			Customer: KorapayCustomer{
				Name:  customerName,
				Email: customerEmail,
			},
		},
	}
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}
	req, err := http.NewRequest("POST", korapayDisburseURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+korapayAPIKey)
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}
	var result DisburseResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}
	return &result, nil
}
