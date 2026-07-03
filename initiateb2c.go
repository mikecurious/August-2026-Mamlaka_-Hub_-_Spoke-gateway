package main

// W0LXtRIf33TqQuSQLevyqTyvO847tqYMB3WauCFdYGRF6PiXZj770OhmG3lJnX4cVMrsm3K258oUx7y2p6MCs1V+W8YXNM3oqfb9pOoXFqnSmyTammvSetJct3/w0UT+0FUJTrg8JXH6j0FYlsqibCXc8f9ATb+twMi4Mxm37Ehu7fNOP40c6BHO7Cp4HHUa5yHjAVOSNDioWZr39bzyvBIiCT9Az/aISj060rCMZLWGlUlbpKVVQFqAwh/flu8AXxVqT2/zhg5NOPjhHb5i8uyau1IhN9LxC9nOGBKqInmlj59g4qTUPlSY1apZ3Y+Ny6iPN3yyseq3mm4UXKxGAg==

// package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"time"
)

type StkPushResponse struct {
	MerchantRequestID   string `json:"MerchantRequestID"`
	CheckoutRequestID   string `json:"CheckoutRequestID"`
	ResponseCode        string `json:"ResponseCode"`
	ResponseDescription string `json:"ResponseDescription"`
	CustomerMessage     string `json:"CustomerMessage"`
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
type TokenResponse struct {
	AccessToken string `json:"access_token"`
	ExpiresIn   string `json:"expires_in"`
}

// const (
// 	conumerKey         = "UH4vHUz8q9IW5XQq9jo0EBPAJ35CsDAphccVp2DFCGMywROu"
// 	conumerSecret      = "LV04A9Utjv9fUmHEUJDMe7oYSiAnZ58BJrabEJrslWcGPZ9epiqx0GOGRGcR7Mo9"
// 	initiatorName      = "b2cuser"
// 	securityCredential = "XtFCm9DJyNLguEK0pEeMCNRuhNnc317n1yaQSvOsdYgoxqLSYY/NYFo0jeRwa2+X8QgGONM4RjKR0yrJiLS0JVppQczyV6+YO3vmzCih8kctjszQiXbmG0D97B3S7ghm52FetpZge/s4AGaWbnAUBhN+ajB3BZBdWv7SOLeIcRltKwvnTOc7jWj7QwA1ktbz+cAtZMhxnM6BwqlSDPum/SThHeWaDDtI9ilEwlpL8OvSPqgghBYG7Dbvdp5AgvFHx9quexdEtMvgt4adHX/3du0k0sfP0Dp5OJDhKibt8JMrhrvz+4yRRwM0aftBOTQW1fWU6WPWFEElAqgfW11VPg=="
// 	shortCode          = "4186621"
// 	callbackURL        = "https://webhook.site/f1c53d6b-ec4d-497b-baf7-4c87cdc298eb"
// 	b2cURL             = "https://api.safaricom.co.ke/mpesa/b2c/v1/paymentrequest"
// 	tokenURL           = "https://api.safaricom.co.ke/oauth/v1/generate?grant_type=client_credentials"
// 	balanceURL         = "https://api.safaricom.co.ke/mpesa/accountbalance/v1/query"
// 	transactionURL     = "https://api.safaricom.co.ke/mpesa/transactionstatus/v1/query"
// )

type B2CRequest struct {
	InitiatorName          string `json:"InitiatorName"`
	SecurityCredential     string `json:"SecurityCredential"`
	CommandID              string `json:"CommandID"`
	Amount                 string `json:"Amount"`
	PartyA                 string `json:"PartyA"`
	PartyB                 string `json:"PartyB"`
	Remarks                string `json:"Remarks"`
	QueueTimeOutURL        string `json:"QueueTimeOutURL"`
	ResultURL              string `json:"ResultURL"`
	Occasion               string `json:"Occasion"`
	ReceiverIdentifierType string `json:"RecieverIdentifierType"`
	SenderIdentifierType   string `json:"SenderIdentifierType"`
}

type B2PRequest struct {
	OriginatorConversationID string `json:"OriginatorConversationID"`
	InitiatorName            string `json:"InitiatorName"`
	SecurityCredential       string `json:"SecurityCredential"`
	CommandID                string `json:"CommandID"`
	Amount                   string `json:"Amount"`
	PartyA                   string `json:"PartyA"`
	PartyB                   string `json:"PartyB"`
	Remarks                  string `json:"Remarks"`
	QueueTimeOutURL          string `json:"QueueTimeOutURL"`
	ResultURL                string `json:"ResultURL"`
	Occassion                string `json:"Occassion"`
}

type AccessTokenResponse struct {
	AccessToken string `json:"access_token"`
	ExpiresIn   string `json:"expires_in"`
}

func getAccessTokenFor(url, consumerKey, consumerSecret string) (string, error) {
	client := &http.Client{Timeout: 30 * time.Second}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", err
	}

	auth := base64.StdEncoding.EncodeToString([]byte(consumerKey + ":" + consumerSecret))
	req.Header.Add("Authorization", "Basic "+auth)

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("token request failed with status %s: %s", resp.Status, string(body))
	}

	var tokenResp AccessTokenResponse
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return "", fmt.Errorf("failed to parse token response: %v. Raw body: %s", err, string(body))
	}
	if tokenResp.AccessToken == "" {
		return "", fmt.Errorf("token response did not include access_token. Raw body: %s", string(body))
	}

	return tokenResp.AccessToken, nil
}

