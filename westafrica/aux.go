package westafrica

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"
)

// West Africa service IDs for Pixel core API (airtime endpoint).
const (
	ServiceIDBeninMTNCollection = 305
	ServiceIDBeninMTNPayout     = 304
	ServiceIDSenegalOrangePayin = 153
	ServiceIDSenegalWavePayin   = 151
	ServiceIDSenegalPayout      = 150
)

var pixelOutboundAPIKeyPattern = regexp.MustCompile(`"api_key"\s*:\s*"[^"]*"`)

func redactAPIKeyInJSON(b []byte) string {
	return pixelOutboundAPIKeyPattern.ReplaceAllString(string(b), `"api_key":"***"`)
}

// HardcodedIPNURL is the ipn_url sent to Pixel for collections/payouts (must be publicly reachable HTTPS).
const HardcodedIPNURL = "https://payments.mam-laka.com/api/v1/west-africa/callback"

// DefaultPublicIPNURL kept as alias for HardcodedIPNURL.
const DefaultPublicIPNURL = HardcodedIPNURL

// ResolveIPNURL returns the hardcoded callback URL (env override disabled for now).
func ResolveIPNURL() string {
	return HardcodedIPNURL
}

// ResolvePixelAPIKey returns PIXEL_CORE_API_KEY trimmed; empty if unset.
func ResolvePixelAPIKey() string {
	return strings.TrimSpace(os.Getenv("PIXEL_CORE_API_KEY"))
}

// NormalizeBeninMSISDN converts +229 / 229-prefixed numbers to local format with a leading 0 (e.g. 0190760023).
func NormalizeBeninMSISDN(phone string) string {
	s := strings.TrimSpace(phone)
	s = strings.TrimPrefix(s, "+")
	s = strings.ReplaceAll(s, " ", "")
	if strings.HasPrefix(s, "229") {
		s = s[3:]
	}
	if s == "" {
		return s
	}
	if !strings.HasPrefix(s, "0") {
		s = "0" + s
	}
	return s
}

// NormalizeSenegalMSISDN accepts +221/221/local numbers and returns local Senegal format.
func NormalizeSenegalMSISDN(phone string) string {
	s := strings.TrimSpace(phone)
	s = strings.TrimPrefix(s, "+")
	s = strings.ReplaceAll(s, " ", "")
	if strings.HasPrefix(s, "221") {
		s = s[3:]
	}
	return s
}

// AirtimeRequest represents the request payload for airtime transaction
type AirtimeRequest struct {
	Amount      int    `json:"amount"`
	Destination string `json:"destination"`
	APIKey      string `json:"api_key"`
	IPNUrl      string `json:"ipn_url"`
	ServiceID   int    `json:"service_id"`
	OMOTP       string `json:"om_otp,omitempty"`
	CustomData  string `json:"custom_data"`
}

// AirtimeResponse represents the response from the airtime API
type AirtimeResponse struct {
	Data struct {
		TransactionID      string  `json:"transaction_id"`
		Amount             int     `json:"amount"`
		Benefice           int     `json:"benefice"`
		Commission         float32 `json:"comission"`
		Destination        string  `json:"destination"`
		Fee                float32 `json:"fee"`
		Response           string  `json:"response"`
		Error              *string `json:"error"`
		ServiceID          int     `json:"service_id"`
		CustomerName       string  `json:"customer_name"`
		State              string  `json:"state"`
		CustomData         string  `json:"custom_data"`
		IPNUrl             string  `json:"ipn_url"`
		TransactionChannel *string `json:"transaction_channel"`
		ProviderID         string  `json:"provider_id"`
		SMSLink            string  `json:"sms_link"`
		PID                int     `json:"p_id"`
		PLastWalletAmount  int     `json:"p_last_wallet_amount"`
		PNewWalletAmount   *int    `json:"p_new_wallet_amount"`
		CreatedAt          string  `json:"created_at"`
		UpdatedAt          string  `json:"updated_at"`
	} `json:"data"`
	Message    string `json:"message"`
	StatusCode int    `json:"statut_code"`
}

// AirtimeClient handles airtime transaction requests
type AirtimeClient struct {
	BaseURL    string
	HTTPClient *http.Client
}

