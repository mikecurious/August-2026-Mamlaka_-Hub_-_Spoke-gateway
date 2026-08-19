// Package airtel is a library port of the standalone mam-laka-airtel service:
// the Airtel Money Kenya client (token, collection STK, disbursement, PIN
// encryption, MSISDN normalization, callback verification) with its Mongo
// ledger and HTTP handlers removed. Persistence, balances, merchant callbacks
// and routing live in the gateway (merchants package); this package is a pure
// client that returns typed results.
//
// Config is fail-closed and resolved at call time (Airtel is an optional rail —
// missing env must fail the request, not the process at boot).
package airtel

import (
	"errors"
	"fmt"
	"os"
	"strings"
)

// BaseURL resolves the Airtel API base URL. An explicit AIRTEL_BASE_URL wins;
// otherwise AIRTEL_ENV selects uat/live. Unlike the standalone there is NO
// silent UAT default — an unset environment is a hard error so a misconfigured
// deployment can never quietly talk to sandbox.
func BaseURL() (string, error) {
	if b := strings.TrimSpace(os.Getenv("AIRTEL_BASE_URL")); b != "" {
		return b, nil
	}
	switch strings.ToLower(strings.TrimSpace(os.Getenv("AIRTEL_ENV"))) {
	case "live", "production", "prod":
		return "https://openapi.airtelkenya.com", nil
	case "uat", "sandbox", "test":
		return "https://openapiuat.airtelkenya.com", nil
	default:
		return "", errors.New("airtel: AIRTEL_ENV is not set (expected \"uat\" or \"live\") and AIRTEL_BASE_URL is empty")
	}
}

// requireEnv returns a trimmed env var or an error naming the missing var. A
// set-but-empty value is treated as missing (mirrors mpesa's fail-closed style).
func requireEnv(key string) (string, error) {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v, nil
	}
	return "", fmt.Errorf("airtel: %s is not configured", key)
}
