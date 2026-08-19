package airtel

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"os"
	"strings"
)

// Callback is the envelope Airtel POSTs to the registered callback URL for both
// collections and disbursements.
type Callback struct {
	Transaction struct {
		ID            string `json:"id"`             // our reference (correlation key)
		Message       string `json:"message"`
		StatusCode    string `json:"status_code"`
		AirtelMoneyID string `json:"airtel_money_id"` // provider receipt
	} `json:"transaction"`
	Hash string `json:"hash"`
}

// ParseCallback unmarshals the raw callback body.
func ParseCallback(body []byte) (*Callback, error) {
	var cb Callback
	if err := json.Unmarshal(body, &cb); err != nil {
		return nil, err
	}
	return &cb, nil
}

// VerifyCallbackHash authenticates an inbound Airtel callback. The "hash" is
// HMAC-SHA256 over the RAW "transaction" object bytes, keyed with
// AIRTEL_CALLBACK_PRIVATE_KEY, base64-encoded.
//
// Fail-closed: unlike the standalone (which accepted callbacks unverified when
// the key was unset), a missing key is an ERROR here — a collections callback
// credits money, so the gateway must reject anything it cannot authenticate.
func VerifyCallbackHash(body []byte) (bool, error) {
	key := strings.TrimSpace(os.Getenv("AIRTEL_CALLBACK_PRIVATE_KEY"))
	if key == "" {
		return false, errors.New("airtel: AIRTEL_CALLBACK_PRIVATE_KEY is not configured (callbacks are fail-closed)")
	}

	var envelope struct {
		Transaction json.RawMessage `json:"transaction"`
		Hash        string          `json:"hash"`
	}
	if err := json.Unmarshal(body, &envelope); err != nil {
		return false, err
	}
	if envelope.Hash == "" || len(envelope.Transaction) == 0 {
		return false, errors.New("airtel: missing transaction or hash in callback body")
	}

	mac := hmac.New(sha256.New, []byte(key))
	mac.Write(envelope.Transaction)
	expected := mac.Sum(nil)

	got, err := base64.StdEncoding.DecodeString(envelope.Hash)
	if err != nil {
		return false, err
	}
	return hmac.Equal(expected, got), nil
}
