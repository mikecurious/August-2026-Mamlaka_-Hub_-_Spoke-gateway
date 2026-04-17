package main

// import (
// 	"bytes"
// 	"encoding/base64"
// 	"encoding/json"
// 	"fmt"
// 	"io"
// 	"log"
// 	"net/http"
// )

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

// type B2CRequest struct {
// 	InitiatorName      string `json:"InitiatorName"`
// 	SecurityCredential string `json:"SecurityCredential"`
// 	CommandID          string `json:"CommandID"`
// 	Amount             string `json:"Amount"`
// 	PartyA             string `json:"PartyA"`
// 	PartyB             string `json:"PartyB"`
// 	Remarks            string `json:"Remarks"`
// 	QueueTimeOutURL    string `json:"QueueTimeOutURL"`
// 	ResultURL          string `json:"ResultURL"`
// 	Occasion           string `json:"Occasion"`
// }

// type AccessTokenResponse struct {
// 	AccessToken string `json:"access_token"`
// 	ExpiresIn   string `json:"expires_in"`
// }

// type PaymentInput struct {
// 	Phone  string `json:"phone" binding:"required"`
// 	Amount string `json:"amount" binding:"required"`
// }

// type TransactionStatusInput struct {
// 	TransactionID string `json:"transaction_id" binding:"required"`
// }

// func getAccessToken() (string, error) {
// 	client := &http.Client{}
// 	req, _ := http.NewRequest("GET", tokenURL, nil)
// 	auth := base64.StdEncoding.EncodeToString([]byte("o9q33r3od3Y0oyNIlJdehJc1kaAL5X0kLWm9vMPLZRGOSgB" + ":" + "Zz5oqLm6QyFrMUM3VWQHtAwGpS6gbjnzNgLhAttLyA1K9bvSqQFsttFM1aJTUZP"))
// 	req.Header.Add("Authorization", "Basic "+auth)

// 	resp, err := client.Do(req)
// 	if err != nil {
// 		return "", err
// 	}
// 	defer resp.Body.Close()

// 	body, _ := io.ReadAll(resp.Body)
// 	var tokenResp AccessTokenResponse
// 	json.Unmarshal(body, &tokenResp)
// 	// print the token
// 	fmt.Println("Access Token:", tokenResp)
// 	return tokenResp.AccessToken, nil
// }

// func sendB2CPayment(phone string, amount string) (map[string]interface{}, error) {
// 	log.Println("Sending B2C payment to:", phone, "Amount:", amount)
// 	token, err := getAccessToken()
// 	if err != nil {
// 		return nil, fmt.Errorf("failed to get access token: %v", err)
// 	}

// 	payload := B2CRequest{
// 		InitiatorName:      initiatorName,
// 		SecurityCredential: securityCredential,
// 		CommandID:          "BusinessPayment",
// 		Amount:             amount,
// 		PartyA:             shortCode,
// 		PartyB:             phone,
// 		Remarks:            "B2C Payment",
// 		QueueTimeOutURL:    callbackURL,
// 		ResultURL:          callbackURL,
// 		Occasion:           "Withdrawal",
// 	}

// 	payloadBytes, _ := json.Marshal(payload)
// 	req, _ := http.NewRequest("POST", b2cURL, bytes.NewBuffer(payloadBytes))
// 	req.Header.Set("Authorization", "Bearer "+token)
// 	req.Header.Set("Content-Type", "application/json")

// 	client := &http.Client{}
// 	resp, err := client.Do(req)
// 	if err != nil {
// 		return nil, fmt.Errorf("B2C request error: %v", err)
// 	}
// 	defer resp.Body.Close()

// 	body, _ := io.ReadAll(resp.Body)

// 	var pretty map[string]interface{}
// 	if err := json.Unmarshal(body, &pretty); err == nil {
// 		return pretty, nil
// 	}
// 	return map[string]interface{}{"raw_response": string(body)}, nil
// }

// // 🔹 Check Paybill Balance
// func checkBalance() (map[string]interface{}, error) {
// 	token, err := getAccessToken()
// 	// fmt.Println("Access Token for balance check:")
// 	if err != nil {
// 		fmt.Printf("Error getting access token: %v\n", err)
// 		return nil, err
// 	}

// 	payload := map[string]interface{}{
// 		"Initiator":          initiatorName,
// 		"SecurityCredential": securityCredential,
// 		"CommandID":          "AccountBalance",
// 		"PartyA":             shortCode,
// 		"IdentifierType":     "4",
// 		"Remarks":            "Balance Check",
// 		"QueueTimeOutURL":    callbackURL,
// 		"ResultURL":          callbackURL,
// 	}

// 	payloadBytes, _ := json.Marshal(payload)
// 	req, _ := http.NewRequest("POST", balanceURL, bytes.NewBuffer(payloadBytes))
// 	req.Header.Set("Authorization", "Bearer "+token)
// 	req.Header.Set("Content-Type", "application/json")

// 	client := &http.Client{}
// 	resp, err := client.Do(req)
// 	if err != nil {
// 		return nil, err
// 	}
// 	defer resp.Body.Close()

