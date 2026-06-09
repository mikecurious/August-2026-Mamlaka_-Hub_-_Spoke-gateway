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
	baseURL              = "https://konnectapigateway.creditbank.co.ke/b2b-tills"
	pesalinkToAccountURL = "https://konnectapigateway.creditbank.co.ke/pesalink-to-account"
	pesalinkStatusURL    = "https://konnectapigateway.creditbank.co.ke/pesalink-payment-status-check"
	apiKey               = "cbapi_production_498d8e02839f4584a70e186865af3cb1"
	appID                = "c40393bd-be12-49cc-9c60-d04a0ac58fef"
	paybillURL           = "https://konnectapigateway.creditbank.co.ke/b2b"
)

type TillDestinationRequest struct {
	TransactionType  string `json:"transactionType"`
	CreditAccount    string `json:"creditAccount"`
	Narration        string `json:"narration"`
	RecieverPhone    string `json:"recieverPhoneNumber"`
	Amount           string `json:"amount"`
	Timestamp        string `json:"timestamp"`
	ErrorCallBackURL string `json:"errorCallBackUrl"`
	CallBackURL      string `json:"callBackUrl"`
	TransactionRef   string `json:"transactionReference"`
}

/*.
"transactionType": "MPESAB2B_PAYBILL",
   "creditAccount": "4136433",

   "narration": "92997",
   "recieverPhoneNumber": "254711354342",
   "amount": "1",
   "timestamp": "2021-01-29T00:54:08+03:00",
   "errorCallBackUrl": "https://whbc4735654c6eb6a002.free.beeceptor.com",
   "callBackUrl": "https://whbc4735654c6eb6a002.free.beeceptor.com",
   "transactionReference": "J21E23KAJ129"
*/

type PaybillDestinationRequest struct {
	TransactionType  string `json:"transactionType"`
	CreditAccount    string `json:"creditAccount"`
	Narration        string `json:"narration"`
	RecieverPhone    string `json:"recieverPhoneNumber"`
	Amount           string `json:"amount"`
	Timestamp        string `json:"timestamp"`
	ErrorCallBackURL string `json:"errorCallBackUrl"`
	CallBackURL      string `json:"callBackUrl"`
	TransactionRef   string `json:"transactionReference"`
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

type PesalinkPayoutRequest struct {
	Amount               string `json:"amount"`
	BankCode             string `json:"bankCode"`
	Direction            string `json:"direction"`
	Narration            string `json:"narration"`
	ReceiverPhoneNumber  string `json:"receiverPhoneNumber"`
	BeneficiaryName      string `json:"beneficiaryName"`
	CreditAccount        string `json:"creditAccount"`
	TransactionType      string `json:"transactionType"`
	TransactionReference string `json:"transactionReference"`
	Timestamp            string `json:"timestamp"`
	Currency             string `json:"currency"`
}

type PesalinkPayoutResponse struct {
	RequestID    string `json:"RequestID"`
	ResponseData struct {
		RequestID         string `json:"requestID"`
		OriginalRequestID string `json:"originalRequestID"`
		Status            string `json:"status"`
		StatusCode        string `json:"statusCode"`
		StatusDescription string `json:"statusDescription"`
		StatusMessage     string `json:"statusMessage"`
		ErrorCode         string `json:"errorCode"`
		ErrorDesc         string `json:"errorDesc"`
		EndToEndID        string `json:"endToEndID"`
	} `json:"ResponseData"`
	Type    string `json:"type"`
	Title   string `json:"title"`
	Status  int    `json:"status"`
	Detail  string `json:"detail"`
	Message string `json:"message"`
	Path    string `json:"path"`
}

type PesalinkStatusRequest struct {
	OriginalRequestID string `json:"originalRequestID"`
}

func postWithTLSFallback(url string, body []byte) ([]byte, error) {
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(body))
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
	return raw, nil
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
	raw, err := postWithTLSFallback(baseURL, body)
	if err != nil {
		return nil, err
	}

	var out TillAPIResponse
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}
	return &out, nil
}

// initiatePaybillPayment sends B2B paybill request to CreditBank gateway.
func InitiatePaybillPayment(creditAccount, narration, amount, callbackURL, transactionReference string) (*TillAPIResponse, error) {
	payload := TillDestinationRequest{
		TransactionType:  "MPESAB2B_PAYBILL",
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
	raw, err := postWithTLSFallback(paybillURL, body)
	if err != nil {
		return nil, err
	}

	var out TillAPIResponse
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}
	return &out, nil
}

func InitiatePesalinkPayout(bankCode, creditAccount, amount, currency, narration, txRef string) (*PesalinkPayoutResponse, error) {
	payload := PesalinkPayoutRequest{
		Amount:               amount,
		BankCode:             bankCode,
		Direction:            "A2A",
		Narration:            narration,
		ReceiverPhoneNumber:  "254768899729",
		BeneficiaryName:      "mam-laka",
		CreditAccount:        creditAccount,
		TransactionType:      "PESALINK",
		TransactionReference: txRef,
		Timestamp:            time.Now().Format(time.RFC3339),
		Currency:             currency,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	raw, err := postWithTLSFallback(pesalinkToAccountURL, body)
	if err != nil {
		return nil, err
	}
	var out PesalinkPayoutResponse
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, fmt.Errorf("failed to parse pesalink payout response: %w", err)
	}
	return &out, nil
}

func CheckPesalinkPayoutStatus(originalRequestID string) (*PesalinkPayoutResponse, error) {
	body, err := json.Marshal(PesalinkStatusRequest{OriginalRequestID: originalRequestID})
	if err != nil {
		return nil, err
	}
	raw, err := postWithTLSFallback(pesalinkStatusURL, body)
	if err != nil {
		return nil, err
	}
	var out PesalinkPayoutResponse
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, fmt.Errorf("failed to parse pesalink status response: %w", err)
	}
	return &out, nil
}
