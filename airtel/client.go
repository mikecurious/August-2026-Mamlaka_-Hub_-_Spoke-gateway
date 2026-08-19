package airtel

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"
)

// Result is the parsed outcome of an Airtel collection or disbursement call.
type Result struct {
	HTTPStatus    int
	AirtelMoneyID string // provider receipt (-> providerReference)
	ReferenceID   string // Airtel's own transaction id
	StatusCode    string // Airtel code: TS / TF / TA / TIP / "" ...
	Message       string
	ResponseCode  string
	ResultCode    string
	Raw           string
	parsedOK      bool
}

// airtelEnvelope mirrors the shape both /merchant/v1/payments and
// /standard/v2/disbursements answer with.
type airtelEnvelope struct {
	Data struct {
		Transaction struct {
			AirtelMoneyID string `json:"airtel_money_id"`
			ID            string `json:"id"`
			ReferenceID   string `json:"reference_id"`
			Status        string `json:"status"`
		} `json:"transaction"`
	} `json:"data"`
	Status struct {
		Code         string `json:"code"`
		Message      string `json:"message"`
		ResponseCode string `json:"response_code"`
		ResultCode   string `json:"result_code"`
		Success      bool   `json:"success"`
	} `json:"status"`
}

func httpStatusOK(s int) bool { return s >= http.StatusOK && s < http.StatusMultipleChoices }

// MapStatus translates an Airtel transaction status code into the gateway's own
// terminal states. Anything unrecognised stays PENDING so a callback can settle
// it.
func MapStatus(code string) string {
	switch code {
	case "TS":
		return "COMPLETE"
	case "TF":
		return "FAILED"
	default:
		return "PENDING"
	}
}

// CollectionStatus is deliberately forgiving: an STK push is only an
// acknowledgement, so a 2xx without a transaction status is normal and must
// stay PENDING for the callback to settle. Only a terminal code settles here.
func (r *Result) CollectionStatus() string {
	if !r.parsedOK || !httpStatusOK(r.HTTPStatus) {
		return "FAILED"
	}
	if r.StatusCode != "" {
		return MapStatus(r.StatusCode)
	}
	return "PENDING"
}

// DisbursementStatus is strict: a disbursement settles synchronously, so a
// non-2xx, unparseable, or status-less body means the money did NOT move and is
// recorded FAILED.
func (r *Result) DisbursementStatus() string {
	if !r.parsedOK || !httpStatusOK(r.HTTPStatus) || r.StatusCode == "" {
		return "FAILED"
	}
	return MapStatus(r.StatusCode)
}

// InitiateSTKPush requests a customer payment (C2B). reference is our globally
// unique correlation id — Airtel echoes it back as callback.transaction.id.
func InitiateSTKPush(msisdn string, amount int, reference string) (*Result, error) {
	base, err := BaseURL()
	if err != nil {
		return nil, err
	}
	token, err := GetValidAccessToken()
	if err != nil {
		return nil, err
	}
	payload := map[string]interface{}{
		"reference": reference,
		"subscriber": map[string]string{
			"country":  "KE",
			"currency": "KES",
			"msisdn":   msisdn,
		},
		"transaction": map[string]interface{}{
			"amount":   amount,
			"country":  "KE",
			"currency": "KES",
			"id":       reference,
		},
	}
	return airtelPost(base+"/merchant/v1/payments/", token, payload, reference, false)
}

// Disburse sends money to a subscriber (B2C/B2B). txType is "B2C" or "B2B".
func Disburse(msisdn string, amount int, reference, txType, name string) (*Result, error) {
	base, err := BaseURL()
	if err != nil {
		return nil, err
	}
	token, err := GetValidAccessToken()
	if err != nil {
		return nil, err
	}
	pin, err := EncryptPIN(token)
	if err != nil {
		return nil, err
	}

	payee := map[string]string{"msisdn": msisdn, "currency": "KES"}
	if n := strings.TrimSpace(name); n != "" {
		payee["name"] = n
	}
	payload := map[string]interface{}{
		"payee":     payee,
		"reference": reference,
		"pin":       pin,
		"transaction": map[string]interface{}{
			"amount": amount,
			"id":     reference,
			"type":   txType,
		},
	}
	return airtelPost(base+"/standard/v2/disbursements/", token, payload, reference, true)
}

// ResolveDisbursementType validates the caller's type; empty defaults to B2C.
func ResolveDisbursementType(rawType string) (string, error) {
	switch strings.ToUpper(strings.TrimSpace(rawType)) {
	case "":
		return "B2C", nil
	case "B2C", "B2B":
		return strings.ToUpper(strings.TrimSpace(rawType)), nil
	default:
		return "", fmt.Errorf("airtel: unsupported disbursement type %q (want B2C or B2B)", rawType)
	}
}

func airtelPost(url, token string, payload map[string]interface{}, reference string, hasPin bool) (*Result, error) {
	bodyBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer(bodyBytes))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("X-Country", "KE")
	req.Header.Set("X-Currency", "KES")

	// Never log the PIN; log the URL/reference only.
	log.Printf("airtel outgoing: url=%s reference=%s", url, reference)

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBytes, _ := io.ReadAll(resp.Body)
	log.Printf("airtel response: url=%s reference=%s http_status=%d body=%s", url, reference, resp.StatusCode, string(respBytes))

	return parseResult(resp.StatusCode, respBytes), nil
}

func parseResult(httpStatus int, respBytes []byte) *Result {
	r := &Result{HTTPStatus: httpStatus, Raw: string(respBytes)}
	var env airtelEnvelope
	if err := json.Unmarshal(respBytes, &env); err == nil {
		r.parsedOK = true
		t := env.Data.Transaction
		r.AirtelMoneyID = t.AirtelMoneyID
		r.ReferenceID = t.ReferenceID
		r.StatusCode = t.Status
		r.Message = env.Status.Message
		r.ResponseCode = env.Status.ResponseCode
		r.ResultCode = env.Status.ResultCode
	}
	return r
}

// ErrRailDisabled is returned by helpers when Airtel is not configured; callers
// map it to a service-unavailable style response.
var ErrRailDisabled = errors.New("airtel: rail not configured")
