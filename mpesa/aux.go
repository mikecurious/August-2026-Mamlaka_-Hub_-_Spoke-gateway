package mpesa

import (
	"bytes"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"regexp"
	"sync"
	"time"
)

// callbackBase is the public base URL Safaricom posts asynchronous results to.
// It is configurable so a single binary can serve boxes published under
// different hostnames during a staged migration: a box reached at one hostname
// must have its callbacks delivered back to itself, not to whichever box the
// legacy hostname happens to resolve to. Defaults to the legacy host, so an
// existing deployment that sets nothing behaves exactly as before.
func callbackBase() string {
	if v := strings.TrimSpace(os.Getenv("CALLBACK_BASE_URL")); v != "" {
		return strings.TrimRight(v, "/")
	}
	return "https://payments.mam-laka.com"
}

var safaricomHTTPClient = &http.Client{
	Timeout: 20 * time.Second,
	Transport: &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   5 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		MaxIdleConns:          200,
		MaxIdleConnsPerHost:   100,
		MaxConnsPerHost:       100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   5 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	},
}

type cachedAccessToken struct {
	token     string
	expiresAt time.Time
}

var (
	tokenCacheMu sync.Mutex
	tokenCache   = make(map[string]cachedAccessToken)
)


// revert amout  using the api

type StkPushResponse struct {
	MerchantRequestID   string `json:"MerchantRequestID"`
	CheckoutRequestID   string `json:"CheckoutRequestID"`
	ResponseCode        string `json:"ResponseCode"`
	ResponseDescription string `json:"ResponseDescription"`
	CustomerMessage     string `json:"CustomerMessage"`
}

type B2BResponse struct {
	ConversationID           string `json:"ConversationID"`
	OriginatorConversationID string `json:"OriginatorConversationID"`
	ResponseCode             string `json:"ResponseCode"`
	ResponseDescription      string `json:"ResponseDescription"`
}

// Structs for the JSON payload and responses
type TokenResponse struct {
	AccessToken string `json:"access_token"`
	ExpiresIn   string `json:"expires_in"`
}

type StkPushRequest struct {
	BusinessShortCode string `json:"BusinessShortCode"`
	Password          string `json:"Password"`
	Timestamp         string `json:"Timestamp"`
	TransactionType   string `json:"TransactionType"`
	Amount            int    `json:"Amount"`
	PartyA            string `json:"PartyA"`
	PartyB            string `json:"PartyB"`
	PhoneNumber       string `json:"PhoneNumber"`
	CallBackURL       string `json:"CallBackURL"`
	AccountReference  string `json:"AccountReference"`
	TransactionDesc   string `json:"TransactionDesc"`
}

type STKStatusQueryRequest struct {
	BusinessShortCode string `json:"BusinessShortCode"`
	Password          string `json:"Password"`
	Timestamp         string `json:"Timestamp"`
	CheckoutRequestID string `json:"CheckoutRequestID"`
}

type STKStatusQueryResponse struct {
	ResponseCode        string `json:"ResponseCode,omitempty"`
	ResponseDescription string `json:"ResponseDescription,omitempty"`
	MerchantRequestID   string `json:"MerchantRequestID,omitempty"`
	CheckoutRequestID   string `json:"CheckoutRequestID,omitempty"`
	ResultCode          string `json:"ResultCode,omitempty"`
	ResultDesc          string `json:"ResultDesc,omitempty"`
	RequestID           string `json:"requestId,omitempty"`
	ErrorCode           string `json:"errorCode,omitempty"`
	ErrorMessage        string `json:"errorMessage,omitempty"`
}

type TransactionStatusQueryRequest struct {
	Initiator          string `json:"Initiator"`
	SecurityCredential string `json:"SecurityCredential"`
	CommandID          string `json:"CommandID"`
	TransactionID      string `json:"TransactionID"`
	PartyA             string `json:"PartyA"`
	IdentifierType     string `json:"IdentifierType"`
	ResultURL          string `json:"ResultURL"`
	QueueTimeOutURL    string `json:"QueueTimeOutURL"`
	Remarks            string `json:"Remarks"`
	Occasion           string `json:"Occasion"`
}

type TransactionStatusQueryResponse struct {
	OriginatorConversationID string `json:"OriginatorConversationID,omitempty"`
	ConversationID           string `json:"ConversationID,omitempty"`
	ResponseCode             string `json:"ResponseCode,omitempty"`
	ResponseDescription      string `json:"ResponseDescription,omitempty"`
	ErrorCode                string `json:"errorCode,omitempty"`
	ErrorMessage             string `json:"errorMessage,omitempty"`
	RequestID                string `json:"requestId,omitempty"`
}