type PaymentInput struct {
	Phone  string `json:"phone" binding:"required"`
	Amount string `json:"amount" binding:"required"`
}

type TransactionStatusInput struct {
	TransactionID string `json:"transaction_id" binding:"required"`
}

func getAccessToken() (string, error) {
	return getAccessTokenFor(tokenURL, conumerKey, conumerSecret)
}

func sendB2CPayment(phone string, amount string) (map[string]interface{}, error) {
	log.Println("Sending B2C payment to:", phone, "Amount:", amount)
	token, err := getAccessToken()
	if err != nil {
		return nil, fmt.Errorf("failed to get access token: %v", err)
	}

	payload := B2CRequest{
		InitiatorName:      initiatorName,
		SecurityCredential: securityCredential,
		CommandID:          "BusinessPayment",
		Amount:             amount,
		PartyA:             shortCode,
		PartyB:             phone,
		Remarks:            "B2C Payment",
		QueueTimeOutURL:    "https://webhook.site/94ea9cd2-5230-4473-ab71-67c69a2a593d",
		ResultURL:          "https://webhook.site/94ea9cd2-5230-4473-ab71-67c69a2a593d",
		Occasion:           "Withdrawal",
	}

	payloadBytes, _ := json.Marshal(payload)
	req, _ := http.NewRequest("POST", b2cURL, bytes.NewBuffer(payloadBytes))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("B2C request error: %v", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("B2C request failed with status %s: %s", resp.Status, string(body))
	}

	var pretty map[string]interface{}
	if err := json.Unmarshal(body, &pretty); err == nil {
		return pretty, nil
	}
	return map[string]interface{}{"raw_response": string(body)}, nil
}

// send pochi

