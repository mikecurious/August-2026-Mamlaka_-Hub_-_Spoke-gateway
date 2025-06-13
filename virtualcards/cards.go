package virtualcards

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Client represents the API client
type Client struct {
	BaseURL string
	APIKey  string
	Client  *http.Client
}

// NewClient creates a new API client
func NewClient(baseURL, apiKey string) *Client {
	return &Client{
		BaseURL: baseURL,
		APIKey:  apiKey,
		Client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// CreateHolderRequest represents the request for creating a card holder
type CreateHolderRequest struct {
	MerchantOrderNo string `json:"merchantOrderNo"`
	CardTypeID      int    `json:"cardTypeId"`
	AreaCode        string `json:"areaCode"`
	Mobile          string `json:"mobile"`
	Email           string `json:"email"`
	FirstName       string `json:"firstName"`
	LastName        string `json:"lastName"`
	BirthDay        string `json:"birthday"`
	Country         string `json:"country"`
	Town            string `json:"town"`
	Address         string `json:"address"`
	PostCode        string `json:"postCode"`
}

// CreateCardRequest represents the request for creating a virtual card
// type CreateCardRequest struct {
// 	CardTypeID int    `json:"cardTypeId"`
// 	HolderID   string `json:"holderId"`
// 	Amount     int    `json:"amount"`
// }

// CardInfoRequest represents the request for getting card info
type CardInfoRequest struct {
	CardNo         string `json:"cardNo"`
	OnlySimpleInfo bool   `json:"onlySimpleInfo"`
}

// DepositRequest represents the request for card top-up
type DepositRequest struct {
	CardNo string  `json:"cardNo"`
	Amount float64 `json:"amount"`
}

// Response structures
type APIResponse struct {
	Success   bool        `json:"success"`
	Data      interface{} `json:"data"`
	Code      int         `json:"code"`
	Message   string      `json:"msg"`
	Timestamp string      `json:"timestamp"`
}

type HolderData struct {
	Success         bool       `json:"success"`
	Code            int        `json:"code"`
	Message         string     `json:"msg"`
	Data            HolderInfo `json:"data"`
	MerchantOrderNo string     `json:"merchantOrderNo"`
	HolderID        string     `json:"holderId"`
	CardTypeID      string     `json:"cardTypeId"`
	Status          string     `json:"status"`
	StatusStr       string     `json:"statusStr"`
	Message2        string     `json:"message"`
}

type HolderInfo struct {
	Success   bool        `json:"success"`
	Code      int         `json:"code"`
	Message   string      `json:"msg"`
	Data      CardDetails `json:"data"`
	Timestamp string      `json:"timestamp"`
}

type CardDetails struct {
	OrderNo          string `json:"orderNo"`
	MerchantOrderNo  string `json:"merchantOrderNo"`
	Currency         string `json:"currency"`
	Amount           string `json:"amount"`
	Fee              string `json:"fee"`
	ReceivedAmount   string `json:"receivedAmount"`
	ReceivedCurrency string `json:"receivedCurrency"`
	Type             string `json:"type"`
	Status           string `json:"status"`
	TransactionTime  int64  `json:"transactionTime"`
}

type CardInfo struct {
	HolderID    string `json:"holderId"`
	CardNo      string `json:"cardNo"`
	CardNumber  string `json:"cardNumber"`
	CVV         string `json:"cvv"`
	ValidPeriod string `json:"validPeriod"`
	Status      string `json:"status"`
	StatusStr   string `json:"statusStr"`
	BindTime    int64  `json:"bindTime"`
}

// makeRequest makes an HTTP request to the API
func (c *Client) makeRequest(method, endpoint string, payload interface{}) (*APIResponse, error) {
	var body io.Reader
	if payload != nil {
		jsonData, err := json.Marshal(payload)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal payload: %w", err)
		}
		body = bytes.NewBuffer(jsonData)
	}

	req, err := http.NewRequest(method, c.BaseURL+endpoint, body)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.APIKey)

	resp, err := c.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	var apiResp APIResponse
	if err := json.Unmarshal(respBody, &apiResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return &apiResp, nil
}

// CreateHolder creates a new card holder
func (c *Client) CreateHolder(req CreateHolderRequest) (*HolderData, error) {
	resp, err := c.makeRequest("POST", "/api/card/holder/create", req)
	if err != nil {
		return nil, err
	}

	// ✅ Pretty-print the response object
	respJSON, err := json.MarshalIndent(resp, "", "  ")
	if err != nil {
		fmt.Println("⚠ Failed to marshal response for printing:", err)
	} else {
		fmt.Println("✅ Creation Response:\n", string(respJSON))
	}

	var holderData HolderData
	dataBytes, err := json.Marshal(resp.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal holder data: %w", err)
	}

	if err := json.Unmarshal(dataBytes, &holderData); err != nil {
		return nil, fmt.Errorf("failed to unmarshal holder data: %w", err)
	}

	return &holderData, nil
}

// CreateCard creates a new virtual card
func (c *Client) CreateCard(req CreateCardRequest) (*CardDetails, error) {
	resp, err := c.makeRequest("POST", "/api/card/create", req)
	if err != nil {
		return nil, err
	}

	var cardData CardDetails
	dataBytes, err := json.Marshal(resp.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal card data: %w", err)
	}

	if err := json.Unmarshal(dataBytes, &cardData); err != nil {
		return nil, fmt.Errorf("failed to unmarshal card data: %w", err)
	}

	return &cardData, nil
}

// GetCardInfo retrieves card information
func (c *Client) GetCardInfo(req CardInfoRequest) (*CardInfo, error) {
	resp, err := c.makeRequest("POST", "/api/card/info", req)
	if err != nil {
		return nil, err
	}

	var cardInfo CardInfo
	dataBytes, err := json.Marshal(resp.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal card info: %w", err)
	}

	if err := json.Unmarshal(dataBytes, &cardInfo); err != nil {
		return nil, fmt.Errorf("failed to unmarshal card info: %w", err)
	}

	return &cardInfo, nil
}

// DepositToCard tops up the card with minimum $10 USD
func (c *Client) DepositToCard(req DepositRequest) (*APIResponse, error) {
	if req.Amount < 10.0 {
		return nil, fmt.Errorf("minimum deposit amount is $10 USD")
	}

	resp, err := c.makeRequest("POST", "/api/card/deposit", req)
	if err != nil {
		return nil, err
	}

	return resp, nil
}

// VirtualCardManager handles the complete virtual card workflow
// func (c *Client) VirtualCardManager(holderReq CreateHolderRequest, cardTypeID string, depositAmount float64) error {
// 	// Step 1: Create holder
// 	fmt.Println("Step 1: Creating card holder...")
// 	holderData, err := c.CreateHolder(holderReq)
// 	if err != nil {
// 		return fmt.Errorf("failed to create holder: %w", err)
// 	}

// 	if !holderData.Success {
// 		return fmt.Errorf("holder creation failed: %s", holderData.Message)
// 	}

// 	holderID := holderData.HolderID
// 	fmt.Printf("✓ Holder created successfully. Holder ID: %s\n", holderID)

// 	// Step 2: Create virtual card
// 	fmt.Println("Step 2: Creating virtual card...")
// 	cardReq := CreateCardRequest{
// 		CardTypeID: cardTypeID,
// 		HolderID:   holderID,
// 		Amount:     10, // Initial amount for card creation
// 	}

// 	cardData, err := c.CreateCard(cardReq)
// 	if err != nil {
// 		return fmt.Errorf("failed to create card: %w", err)
// 	}

// 	fmt.Printf("✓ Virtual card created successfully. Order No: %s\n", cardData.OrderNo)

// 	// Step 3: Get card info (wait a moment for card to be processed)
// 	fmt.Println("Step 3: Retrieving card information...")
// 	time.Sleep(2 * time.Second) // Wait for card processing

// 	cardInfoReq := CardInfoRequest{
// 		CardNo:         cardData.MerchantOrderNo, // Use the merchant order number
// 		OnlySimpleInfo: true,
// 	}

// 	cardInfo, err := c.GetCardInfo(cardInfoReq)
// 	if err != nil {
// 		return fmt.Errorf("failed to get card info: %w", err)
// 	}

// 	fmt.Printf("✓ Card Info Retrieved:\n")
// 	fmt.Printf("  Card Number: %s\n", cardInfo.CardNumber)
// 	fmt.Printf("  CVV: %s\n", cardInfo.CVV)
// 	fmt.Printf("  Valid Period: %s\n", cardInfo.ValidPeriod)
// 	fmt.Printf("  Status: %s\n", cardInfo.StatusStr)

// 	// Step 4: Top up card (minimum $10 USD)
// 	if depositAmount >= 10.0 {
// 		fmt.Printf("Step 4: Topping up card with $%.2f USD...\n", depositAmount)
// 		depositReq := DepositRequest{
// 			CardNo: cardInfo.CardNo,
// 			Amount: depositAmount,
// 		}

// 		depositResp, err := c.DepositToCard(depositReq)
// 		if err != nil {
// 			return fmt.Errorf("failed to deposit to card: %w", err)
// 		}

// 		if depositResp.Success {
// 			fmt.Printf("✓ Card topped up successfully with $%.2f USD\n", depositAmount)
// 		} else {
// 			fmt.Printf("✗ Card top-up failed: %s\n", depositResp.Message)
// 		}
// 	} else {
// 		fmt.Printf("⚠ Skipping top-up: Amount $%.2f is less than minimum $10 USD\n", depositAmount)
// 	}

// 	fmt.Println("🎉 Virtual card workflow completed successfully!")
// 	return nil
// }

// Example usage
// func main() {
// 	// Initialize the client
// 	client := NewClient("https://virtualcards.impalapay.com", "your-api-key-here")

// 	// Create holder request
// 	holderReq := CreateHolderRequest{
// 		MerchantOrderNo: fmt.Sprintf("ORDER-%d", time.Now().Unix()),
// 		CardTypeID:      "111010",
// 		AreaCode:        "+254",
// 		Mobile:          "254794940168",
// 		Email:           "collinsachieng003@gmail.com",
// 		FirstName:       "Collins",
// 		LastName:        "Ochieng",
// 		BirthDay:        "1990-05-15",
// 		Country:         "US",
// 		Town:            "US_CM1",
// 		Address:         "123 Main St",
// 		PostCode:        "10001",
// 	}

// 	// Execute the complete workflow
// 	err := client.VirtualCardManager(holderReq, "111010", 25.0)
// 	if err != nil {
// 		fmt.Printf("Error: %v\n", err)
// 	}
// }
// simplified functions

// CreateHolderRequest is the input payload structure
// type CreateHolderRequest struct {
// 	MerchantOrderNo string `json:"merchantOrderNo"`
// 	CardTypeID      int    `json:"cardTypeId"`
// 	AreaCode        string `json:"areaCode"`
// 	Mobile          string `json:"mobile"`
// 	Email           string `json:"email"`
// 	FirstName       string `json:"firstName"`
// 	LastName        string `json:"lastName"`
// 	BirthDay        string `json:"birthday"`
// 	Country         string `json:"country"`
// 	Town            string `json:"town"`
// 	Address         string `json:"address"`
// 	PostCode        string `json:"postCode"`
// }

// CreateHolderResponse defines the structure of the full API response
type CreateHolderResponse struct {
	Success bool `json:"success"`
	Data    struct {
		Success bool   `json:"success"`
		Code    int    `json:"code"`
		Msg     string `json:"msg"`
		Data    struct {
			MerchantOrderNo string `json:"merchantOrderNo"`
			HolderID        int    `json:"holderId"`
			CardTypeID      int    `json:"cardTypeId"`
			Status          string `json:"status"`
			StatusStr       string `json:"statusStr"`
			Message         string `json:"message"`
		} `json:"data"`
	} `json:"data"`
	Timestamp string `json:"timestamp"`
}

// Request struct for creating a card
type CreateCardRequest struct {
	CardTypeID  int    `json:"cardTypeId"`
	HolderID    string `json:"holderId"`
	Amount      int    `json:"amount"`
	CallbackUrl string `json:"callbackUrl"`
	Status      string `json:"status"` /// SENT  or
}

// Response struct for the card creation API
type CardCreateResponse struct {
	Success bool `json:"success"`
	Data    struct {
		Success bool   `json:"success"`
		Code    int    `json:"code"`
		Msg     string `json:"msg"`
		Data    []struct {
			OrderNo          string `json:"orderNo"`
			MerchantOrderNo  string `json:"merchantOrderNo"`
			Currency         string `json:"currency"`
			Amount           string `json:"amount"`
			Fee              string `json:"fee"`
			ReceivedAmount   string `json:"receivedAmount"`
			ReceivedCurrency string `json:"receivedCurrency"`
			Type             string `json:"type"`
			Status           string `json:"status"`
			TransactionTime  int64  `json:"transactionTime"`
		} `json:"data"`
	} `json:"data"`
	Timestamp string `json:"timestamp"`
}

// SimpleCreateCardHolder sends a simplified API request
func SimpleCreateCardHolder(baseURL string, req CreateHolderRequest) (*CreateHolderResponse, error) {
	// Marshal request to JSON
	jsonBody, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	url := baseURL + "/api/card/holder/create"
	fmt.Println("➡️  Request URL:", url)
	fmt.Println("➡️  Request Body:", string(jsonBody))

	// Prepare HTTP client and request
	httpClient := &http.Client{Timeout: 20 * time.Second}
	httpReq, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	// Perform the request
	resp, err := httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	// Read raw response
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}
	fmt.Println("⬅️  Raw Response Body:", string(bodyBytes))

	// Unmarshal response
	var parsedResp CreateHolderResponse
	if err := json.Unmarshal(bodyBytes, &parsedResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	fmt.Println("✅ Parsed Response Struct:", parsedResp)
	return &parsedResp, nil
}

func CallCreateCard(baseURL string, req CreateCardRequest) (*CardCreateResponse, error) {
	jsonBody, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	client := &http.Client{Timeout: 20 * time.Second}
	httpReq, err := http.NewRequest("POST", baseURL+"/api/card/create", bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	var parsedResp CardCreateResponse
	body, _ := io.ReadAll(resp.Body)
	fmt.Println("⬅️  Raw Response Body:", string(body)) // log full response
	if err := json.Unmarshal(body, &parsedResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &parsedResp, nil
}

// card info endpoint
type CardBalanceRequest struct {
	CardNo string `json:"cardNo"`
}

type CardInfoDetails struct {
	HolderID    int64  `json:"holderId"`
	CardNo      string `json:"cardNo"`
	CardNumber  string `json:"cardNumber"`
	CVV         string `json:"cvv"`
	ValidPeriod string `json:"validPeriod"`
	Status      string `json:"status"`
	StatusStr   string `json:"statusStr"`
	BindTime    int64  `json:"bindTime"`
}

type CardInfoInnerData struct {
	Success bool             `json:"success"`
	Code    int              `json:"code"`
	Msg     string           `json:"msg"`
	Data    *CardInfoDetails `json:"data,omitempty"`
}

type CardInfoResponse struct {
	Success   bool              `json:"success"`
	Data      CardInfoInnerData `json:"data"`
	Timestamp string            `json:"timestamp"`
}

func CallGetCardInfo(baseURL string, req CardInfoRequest) (*CardInfoResponse, error) {
	jsonBody, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	client := &http.Client{Timeout: 20 * time.Second}
	httpReq, err := http.NewRequest("POST", baseURL+"/api/card/info", bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	var parsedResp CardInfoResponse
	body, _ := io.ReadAll(resp.Body)
	fmt.Println("⬅️ Raw Response Body:", string(body)) // Optional log

	if err := json.Unmarshal(body, &parsedResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &parsedResp, nil
}

// card balance
// Request body
type CardTopUpRequest struct {
	CardNo string  `json:"cardNo"`
	Amount float64 `json:"amount"` // Amount to top up
}

// API success data structure
type CardBalanceData struct {
	CardNo     string `json:"cardNo"`
	Amount     string `json:"amount"`
	UsedAmount string `json:"usedAmount"`
	Currency   string `json:"currency"`
}

// Nested response structure
type CardBalanceNestedResponse struct {
	Success bool             `json:"success"`
	Code    int              `json:"code"`
	Msg     string           `json:"msg"`
	Data    *CardBalanceData `json:"data,omitempty"`
}

// Top-level response from external API
type CardBalanceResponse struct {
	Success   bool                      `json:"success"`
	Data      CardBalanceNestedResponse `json:"data"`
	Timestamp string                    `json:"timestamp"`
}

func CallCardBalanceAPI(baseURL string, req CardBalanceRequest) (*CardBalanceResponse, error) {
	jsonBody, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	client := &http.Client{Timeout: 20 * time.Second}
	httpReq, err := http.NewRequest("POST", baseURL+"/api/card/balance", bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	var parsedResp CardBalanceResponse
	body, _ := io.ReadAll(resp.Body)
	fmt.Println("⬅️  Raw Response Body:", string(body)) // for debugging

	if err := json.Unmarshal(body, &parsedResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &parsedResp, nil
}

// Request body for recharge
type CardRechargeRequest struct {
	CardNo string  `json:"cardNo"`
	Amount float64 `json:"amount"`
}

// Response struct for successful data
type CardRechargeData struct {
	OrderNo         string `json:"orderNo"`
	MerchantOrderNo string `json:"merchantOrderNo"`
	CardNo          string `json:"cardNo"`
	Currency        string `json:"currency"`
	Amount          string `json:"amount"`
	Fee             string `json:"fee"`
	Type            string `json:"type"`
	Status          string `json:"status"`
	Remark          string `json:"remark"`
	TransactionTime int64  `json:"transactionTime"`
}

// Nested structure from API
type CardRechargeNestedResponse struct {
	Success bool              `json:"success"`
	Code    int               `json:"code"`
	Msg     string            `json:"msg"`
	Data    *CardRechargeData `json:"data,omitempty"`
}

// Top-level response from external API
type CardRechargeResponse struct {
	Success   bool                       `json:"success"`
	Data      CardRechargeNestedResponse `json:"data"`
	Timestamp string                     `json:"timestamp"`
}

func CallCardRechargeAPI(baseURL string, req CardRechargeRequest) (*CardRechargeResponse, error) {
	jsonBody, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	client := &http.Client{Timeout: 20 * time.Second}
	httpReq, err := http.NewRequest("POST", baseURL+"/api/card/deposit", bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	var parsedResp CardRechargeResponse
	body, _ := io.ReadAll(resp.Body)
	fmt.Println("⬅️  Raw Response Body:", string(body)) // log full response

	if err := json.Unmarshal(body, &parsedResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &parsedResp, nil
}
