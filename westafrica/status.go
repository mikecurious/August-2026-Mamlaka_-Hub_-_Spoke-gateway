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
	APIKey          string `json:"api_key"`
	TransactionIDs  string `json:"transaction_ids"`
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

	var envelope AirtimeResponse
	if err := json.Unmarshal(body, &envelope); err != nil {
		return nil, fmt.Errorf("parse status response: %w", err)
	}

	if envelope.StatusCode == http.StatusNotFound {
		return nil, ErrPixelTransactionNotFound
	}
	if strings.Contains(strings.ToLower(envelope.Message), "transaction not found") {
		return nil, ErrPixelTransactionNotFound
	}
	if resp.StatusCode != http.StatusOK {
		return nil, providerHTTPError(resp.StatusCode, body)
	}
	if envelope.StatusCode != 0 && envelope.StatusCode != http.StatusOK {
		msg := strings.TrimSpace(envelope.Message)
		if msg == "" {
			msg = fmt.Sprintf("payment provider error (code %d)", envelope.StatusCode)
		}
		return nil, fmt.Errorf("%s", msg)
	}
	if strings.TrimSpace(envelope.Data.TransactionID) == "" {
		return nil, fmt.Errorf("empty transaction data from provider")
	}

	return &envelope, nil
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
