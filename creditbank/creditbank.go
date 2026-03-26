package creditbank

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const (
	baseURL = "https://konnectapigateway.creditbank.co.ke/b2b-tills"
	apiKey  = "cbapi_production_960ea099c6df4125b5dfcd8d9747b0a9"
	appID   = "c4921d47-1bbb-4236-a221-5d8c760576d8"
)

type TillDestinationRequest struct {
	TransactionType    string `json:"transactionType"`
	CreditAccount      string `json:"creditAccount"`
	Narration          string `json:"narration"`
	RecieverPhone      string `json:"recieverPhoneNumber"`
	Amount             string `json:"amount"`
	Timestamp          string `json:"timestamp"`
	ErrorCallBackURL   string `json:"errorCallBackUrl"`
	CallBackURL        string `json:"callBackUrl"`
	TransactionRef     string `json:"transactionReference"`
}

type TillAPIResponse struct {
	Message         string `json:"message"`
	ReferenceNumber string `json:"referenceNumber"`
	Data            struct {
		OriginatorConversationID string `json:"OriginatorConversationID"`
		ConversationID           string `json:"ConversationID"`
		ResponseCode             string `json:"ResponseCode"`
		ResponseDescription      string `json:"ResponseDescription"`
	} `json:"data"`
}

// InitiateTillPayment sends B2B till request to CreditBank gateway.
func InitiateTillPayment(creditAccount, narration, amount, callbackURL, transactionReference string) (*TillAPIResponse, error) {
	payload := TillDestinationRequest{
		TransactionType:  "MPESAB2B_TILL",
		CreditAccount:    creditAccount,
		Narration:        narration,
		RecieverPhone:    "254768899729",
		Amount:           amount,
		Timestamp:        time.Now().Format(time.RFC3339),
		ErrorCallBackURL: callbackURL,
		CallBackURL:      callbackURL,
		TransactionRef:   transactionReference,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", baseURL, bytes.NewBuffer(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("x-api-key", apiKey)
	req.Header.Set("x-app-id", appID)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		// Some environments fail TLS validation for this gateway (unknown CA, expired/not-yet-valid cert, etc).
		// Retry once with TLS verification disabled only for this call.
		errMsg := err.Error()
		if strings.Contains(errMsg, "tls: failed to verify certificate") ||
			strings.Contains(errMsg, "x509: certificate signed by unknown authority") ||
			strings.Contains(errMsg, "x509: certificate has expired or is not yet valid") {
			insecureClient := &http.Client{
				Timeout: 30 * time.Second,
				Transport: &http.Transport{
					TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
				},
			}
			resp, err = insecureClient.Do(req)
			if err != nil {
				return nil, err
			}
		} else {
			return nil, err
		}
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var out TillAPIResponse
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}
	return &out, nil
}

