package mpesa

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const sfcVerifyURL = "https://api.safaricom.co.ke/sfcverify/v1/query/info"

const (
	IdentifierTypeTill    = "2"
	IdentifierTypePaybill = "4"
)

// SFCVerifyRequest is the Safaricom identifier lookup payload.
type SFCVerifyRequest struct {
	IdentifierType string `json:"IdentifierType"`
	Identifier     string `json:"Identifier"`
}

// SFCVerifyResponse is the Safaricom identifier lookup response.
type SFCVerifyResponse struct {
	ConversationID          string `json:"ConversationID"`
	ResponseCode            string `json:"ResponseCode"`
	ResponseMessage         string `json:"ResponseMessage"`
	DetailedMessage         string `json:"DetailedMessage"`
	TillNumber              string `json:"TillNumber"`
	OrganizationShortCode   string `json:"OrganizationShortCode"`
	OrganizationName        string `json:"OrganizationName"`
	ChargeProfileID         string `json:"ChargeProfileID"`
}

func (r *SFCVerifyResponse) IsSuccess() bool {
	return strings.TrimSpace(r.ResponseCode) == "4000"
}

func (r *SFCVerifyResponse) DisplayNumber() string {
	if n := strings.TrimSpace(r.TillNumber); n != "" {
		return n
	}
	return strings.TrimSpace(r.OrganizationShortCode)
}

// QueryIdentifierInfo looks up a till (type 2) or paybill (type 4) via Safaricom SFC verify.
func QueryIdentifierInfo(consumerKey, consumerSecret, identifierType, identifier string) (*SFCVerifyResponse, error) {
	identifierType = strings.TrimSpace(identifierType)
	identifier = strings.TrimSpace(identifier)
	if identifierType != IdentifierTypeTill && identifierType != IdentifierTypePaybill {
		return nil, fmt.Errorf("invalid identifier type: %s", identifierType)
	}
	if identifier == "" {
		return nil, fmt.Errorf("identifier is required")
	}

	token, err := GenerateAccessToken(consumerKey, consumerSecret)
	if err != nil {
		return nil, fmt.Errorf("oauth token: %w", err)
	}

	body, err := json.Marshal(SFCVerifyRequest{
		IdentifierType: identifierType,
		Identifier:     identifier,
	})
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest(http.MethodPost, sfcVerifyURL, bytes.NewBuffer(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	client := &http.Client{Timeout: 45 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("sfc verify request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("sfc verify HTTP %d: %s", resp.StatusCode, string(respBody))
	}

	var out SFCVerifyResponse
	if err := json.Unmarshal(respBody, &out); err != nil {
		return nil, fmt.Errorf("parse sfc verify response: %w", err)
	}

	return &out, nil
}

// ResolveIdentifierType maps API type string to Safaricom IdentifierType.
func ResolveIdentifierType(kind string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(kind)) {
	case "till", "2":
		return IdentifierTypeTill, nil
	case "paybill", "pay_bill", "4":
		return IdentifierTypePaybill, nil
	default:
		return "", fmt.Errorf("type must be till or paybill")
	}
}
