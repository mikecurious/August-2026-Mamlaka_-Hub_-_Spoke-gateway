package cameroon

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"
)

// Token response struct
type TokenResponse struct {
	Message     string `json:"message"`
	Status      int    `json:"status"`
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int    `json:"expires_in"`
}

// Collect request struct
type CollectRequest struct {
	SenderAccount struct {
		CountryCode  string  `json:"countryCode"`
		Number       string  `json:"number"`
		Type         string  `json:"type"`
		Name         string  `json:"name"`
		Address      string  `json:"address"`
		Amount       float64 `json:"amount"`
		CurrencyCode string  `json:"currencyCode"`
	} `json:"senderAccount"`
	BeneficiaryAccount struct {
		Service      string `json:"service"`
		CurrencyCode string `json:"currencyCode"`
	} `json:"beneficiaryAccount"`
	Reference   string `json:"reference"`
	CallbackURL string `json:"callbackURL"`
}

// Collect response struct
type CollectResponse struct {
	Message string `json:"message"`
	Status  int    `json:"status"`
}

var insecureClient = &http.Client{
	Timeout: time.Second * 10,
	Transport: &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true}, // ⚠️ Disables cert validation
	},
}

// GetAccessToken fetches the OAuth token from the sandbox server
func GetAccessToken() (string, error) {
	url := "https://api.g-payment.net/switch/api/enterprise/oauth/token"

	payload := map[string]string{
		"client_id":     "Kenya Int",
		"client_secret": "K3nya@1nt",
		"grant_type":    "partner",
		"scope":         "MH",
	}
	jsonPayload, _ := json.Marshal(payload)

	resp, err := insecureClient.Post(url, "application/json", bytes.NewBuffer(jsonPayload))
	if err != nil {
		return "", fmt.Errorf("failed to request token: %v", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("token request failed: %s", body)
	}

	var tokenResp TokenResponse
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return "", fmt.Errorf("failed to parse token response: %v", err)
	}

	return tokenResp.AccessToken, nil
}

// SendCollectRequest triggers a collect payment request
func SendCollectRequest(token, phone string, amount float64, reference, callbackURL string) error {
	url := "https://api.g-payment.net/switch/api/enterprise/collect"

	requestData := CollectRequest{}
	requestData.SenderAccount.CountryCode = "CMR"
	requestData.SenderAccount.Number = phone
	requestData.SenderAccount.Type = "M"
	requestData.SenderAccount.Name = "Test"
	requestData.SenderAccount.Address = "Test"
	requestData.SenderAccount.Amount = amount
	requestData.SenderAccount.CurrencyCode = "XAF"
	requestData.BeneficiaryAccount.Service = "WPCMHQ"
	requestData.BeneficiaryAccount.CurrencyCode = "XAF"
	requestData.Reference = reference
	requestData.CallbackURL = "https://payments.mam-laka.com/api/v1/cameroon/collect/callback"

	jsonData, _ := json.Marshal(requestData)

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to build request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := insecureClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send collect request: %v", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != 200 {
		return fmt.Errorf("collect request failed: %s", string(body))
	}

	var response CollectResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return fmt.Errorf("failed to parse collect response: %v", err)
	}

	log.Printf("✅ Collect Success: %s", response.Message)
	return nil
}

// DisburseRequest represents the payload for a disbursement
type DisburseRequest struct {
	SenderAccount struct {
		Service      string `json:"service"`
		CountryCode  string `json:"countryCode"`
		Address      string `json:"address"`
		CurrencyCode string `json:"currencyCode"`
	} `json:"senderAccount"`
	BeneficiaryAccount struct {
		CountryCode  string  `json:"countryCode"`
		Number       string  `json:"number"`
		Type         string  `json:"type"`
		Amount       float64 `json:"amount"`
		CurrencyCode string  `json:"currencyCode"`
	} `json:"beneficiaryAccount"`
	Reference   string `json:"reference"`
	CallbackURL string `json:"callbackURL"`
}

// SendDisburseRequest sends a disbursement to a mobile wallet
func SendDisburseRequest(token, phone string, amount float64, reference, callbackURL string) error {
	url := "https://api.g-payment.net/switch/api/enterprise/disburse"

	reqData := DisburseRequest{}
	reqData.SenderAccount.Service = "WPCMHQ"
	reqData.SenderAccount.CountryCode = "CMR"
	reqData.SenderAccount.Address = "Live"
	reqData.SenderAccount.CurrencyCode = "XAF"

	reqData.BeneficiaryAccount.CountryCode = "CMR"
	reqData.BeneficiaryAccount.Number = phone
	reqData.BeneficiaryAccount.Type = "M"
	reqData.BeneficiaryAccount.Amount = amount
	reqData.BeneficiaryAccount.CurrencyCode = "XAF"

	reqData.Reference = reference
	reqData.CallbackURL = "https://payments.mam-laka.com/api/v1/cameroon/disburse/callback"

	jsonData, _ := json.Marshal(reqData)

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to build disburse request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := insecureClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send disburse request: %v", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != 200 {
		return fmt.Errorf("disburse request failed: %s", string(body))
	}

	var response CollectResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return fmt.Errorf("failed to parse disburse response: %v", err)
	}

	log.Printf("✅ Disburse Success: %s", response.Message)
	return nil
}