func GenerateAccessToken(consumerKey, consumerSecret string) (string, error) {
	cacheKey := consumerKey + ":" + consumerSecret
	tokenCacheMu.Lock()
	if cached, ok := tokenCache[cacheKey]; ok && cached.token != "" && time.Now().Before(cached.expiresAt) {
		tokenCacheMu.Unlock()
		return cached.token, nil
	}
	tokenCacheMu.Unlock()

	url := "https://api.safaricom.co.ke/oauth/v1/generate?grant_type=client_credentials"

	// Create the basic auth header
	credentials := base64.StdEncoding.EncodeToString([]byte(consumerKey + ":" + consumerSecret))
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Basic "+credentials)

	resp, err := safaricomHTTPClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to get access token: %s", resp.Status)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var tokenResponse TokenResponse
	err = json.Unmarshal(body, &tokenResponse)
	if err != nil {
		return "", err
	}

	tokenCacheMu.Lock()
	tokenCache[cacheKey] = cachedAccessToken{
		token:     tokenResponse.AccessToken,
		expiresAt: time.Now().Add(50 * time.Minute),
	}
	tokenCacheMu.Unlock()

	return tokenResponse.AccessToken, nil
}

// alias payins
func StkPush(phoneNumber string, amount int, callbackURL, accountReference, consumerKey, consumerSecret, businessShortCode, passKey string) (*StkPushResponse, error) {
	url := "https://api.safaricom.co.ke/mpesa/stkpush/v1/processrequest"
	timestamp := time.Now().Format("20060102150405")
	token, err := GenerateAccessToken(consumerKey, consumerSecret)
	if err != nil {
		return nil, err
	}

	password := base64.StdEncoding.EncodeToString([]byte(businessShortCode + passKey + timestamp))

	requestBody := StkPushRequest{
		BusinessShortCode: businessShortCode,
		Password:          password,
		Timestamp:         timestamp,
		TransactionType:   "CustomerPayBillOnline",
		Amount:            amount,
		PartyA:            phoneNumber,
		PartyB:            businessShortCode,
		PhoneNumber:       phoneNumber,
		CallBackURL:       callbackBase() + "/api/v1/mobile/callback",
		AccountReference:  accountReference,
		TransactionDesc:   accountReference,
	}

	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := safaricomHTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := ioutil.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to initiate STK push: %s, %s", resp.Status, string(body))
	}

	// Parse the response body
	var stkResponse StkPushResponse
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(body, &stkResponse)
	if err != nil {
		return nil, err
	}

	// Log or print the response for debugging
	fmt.Println("STK Push Response:", stkResponse)

	// Return the parsed response
	return &stkResponse, nil
}

func QuerySTKStatus(checkoutRequestID string, creds STKCredentials) (*STKStatusQueryResponse, error) {
	url := "https://api.safaricom.co.ke/mpesa/stkpushquery/v1/query"
	timestamp := time.Now().Format("20060102150405")
	token, err := GenerateAccessToken(creds.ConsumerKey, creds.ConsumerSecret)
	if err != nil {
		return nil, err
	}

	password := base64.StdEncoding.EncodeToString([]byte(creds.BusinessShortCode + creds.PassKey + timestamp))
	requestBody := STKStatusQueryRequest{
		BusinessShortCode: creds.BusinessShortCode,
		Password:          password,
		Timestamp:         timestamp,
		CheckoutRequestID: checkoutRequestID,
	}

	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := safaricomHTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var queryResponse STKStatusQueryResponse
	if err := json.Unmarshal(body, &queryResponse); err != nil {
		return nil, fmt.Errorf("failed to parse STK query response: %w. Raw body: %s", err, string(body))
	}

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		if queryResponse.ErrorMessage != "" {
			return &queryResponse, nil
		}
		return nil, fmt.Errorf("STK query failed: %s, %s", resp.Status, string(body))
	}

	return &queryResponse, nil
}