// NewAirtimeClient creates a new airtime client
func NewAirtimeClient() *AirtimeClient {
	return &AirtimeClient{
		BaseURL: "https://proxy-coreapi.pixelinnov.net",
		HTTPClient: &http.Client{
			Timeout: 60 * time.Second,
		},
	}
}

// SendAirtimeTransaction sends an airtime transaction request
func (c *AirtimeClient) SendAirtimeTransaction(ctx context.Context, req *AirtimeRequest) (*AirtimeResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("request cannot be nil")
	}

	// Sanitize fields (env / copy-paste often introduces leading/trailing spaces).
	clean := *req
	clean.IPNUrl = strings.TrimSpace(clean.IPNUrl)
	clean.APIKey = strings.TrimSpace(clean.APIKey)
	clean.Destination = strings.TrimSpace(clean.Destination)
	clean.CustomData = strings.TrimSpace(clean.CustomData)
	clean.OMOTP = strings.TrimSpace(clean.OMOTP)
	req = &clean

	// Validate required fields
	if err := c.validateRequest(req); err != nil {
		return nil, fmt.Errorf("validation error: %w", err)
	}

	log.Printf("westafrica Pixel request: service_id=%d ipn_url=%q destination=%q custom_data=%q", req.ServiceID, req.IPNUrl, req.Destination, req.CustomData)

	// Marshal request to JSON
	jsonData, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	log.Printf("westafrica Pixel outbound JSON: %s", redactAPIKeyInJSON(jsonData))

	// Create HTTP request
	url := fmt.Sprintf("%s/api_v1/transaction/airtime", c.BaseURL)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request: %w", err)
	}

	// Set headers
	httpReq.Header.Set("Content-Type", "application/json")

	// Send request
	resp, err := c.HTTPClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	// Check HTTP status code
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP error: %d - %s", resp.StatusCode, string(body))
	}

	// Parse response
	var airtimeResp AirtimeResponse
	if err := json.Unmarshal(body, &airtimeResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	if airtimeResp.StatusCode != 0 && airtimeResp.StatusCode != http.StatusOK {
		return &airtimeResp, fmt.Errorf("API statut_code %d: %s", airtimeResp.StatusCode, airtimeResp.Message)
	}

	// Check if API returned an error
	if airtimeResp.Data.Error != nil && *airtimeResp.Data.Error != "" {
		return &airtimeResp, fmt.Errorf("API error: %s", *airtimeResp.Data.Error)
	}

	return &airtimeResp, nil
}

// validateRequest validates the airtime request
func (c *AirtimeClient) validateRequest(req *AirtimeRequest) error {
	if req.Amount <= 0 {
		return fmt.Errorf("amount must be greater than 0")
	}
	if req.Destination == "" {
		return fmt.Errorf("destination is required")
	}
	if req.APIKey == "" {
		return fmt.Errorf("api_key is required")
	}
	if req.ServiceID <= 0 {
		return fmt.Errorf("service_id must be greater than 0")
	}
	if req.IPNUrl == "" {
		return fmt.Errorf("ipn_url is required (set WEST_AFRICA_IPN_URL or pass a public https URL)")
	}
	return nil
}

// Example usage function
// func ExampleUsage() {
// 	// Create client
// 	client := NewAirtimeClient()

// 	// Create request
// 	req := &AirtimeRequest{
// 		Amount:      200,
// 		Destination: "0709110204",
// 		APIKey:      "PIX_737219e4-4980-4000-b0a9-a0393bbcaf28",
// 		IPNUrl:      "https://webhook.site/f0f0657a-4fef-4c19-a6e3-f521cf2947b7",
// 		ServiceID:   7,
// 		OMOTP:       "1080",
// 		CustomData:  "your_custom_data",
// 	}

// 	// Send request with context and timeout
// 	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
// 	defer cancel()

// 	response, err := client.SendAirtimeTransaction(ctx, req)
// 	if err != nil {
// 		fmt.Printf("Error: %v\n", err)
// 		return
// 	}

// 	// Handle successful response
// 	fmt.Printf("Transaction ID: %s\n", response.Data.TransactionID)
// 	fmt.Printf("Amount: %d\n", response.Data.Amount)
// 	fmt.Printf("State: %s\n", response.Data.State)
// 	fmt.Printf("SMS Link: %s\n", response.Data.SMSLink)
// 	fmt.Printf("Message: %s\n", response.Message)
// }

// func main() {
// 	ExampleUsage()
// }
