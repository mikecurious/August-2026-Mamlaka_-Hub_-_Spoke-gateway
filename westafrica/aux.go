package westafrica

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// AirtimeRequest represents the request payload for airtime transaction
type AirtimeRequest struct {
	Amount      int    `json:"amount"`
	Destination string `json:"destination"`
	APIKey      string `json:"api_key"`
	IPNUrl      string `json:"ipn_url"`
	ServiceID   int    `json:"service_id"`
	OMOTP       string `json:"om_otp"`
	CustomData  string `json:"custom_data"`
}

// AirtimeResponse represents the response from the airtime API
type AirtimeResponse struct {
	Data struct {
		TransactionID      string  `json:"transaction_id"`
		Amount             int     `json:"amount"`
		Benefice           int     `json:"benefice"`
		Commission         string  `json:"comission"`
		Destination        string  `json:"destination"`
		Fee                string  `json:"fee"`
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
			Timeout: 10 * time.Second,
		},
	}
}

// SendAirtimeTransaction sends an airtime transaction request
func (c *AirtimeClient) SendAirtimeTransaction(ctx context.Context, req *AirtimeRequest) (*AirtimeResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("request cannot be nil")
	}

	// Validate required fields
	if err := c.validateRequest(req); err != nil {
		return nil, fmt.Errorf("validation error: %w", err)
	}

	// Marshal request to JSON
	jsonData, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

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
