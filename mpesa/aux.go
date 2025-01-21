package mpesa

import (
	"bytes"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"
)

const (
	consumerKey       = "4lfHAgdiHnIf3c5fMaaNSKOKpdHyO5xbT38iT0suJzJg0nbw"
	consumerSecret    = "QFgAD4B6mGCG1Hn9aCsLYhDLGn3p8RFpnoE0HNFhzcuzQxw8ccjww2C8xHue61nO"
	businessShortCode = "4904606"
	passKey           = "5aa7cbe3bb62309914d03219211169df449418b3b723f959e1210429b8f6e425"
	phoneNumber       = "254768899729" // Replace with a valid phone number
	callbackURL       = "https://example.com/callback"
	accountReference  = "Account123"
	amount            = 1
)

type StkPushResponse struct {
	MerchantRequestID   string `json:"MerchantRequestID"`
	CheckoutRequestID   string `json:"CheckoutRequestID"`
	ResponseCode        string `json:"ResponseCode"`
	ResponseDescription string `json:"ResponseDescription"`
	CustomerMessage     string `json:"CustomerMessage"`
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

// func main() {
// 	// Replace these with your values

// 	// Generate timestamp in required format
// 	timestamp := time.Now().Format("20060102150405")

// 	// Generate password
// 	password := base64.StdEncoding.EncodeToString([]byte(businessShortCode + passKey + timestamp))

// 	// Generate access token
// 	token, err := generateAccessToken(consumerKey, consumerSecret)
// 	if err != nil {
// 		fmt.Println("Error generating access token:", err)
// 		return
// 	}

// 	// Initiate STK Push
// 	err = stkPush(token, businessShortCode, password, timestamp, phoneNumber, amount, callbackURL, accountReference)
// 	if err != nil {
// 		fmt.Println("Error initiating STK push:", err)
// 		return
// 	}

// 	fmt.Println("STK Push initiated successfully")
// }

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
		CallBackURL:       callbackURL,
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
