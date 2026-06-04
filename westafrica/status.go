package westafrica

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// ErrPixelTransactionNotFound is returned when Pixel reports the transaction id does not exist for the API key used.
var ErrPixelTransactionNotFound = errors.New("pixel transaction not found")

// TransactionStatusRequest is the Pixel status query payload.
type TransactionStatusRequest struct {
	APIKey         string `json:"api_key"`
	TransactionIDs string `json:"transaction_ids"`
}

// pixelStatusEnvelope handles Pixel status API where data may be an object or an array.
type pixelStatusEnvelope struct {
	Data       json.RawMessage `json:"data"`
	Message    string          `json:"message"`
	StatusCode int             `json:"statut_code"`
}

// pixelTxnData mirrors transaction fields returned by Pixel status/airtime APIs.
type pixelTxnData struct {
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
	Currency           string  `json:"currency"`
}

func (d pixelTxnData) toAirtimeResponse(msg string, statusCode int) *AirtimeResponse {
	var resp AirtimeResponse
	resp.Message = msg
	resp.StatusCode = statusCode
	resp.Data.TransactionID = d.TransactionID
	resp.Data.Amount = d.Amount
	resp.Data.Benefice = d.Benefice
	resp.Data.Commission = d.Commission
	resp.Data.Destination = d.Destination
	resp.Data.Fee = d.Fee
	resp.Data.Response = d.Response
	resp.Data.Error = d.Error
	resp.Data.ServiceID = d.ServiceID
	resp.Data.CustomerName = d.CustomerName
	resp.Data.State = d.State
	resp.Data.CustomData = d.CustomData
	resp.Data.IPNUrl = d.IPNUrl
	resp.Data.TransactionChannel = d.TransactionChannel
	resp.Data.ProviderID = d.ProviderID
	resp.Data.SMSLink = d.SMSLink
	resp.Data.PID = d.PID
	resp.Data.PLastWalletAmount = d.PLastWalletAmount
	resp.Data.PNewWalletAmount = d.PNewWalletAmount
	resp.Data.CreatedAt = d.CreatedAt
	resp.Data.UpdatedAt = d.UpdatedAt
	resp.Data.Currency = d.Currency
	return &resp
}

func parsePixelStatusBody(body []byte, requestedID string) (*AirtimeResponse, error) {
	var env pixelStatusEnvelope
	if err := json.Unmarshal(body, &env); err != nil {
		return nil, fmt.Errorf("parse status response: %w", err)
	}

	msgLower := strings.ToLower(strings.TrimSpace(env.Message))
	if env.StatusCode == http.StatusNotFound || strings.Contains(msgLower, "transaction not found") {
		return nil, ErrPixelTransactionNotFound
	}

	raw := bytes.TrimSpace(env.Data)
	if len(raw) == 0 {
		return nil, fmt.Errorf("empty transaction data from provider")
	}

	var txn pixelTxnData
	var err error

	switch raw[0] {
	case '{':
		err = json.Unmarshal(raw, &txn)
	case '[':
		txn, err = parsePixelStatusDataArray(raw, requestedID)
	default:
		return nil, fmt.Errorf("unexpected data type from provider")
	}
	if err != nil {
		if errors.Is(err, ErrPixelTransactionNotFound) {
			return nil, err
		}
		return nil, fmt.Errorf("parse status data: %w", err)
	}

	if strings.TrimSpace(txn.TransactionID) == "" {
		return nil, fmt.Errorf("empty transaction data from provider")
	}

	return txn.toAirtimeResponse(env.Message, env.StatusCode), nil
}

// parsePixelStatusDataArray handles data as []string (not found) or []transaction object.
func parsePixelStatusDataArray(raw []byte, requestedID string) (pixelTxnData, error) {
	var strIDs []string
	if err := json.Unmarshal(raw, &strIDs); err == nil {
		if len(strIDs) == 0 || strings.TrimSpace(strIDs[0]) == "" {
			return pixelTxnData{}, ErrPixelTransactionNotFound
		}
		// Pixel returns e.g. data: ["PIX_22827809"] when the id is unknown for this api_key.
		return pixelTxnData{}, ErrPixelTransactionNotFound
	}

	var list []pixelTxnData
	if err := json.Unmarshal(raw, &list); err != nil {
		return pixelTxnData{}, err
	}
	if len(list) == 0 {
		return pixelTxnData{}, ErrPixelTransactionNotFound
	}

	req := strings.TrimSpace(requestedID)
	if req != "" {
		for _, item := range list {
			if strings.EqualFold(strings.TrimSpace(item.TransactionID), req) {
				return item, nil
			}
		}
	}

	return list[0], nil
}

// QueryTransactionStatus fetches a single Pixel transaction by provider id (e.g. PIX_37068773).
func (c *AirtimeClient) QueryTransactionStatus(ctx context.Context, apiKey, transactionID string) (*AirtimeResponse, error) {
	apiKey = strings.TrimSpace(apiKey)
	transactionID = strings.TrimSpace(transactionID)
	if apiKey == "" {
		return nil, fmt.Errorf("api_key is required")
	}
	if transactionID == "" {
		return nil, fmt.Errorf("transaction_id is required")
	}

	payload, err := json.Marshal(TransactionStatusRequest{
		APIKey:         apiKey,
		TransactionIDs: transactionID,
	})
	if err != nil {
		return nil, fmt.Errorf("marshal status request: %w", err)
	}

	url := fmt.Sprintf("%s/api_v1/transaction/status", c.BaseURL)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBuffer(payload))
	if err != nil {
		return nil, fmt.Errorf("create status request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("status request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read status response: %w", err)
	}

	parsed, err := parsePixelStatusBody(body, transactionID)
	if err != nil {
		if resp.StatusCode != http.StatusOK && !errors.Is(err, ErrPixelTransactionNotFound) {
			return nil, providerHTTPError(resp.StatusCode, body)
		}
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, providerHTTPError(resp.StatusCode, body)
	}
	if parsed.StatusCode != 0 && parsed.StatusCode != http.StatusOK {
		msg := strings.TrimSpace(parsed.Message)
		if msg == "" {
			msg = fmt.Sprintf("payment provider error (code %d)", parsed.StatusCode)
		}
		return nil, fmt.Errorf("%s", msg)
	}

	return parsed, nil
}

// PixelAPIKeysForCurrency returns API keys to try, in order, for a wallet currency.
func PixelAPIKeysForCurrency(currency string) []string {
	currency = strings.ToUpper(strings.TrimSpace(currency))
	core := ResolvePixelAPIKey()
	gmd := ResolvePixelGambiaAPIKey()

	var keys []string
	seen := map[string]struct{}{}
	add := func(k string) {
		k = strings.TrimSpace(k)
		if k == "" {
			return
		}
		if _, ok := seen[k]; ok {
			return
		}
		seen[k] = struct{}{}
		keys = append(keys, k)
	}

	switch currency {
	case "GMD":
		add(gmd)
		add(core)
	default:
		add(core)
		add(gmd)
	}
	return keys
}