// 	body, _ := io.ReadAll(resp.Body)
// 	var pretty map[string]interface{}
// 	json.Unmarshal(body, &pretty)
// 	return pretty, nil
// }

// // 🔹 Check Transaction Status
// func checkTransactionStatus(transactionID string) (map[string]interface{}, error) {
// 	token, err := getAccessToken()
// 	if err != nil {
// 		return nil, err
// 	}

// 	payload := map[string]interface{}{
// 		"Initiator":          initiatorName,
// 		"SecurityCredential": securityCredential,
// 		"CommandID":          "TransactionStatusQuery",
// 		"TransactionID":      transactionID,
// 		"PartyA":             shortCode,
// 		"IdentifierType":     "4",
// 		"ResultURL":          callbackURL,
// 		"QueueTimeOutURL":    callbackURL,
// 		"Remarks":            "Check transaction status",
// 	}

// 	payloadBytes, _ := json.Marshal(payload)
// 	req, _ := http.NewRequest("POST", transactionURL, bytes.NewBuffer(payloadBytes))
// 	req.Header.Set("Authorization", "Bearer "+token)
// 	req.Header.Set("Content-Type", "application/json")

// 	client := &http.Client{}
// 	resp, err := client.Do(req)
// 	if err != nil {
// 		return nil, err
// 	}
// 	defer resp.Body.Close()

// 	body, _ := io.ReadAll(resp.Body)
// 	var pretty map[string]interface{}
// 	json.Unmarshal(body, &pretty)
// 	return pretty, nil
// }

// func main() {
// 	// r := gin.Default()

// 	// B2C Payment
// 	// r.POST("/send-b2c", func(c *gin.Context) {
// 	// 	var input PaymentInput
// 	// 	if err := c.ShouldBindJSON(&input); err != nil {
// 	// 		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
// 	// 		return
// 	// 	}
// 	// 	resp, err := sendB2CPayment(input.Phone, input.Amount)
// 	// 	if err != nil {
// 	// 		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
// 	// 		return
// 	// 	}
// 	// 	c.JSON(http.StatusOK, resp)
// 	// })

// 	// Balance Check
// 	// r.GET("/check-balance", func(c *gin.Context) {
// 	resp, err := checkBalance()
// 	if err != nil {
// 		fmt.Printf("Balance check error: %v\n", err)
// 		return
// 	}
// 	// c.JSON(http.StatusOK, resp)
// 	fmt.Printf("Balance check response: %+v\n", resp)
// 	// })

// 	// Transaction Status
// 	// r.POST("/transaction-status", func(c *gin.Context) {
// 	// 	var input TransactionStatusInput
// 	// 	if err := c.ShouldBindJSON(&input); err != nil {
// 	// 		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
// 	// 		return
// 	// 	}
// 	// 	resp, err := checkTransactionStatus(input.TransactionID)
// 	// 	if err != nil {
// 	// 		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
// 	// 		return
// 	// 	}
// 	// 	c.JSON(http.StatusOK, resp)
// 	// })

// 	// Callback
// 	// r.POST("/callbackb2b", func(c *gin.Context) {
// 	// 	var body map[string]interface{}
// 	// 	if err := c.BindJSON(&body); err == nil {
// 	// 		log.Printf("Callback received: %+v\n", body)
// 	// 	}
// 	// 	c.JSON(http.StatusOK, gin.H{"status": "callback received"})
// 	// })

// 	// r.Run(":8080")
// }

// const (
// 	conumerKey         = "o9q33r3od3Y0oyNIlJdehJc1kaAL5X0kLWm9vMPLZRGOSgB"
// 	conumerSecret      = "Zz5oqLm6QyFrMUM3VWQHtAwGpS6gbjnzNgLhAttLyA1K9bvSqQFsttFM1aJTUZP"
// 	initiatorName      = "HAYAPI"
// 	securityCredential = "cht0RZwPIaDrMUNZUMO4Tr9DV845FulyOJ1TIuq59PbKSaxv8uSRCvqznmvcU+ToM2hMKBKNqCHUQQV6M/HUKpalNpnnvHyaWl7KMCaVZIx3Iwf5htpJz4CibuQS1bcWolcIhVVpAwqrR7C6qfm3fJUE/etmvtLCrIN22BdIu9PS/cE3BaNYipXvHO1QOKw9PnCX/5IouVF0FYPtGbNgUgGa0d43Je4wU71dBam55eBIZgn4ndruyMvl/YHQqcd9u/e2E2O5Iv6IEM9zk7u1X8X+adBY1JVClSfrey5eYS/iwZSYq/9nbgKZdA8+Hgg1fi3j7PUUGVpUN2VUeZznQ=="
// 	shortCode          = "4954120"
// 	callbackURL        = "https://webhook.site/5ec8b8d5-43b3-4de3-8895-34d34165546c"
// 	b2cURL             = "https://api.safaricom.co.ke/mpesa/b2c/v1/paymentrequest"
// 	tokenURL           = "https://api.safaricom.co.ke/oauth/v1/generate?grant_type=client_credentials"
// 	balanceURL         = "https://api.safaricom.co.ke/mpesa/accountbalance/v1/query"
// 	transactionURL     = "https://api.safaricom.co.ke/mpesa/transactionstatus/v1/query"
// )