func sendB2PPayment(phone string, amount string) (map[string]interface{}, error) {
	log.Println("Sending B2P payment to:", phone, "Amount:", amount)
	token, err := getAccessTokenFor(b2pTokenURL, b2pConsumerKey, b2pConsumerSecret)
	if err != nil {
		return nil, fmt.Errorf("failed to get B2P access token: %v", err)
	}

	callbackURL := "https://webhook.site/d3ea515d-266b-477e-aa39-1d9299d8a487"
	originatorConversationID := fmt.Sprintf("%s_Test_%d", b2pShortCode, time.Now().UnixNano())

	payload := B2PRequest{
		OriginatorConversationID: originatorConversationID,
		InitiatorName:            b2pInitiatorName,
		SecurityCredential:       b2pSecurityCredential,
		CommandID:                "BusinessPayToPochi",
		Amount:                   amount,
		PartyA:                   b2pShortCode,
		PartyB:                   phone,
		Remarks:                  "B2P Payment",
		QueueTimeOutURL:          callbackURL,
		ResultURL:                callbackURL,
		Occassion:                "Withdrawal",
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal B2P payload: %v", err)
	}

	req, err := http.NewRequest("POST", b2pURL, bytes.NewBuffer(payloadBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to create B2P request: %v", err)
	}

	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("B2P request error: %v", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("B2P request failed with status %s: %s", resp.Status, string(body))
	}

	var pretty map[string]interface{}
	if err := json.Unmarshal(body, &pretty); err == nil {
		return pretty, nil
	}
	return map[string]interface{}{"raw_response": string(body)}, nil
}

func sendB2BPayment(phone string, amount string) (map[string]interface{}, error) {
	log.Println("Sending B2C payment to:", phone, "Amount:", amount)
	token, err := getAccessToken()
	if err != nil {
		return nil, fmt.Errorf("failed to get access token: %v", err)
	}

	payload := B2CRequest{
		InitiatorName:          "Collins",
		SecurityCredential:     securityCredential,
		CommandID:              "BusinessPayBill",
		ReceiverIdentifierType: "4",
		SenderIdentifierType:   "4",
		Amount:                 amount,
		PartyA:                 shortCode,
		PartyB:                 phone,
		Remarks:                "B2B Payment",
		QueueTimeOutURL:        "https://webhook.site/3ca38fcd-84e7-46ec-be9a-d3e100a889bf",
		ResultURL:              "https://webhook.site/3ca38fcd-84e7-46ec-be9a-d3e100a889bf",
		Occasion:               "Withdrawal",
	}

	payloadBytes, _ := json.Marshal(payload)
	req, _ := http.NewRequest("POST", b2bURL, bytes.NewBuffer(payloadBytes))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("B2C request error: %v", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	var pretty map[string]interface{}
	if err := json.Unmarshal(body, &pretty); err == nil {
		return pretty, nil
	}
	return map[string]interface{}{"raw_response": string(body)}, nil
}

// 🔹 Check Paybill Balance
func checkBalance() (map[string]interface{}, error) {
	token, err := getAccessToken()
	if err != nil {
		return nil, err
	}

	payload := map[string]interface{}{
		"Initiator":          initiatorName,
		"SecurityCredential": securityCredential,
		"CommandID":          "AccountBalance",
		"PartyA":             shortCode,
		"IdentifierType":     "4",
		"Remarks":            "Balance Check",
		"QueueTimeOutURL":    "https://webhook.site/91ee5dd4-1347-4576-a397-862c05e176fd",
		"ResultURL":          "https://webhook.site/91ee5dd4-1347-4576-a397-862c05e176fd",
	}

	payloadBytes, _ := json.Marshal(payload)
	req, _ := http.NewRequest("POST", balanceURL, bytes.NewBuffer(payloadBytes))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var pretty map[string]interface{}
	json.Unmarshal(body, &pretty)
	return pretty, nil
}

// 🔹 Check Transaction Status
func checkTransactionStatus(transactionID string) (map[string]interface{}, error) {
	token, err := getAccessToken()
	if err != nil {
		return nil, err
	}

	payload := map[string]interface{}{
		"Initiator":          initiatorName,
		"SecurityCredential": securityCredential,
		"CommandID":          "TransactionStatusQuery",
		"TransactionID":      transactionID,
		"PartyA":             shortCode,
		"IdentifierType":     "4",
		"ResultURL":          "https://webhook.site/6667752f-d403-4d57-9405-15bdfa9cdc31",
		"QueueTimeOutURL":    "https://webhook.site/6667752f-d403-4d57-9405-15bdfa9cdc31",
		"Remarks":            "Check transaction status",
	}

	payloadBytes, _ := json.Marshal(payload)
	req, _ := http.NewRequest("POST", transactionURL, bytes.NewBuffer(payloadBytes))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var pretty map[string]interface{}
	json.Unmarshal(body, &pretty)
	return pretty, nil
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

func StkPush(phoneNumber string, amount int, callbackURL, accountReference, consumerKey, consumerSecret, businessShortCode, passKey string) (*StkPushResponse, error) {
	url := "https://api.safaricom.co.ke/mpesa/stkpush/v1/processrequest"
	timestamp := time.Now().Format("20060102150405")
	token, err := GenerateAccessToken(consumerKey, consumerSecret)
	fmt.Println("Access Token:", token)
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
	// fmt.Println("STK Push Response:", stkResponse)

	// Return the parsed response
	return &stkResponse, nil
}

func main101() {

	//call transaction query
	// resp, err := checkTransactionStatus("UEIEP4MNLG")
	// if err != nil {
	// 	fmt.Println("Error checking transaction status:", err)
	// }
	// fmt.Println("Transaction Status Response:", resp)

	// StkPush fetches its own OAuth token from consumerKey + consumerSecret (do not pass the token here).
	// stkResponse, err := StkPush("254768899729", 1, callbackURL, "LOCAL-TEST", conumerKey, conumerSecret, shortCode, c2bPassKey)
	// if err != nil {
	// 	log.Fatal("Error initiating STK push:", err)
	// }
	// fmt.Println("STK Push Response:", stkResponse)

	// r := gin.Default()

	// B2C Payment
	// r.POST("/send-b2c", func(c *gin.Context) {
	// 	var input PaymentInput
	// 	if err := c.ShouldBindJSON(&input); err != nil {
	// 		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	// 		return
	// 	}
	// resp, err := sendB2PPayment("254768899729", "10")
	// if err != nil {
	// 	fmt.Println("Error sending B2P payment:", err)
	// }

	// fmt.Println("B2P Payment Response:", resp)

	resp, err := sendB2CPayment("254768899729", "10")
	if err != nil {
		fmt.Println("Error sending B2P payment:", err)
	}

	fmt.Println("B2P Payment Response:", resp)

	// resp2, err1 := checkBalance()

	// resp, err := sendB2BPayment("", "10")
	// if err != nil {
	// 	fmt.Println("Error sending B2B payment:", err)
	// }

	// fmt.Println("B2B Payment Response:", resp)

	// resp2, err1 := checkBalance()

	// if err1 != nil {
	// 	fmt.Println("Error checking balance:", err1)
	// }

	// fmt.Println("Balance Check Response:", resp2)

	// check balance

	// c.JSON(http.StatusOK, resp)
	// })

	// // Balance Check
	// r.GET("/check-balance", func(c *gin.Context) {
	// resp, err := checkBalance()
	// if err != nil {
	// 	fmt.Println("Error checking balance:", err)
	// }
	// fmt.Println("Balance Check Response:", resp)
	// })

	// // Transaction Status
	// r.POST("/transaction-status", func(c *gin.Context) {
	// 	var input TransactionStatusInput
	// 	if err := c.ShouldBindJSON(&input); err != nil {
	// 		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	// 		return
	// 	}
	// 	resp, err := checkTransactionStatus(input.TransactionID)
	// 	if err != nil {
	// 		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
	// 		return
	// 	}
	// 	c.JSON(http.StatusOK, resp)
	// })

	// // Callback
	// r.POST("/callbackb2b", func(c *gin.Context) {
	// 	var body map[string]interface{}
	// 	if err := c.BindJSON(&body); err == nil {
	// 		log.Printf("Callback received: %+v\n", body)
	// 	}
	// 	c.JSON(http.StatusOK, gin.H{"status": "callback received"})
	// })

	// r.Run(":8080")
}

// )
/*
LipadPayB2CConsumerKey    = "OsUs5kpfSKdPOToCdY77ZB9dAE3HmX2zhwEfEVr1G66xOU1m"
	LipadPayB2CConsumerSecret = "PGNzG7TIZYP1LU8mNdSH5kvJx7GVD1ilfqW9xd9UnsBV5pIreONI0VebzqdsX5dg"
	LipadPayB2CInitiatorName  = "Collins"
	LipadPayB2CPassword       = "g/cA19za1LNCJXYCZbRLnfzq87iOfdL+tF3g17GTYc8nleCqm0vnchLdHnjr6mBgjIOZtaueJiOaM7xd0z6CnkESso437gtexhUMoShwVKKk9Kl+h4GsHiyQrKXrRDPVsyPLKtnvBZeKZXkgO2JmRU7pa1qRdjXXUYqVOJ2TO19mJpMh6FbqgGRTYCXwiwyaWyA+hxbMtjEu1nKFlE9Kk8TnEfBxCNpwOLVsTdkrK6siLbcUJ1NqE7MNEddIHKTF7z+4nOtEAPGTwXVneWvPeUnWL0ooqxdLis+81mLTxtxV2MisHjHHDgTWw/EuEGjG1PwtTE4grcPLTdKhj13M2Q=="
	LipadPayB2CShortCode      = "4564641"
*/

const (
	conumerKey         = "OsUs5kpfSKdPOToCdY77ZB9dAE3HmX2zhwEfEVr1G66xOU1m"
	conumerSecret      = "PGNzG7TIZYP1LU8mNdSH5kvJx7GVD1ilfqW9xd9UnsBV5pIreONI0VebzqdsX5dg"
	initiatorName      = "Collins"
	securityCredential = "VNTumMLkIpRAhV/ieQS1JxKVqWcCY1G3TcbnLRBCOkQqJJQuXyWH8sx2IM5/m8RojRr9A8w0tF/EJW0p5TXJ957IyPiLNFqtyA1SGQb7ikgosCpGZxtuO/+qNHt7a6uwV/d35/lAsw+cmQnSzNb36aDc23yZqgLis8qGU5QdluGO4QZT1QOsnlYCwnWSbsUhxjYdTxwahktLyyr3ShEckxAp5FTO5YG3s+HFctCjo44L0YmvomChlQSSw7/BD5/eb3rDQAUgQmnpYzpurBbNVrch9WaI0x+5d3eU83G7Ud8EhqYoDzdEWJoMkD54/w7oBvY0stIg9bTrhZQrzA4CEA=="
	// LipadC2BSecurityCredential = "VNTumMLkIpRAhV/ieQS1JxKVqWcCY1G3TcbnLRBCOkQqJJQuXyWH8sx2IM5/m8RojRr9A8w0tF/EJW0p5TXJ957IyPiLNFqtyA1SGQb7ikgosCpGZxtuO/+qNHt7a6uwV/d35/lAsw+cmQnSzNb36aDc23yZqgLis8qGU5QdluGO4QZT1QOsnlYCwnWSbsUhxjYdTxwahktLyyr3ShEckxAp5FTO5YG3s+HFctCjo44L0YmvomChlQSSw7/BD5/eb3rDQAUgQmnpYzpurBbNVrch9WaI0x+5d3eU83G7Ud8EhqYoDzdEWJoMkD54/w7oBvY0stIg9bTrhZQrzA4CEA=="
	shortCode  = "4564641"
	c2bPassKey = "f79caa1b22f802af4f0489e03e2959d3d3463dc593e8bfe630adcac2b79d5c90"

	callbackURL    = "https://webhook.site/94ea9cd2-5230-4473-ab71-67c69a2a593d"
	b2pURL         = "https://api.safaricom.co.ke/mpesa/b2pochi/v1/paymentrequest"
	b2cURL         = "https://api.safaricom.co.ke/mpesa/b2c/v1/paymentrequest"
	b2bURL         = "https://api.safaricom.co.ke/mpesa/b2b/v1/paymentrequest"
	tokenURL       = "https://api.safaricom.co.ke/oauth/v1/generate?grant_type=client_credentials"
	balanceURL     = "https://api.safaricom.co.ke/mpesa/accountbalance/v1/query"
	transactionURL = "https://api.safaricom.co.ke/mpesa/transactionstatus/v1/query"
)

const (
	b2pConsumerKey        = conumerKey
	b2pConsumerSecret     = conumerSecret
	b2pInitiatorName      = initiatorName
	b2pShortCode          = shortCode
	b2pSecurityCredential = securityCredential
	b2pTokenURL           = tokenURL
)
