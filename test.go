package main

import (
	"bytes"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"regexp"
	"time"
)

const (
	//4130455:172f9892373eafe6dac71a87e4e8ade1792599809f7de1667c647bce03364ca7
	// 4904594
	consumerKey       = "a53D2lxIgGTgXtTDEnMo5btDnG90nOhg16GOK0MAlOOQNhBe"
	consumerSecret    = "LchqRZnB48pQfGB1WUvNhp6qqzGQ3MfBFd32sGsqvYIzvmJswghXXWA0KormP3NV"
	businessShortCode = "4130455"
	passKey           = "172f9892373eafe6dac71a87e4e8ade1792599809f7de1667c647bce03364ca7"
	phoneNumber       = "254768899729" // Replace with a valid phone number
	callbackURL       = "https://example.com/callback"
	accountReference  = "Account123"
	amount            = 1
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

func GenerateAccessToken(consumerKey, consumerSecret string) (string, error) {
	url := "https://api.safaricom.co.ke/oauth/v1/generate?grant_type=client_credentials"

	// Create the basic auth header
	credentials := base64.StdEncoding.EncodeToString([]byte(consumerKey + ":" + consumerSecret))
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Basic "+credentials)

	client := &http.Client{}
	resp, err := client.Do(req)
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

	return tokenResponse.AccessToken, nil
}

// alias payins
func StkPush(phoneNumber string, amount int, callbackURL, accountReference string) (*StkPushResponse, error) {
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
		CallBackURL:       "https://payments.mam-laka.com/api/v1/mobile/callback",
		AccountReference:  accountReference,
		TransactionDesc:   "Payment",
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

	client := &http.Client{}
	resp, err := client.Do(req)
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
	// Endpoint for generating the access token
	url := "https://api.safaricom.co.ke/oauth/v1/generate?grant_type=client_credentials"

	// Encode credentials in Base64
	credentials := base64.StdEncoding.EncodeToString([]byte(consumerKey + ":" + consumerSecret))

	// Create a new HTTP request
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	// Add headers
	req.Header.Add("Authorization", "Basic "+credentials)

	// Create HTTP client
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	// Read response body
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response body: %w", err)
	}

	// Parse JSON response to extract the access token
	var response map[string]interface{}
	if err := json.Unmarshal(body, &response); err != nil {
		return "", fmt.Errorf("failed to parse JSON response: %w", err)
	}

	// Extract and return the access token
	token, ok := response["access_token"].(string)
	if !ok {
		return "", fmt.Errorf("access token not found in response")
	}

	return token, nil
}

func GenerateB2CRequest(phoneNumber string, amount float64, callbackURL, externalID string, identifier string) (*B2BResponse, error) {
	consumer_key := "oLwt5LEkO7zkQaqV8Sy9Gs8MvgA8PFADM6VOUe4jYj98nVr1"
	consumer_secret := "YylBuouNZdeOJeU8ltCKll5QBQ0xSDrdAq7pdaurpOS8FNYPkaSAA8kZLlblwslM"

	token, _ := generateB2BAccessToken(consumer_key, consumer_secret)
	fmt.Println("Access Token:", token)

	businessShortCode := "3039805"
	password := "Xw8NWgC6K4Hnese1stlIMC0sE3p+kbcMtTVVxG57s4K/WZB2owiOf30B3yYSdTaTqdz2gv22we9sd4bgvfPVl7jynLtAglZn6KuGtdhhdy3eVQ0nosw3wZdfHDum8DCu5BAI/jU+x32PMSB/vtx9bbreV0rUHEvx7Gx4CI4Eze4BnhFQ368Z2x7x9Q+82r/tZxDlgG76NbWnLfj9DHbcs5hOBoMYiMbnXg8HsLUaI688qNGqqK9CLr8uKfIgXgFBSD4Ky7P9UwWBXlTOODtmv/TRJBnrD+8IFttZqjruDxV81NGIeASl9q6Ni8go5gBGrNHGxSJ/SF5rGhloTXLtHg=="

	re := regexp.MustCompile(`\D`)
	phoneNumberStr := re.ReplaceAllString(fmt.Sprintf("%s", phoneNumber), "")
	fmt.Println("Phone Number:", phoneNumberStr)
	fmt.Println("Identifier:", identifier)

	b2cRequest := B2CRequest{
		OriginatorConversationID: identifier,
		InitiatorName:            "b2cInit",
		SecurityCredential:       password,
		CommandID:                "PromotionPayment",
		Amount:                   amount,
		PartyA:                   businessShortCode,
		PartyB:                   phoneNumber,
		Remarks:                  "payments done",
		QueueTimeOutURL:          "https://payments.mam-laka.com/api/v1/mobile/callback",
		// ResultURL:                "https://payments.mam-laka.com/api/v1/mobile/callback",
		ResultURL: "https://webhook.site/00f617b7-7815-4bd6-b233-1e115aa8671e",

		Occassion: "Ok",
	}

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

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
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

// a main test function to test b2c
func main2() {
	phoneNumber := "254768899729"
	amount := 10.0
	callbackURL := "https://webhook.site/c7edfd71-ae2e-4a15-9d7a-c4132c05e746"
	externalID := "External1239000"
	identifier := GenerateSecureID()

	response, err := GenerateB2CRequest(phoneNumber, amount, callbackURL, externalID, identifier)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}

	fmt.Println("B2C Response:", response)
}