func QueryB2CTransactionStatus(transactionID string, creds B2CCredentials) (*TransactionStatusQueryResponse, error) {
	if strings.TrimSpace(transactionID) == "" {
		return nil, fmt.Errorf("transaction status query: empty transaction id")
	}
	url := "https://api.safaricom.co.ke/mpesa/transactionstatus/v1/query"
	token, err := GenerateAccessToken(creds.ConsumerKey, creds.ConsumerSecret)
	if err != nil {
		return nil, err
	}

	requestBody := TransactionStatusQueryRequest{
		Initiator:          creds.InitiatorName,
		SecurityCredential: creds.SecurityCredential,
		CommandID:          "TransactionStatusQuery",
		TransactionID:      transactionID,
		PartyA:             creds.BusinessShortCode,
		IdentifierType:     "4",
		ResultURL:          callbackBase() + "/api/v1/mobile/b2c/callback",
		QueueTimeOutURL:    callbackBase() + "/api/v1/mobile/b2c/callback",
		Remarks:            transactionID,
		Occasion:           "Withdrawal status query",
	}

	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := safaricomHTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var queryResponse TransactionStatusQueryResponse
	if err := json.Unmarshal(body, &queryResponse); err != nil {
		return nil, fmt.Errorf("failed to parse transaction status query response: %w. Raw body: %s", err, string(body))
	}

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		if queryResponse.ErrorMessage != "" {
			return &queryResponse, nil
		}
		return nil, fmt.Errorf("transaction status query failed: %s, %s", resp.Status, string(body))
	}

	return &queryResponse, nil
}

func GenerateSecureID() string {
	b := make([]byte, 16) // Generate 16 random bytes
	_, _ = rand.Read(b)
	return base64.URLEncoding.EncodeToString(b)
}

type B2CRequest struct {
	OriginatorConversationID string  `json:"OriginatorConversationID"`
	InitiatorName            string  `json:"InitiatorName"`
	SecurityCredential       string  `json:"SecurityCredential"`
	CommandID                string  `json:"CommandID"`
	Amount                   float64 `json:"Amount"`
	PartyA                   string  `json:"PartyA"`
	PartyB                   string  `json:"PartyB"`
	Remarks                  string  `json:"Remarks"`
	QueueTimeOutURL          string  `json:"QueueTimeOutURL"`
	ResultURL                string  `json:"ResultURL"`
	Occassion                string  `json:"Occassion"`
}

func generateB2BAccessToken(consumerKey, consumerSecret string) (string, error) {
	return GenerateAccessToken(consumerKey, consumerSecret)
}

func GenerateB2CRequest(phoneNumber string, amount float64, callbackURL, externalID string, identifier, consumerKey, consumerSecret, password, businessShortCode, initiatorName string) (*B2BResponse, error) {
	return GenerateB2CRequestWithCommand(phoneNumber, amount, callbackURL, externalID, identifier, consumerKey, consumerSecret, password, businessShortCode, initiatorName, "PromotionPayment")
}

func GenerateB2CRequestWithCommand(phoneNumber string, amount float64, callbackURL, externalID string, identifier, consumerKey, consumerSecret, password, businessShortCode, initiatorName, commandID string) (*B2BResponse, error) {
	if commandID == "" {
		commandID = "PromotionPayment"
	}

	token, err := generateB2BAccessToken(consumerKey, consumerSecret)
	if err != nil {
		return nil, fmt.Errorf("failed to generate B2C access token: %w", err)
	}

	re := regexp.MustCompile(`\D`)
	phoneNumberStr := re.ReplaceAllString(fmt.Sprintf("%s", phoneNumber), "")
	fmt.Println("Phone Number:", phoneNumberStr)
	fmt.Println("Identifier:", identifier)

	b2cRequest := B2CRequest{
		OriginatorConversationID: identifier,
		InitiatorName:            initiatorName,
		SecurityCredential:       password,
		CommandID:                commandID,
		Amount:                   amount,
		PartyA:                   businessShortCode,
		PartyB:                   phoneNumberStr,
		Remarks:                  identifier,
		QueueTimeOutURL:          callbackBase() + "/api/v1/mobile/b2c/callback",
		ResultURL:                callbackBase() + "/api/v1/mobile/b2c/callback",

		Occassion: "Ok",
	}
	// print the b2c request payload
	requestBody, err := json.Marshal(b2cRequest)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize request body: %w", err)
	}

	url := "https://api.safaricom.co.ke/mpesa/b2c/v1/paymentrequest"
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(requestBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := safaricomHTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	body, _ := ioutil.ReadAll(resp.Body)

	// Always print Safaricom's full JSON response
	fmt.Println("Safaricom Response Body:", string(body))

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("B2C request failed with status %s: %s", resp.Status, string(body))
	}

	var b2bResponse B2BResponse
	err = json.Unmarshal(body, &b2bResponse)
	if err != nil {
		return nil, fmt.Errorf("failed to parse Safaricom JSON: %w. Raw body: %s", err, string(body))
	}

	fmt.Println("Parsed Response:", b2bResponse)
	return &b2bResponse, nil
}

// RemovePlusPrefix removes the '+' sign from the beginning of a phone number if present.
func RemovePlusPrefix(phoneNumber string) string {
	if len(phoneNumber) > 0 && phoneNumber[0] == '+' {
		return phoneNumber[1:] // Remove the first character
	}
	return phoneNumber
}
