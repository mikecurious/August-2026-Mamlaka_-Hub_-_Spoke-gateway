package main

// W0LXtRIf33TqQuSQLevyqTyvO847tqYMB3WauCFdYGRF6PiXZj770OhmG3lJnX4cVMrsm3K258oUx7y2p6MCs1V+W8YXNM3oqfb9pOoXFqnSmyTammvSetJct3/w0UT+0FUJTrg8JXH6j0FYlsqibCXc8f9ATb+twMi4Mxm37Ehu7fNOP40c6BHO7Cp4HHUa5yHjAVOSNDioWZr39bzyvBIiCT9Az/aISj060rCMZLWGlUlbpKVVQFqAwh/flu8AXxVqT2/zhg5NOPjhHb5i8uyau1IhN9LxC9nOGBKqInmlj59g4qTUPlSY1apZ3Y+Ny6iPN3yyseq3mm4UXKxGAg==

// package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
)

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
	InitiatorName      string `json:"InitiatorName"`
	SecurityCredential string `json:"SecurityCredential"`
	CommandID          string `json:"CommandID"`
	Amount             string `json:"Amount"`
	PartyA             string `json:"PartyA"`
	PartyB             string `json:"PartyB"`
	Remarks            string `json:"Remarks"`
	QueueTimeOutURL    string `json:"QueueTimeOutURL"`
	ResultURL          string `json:"ResultURL"`
	Occasion           string `json:"Occasion"`
}

type AccessTokenResponse struct {
	AccessToken string `json:"access_token"`
	ExpiresIn   string `json:"expires_in"`
}

type PaymentInput struct {
	Phone  string `json:"phone" binding:"required"`
	Amount string `json:"amount" binding:"required"`
}

type TransactionStatusInput struct {
	TransactionID string `json:"transaction_id" binding:"required"`
}

func getAccessToken() (string, error) {
	client := &http.Client{}
	req, _ := http.NewRequest("GET", tokenURL, nil)
	auth := base64.StdEncoding.EncodeToString([]byte(conumerKey + ":" + conumerSecret))
	req.Header.Add("Authorization", "Basic "+auth)

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var tokenResp AccessTokenResponse
	json.Unmarshal(body, &tokenResp)
	return tokenResp.AccessToken, nil
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
		QueueTimeOutURL:    "https://webhook.site/0f36025a-6733-4249-8ca1-e44c37fd8a28",
		ResultURL:          "https://webhook.site/0f36025a-6733-4249-8ca1-e44c37fd8a28",
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
		"QueueTimeOutURL":    "https://webhook.site/0f36025a-6733-4249-8ca1-e44c37fd8a28",
		"ResultURL":          "https://webhook.site/0f36025a-6733-4249-8ca1-e44c37fd8a28",
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
		"ResultURL":          "https://webhook.site/52bb1efc-6e06-41c9-9073-b2685de6f5ed",
		"QueueTimeOutURL":    "https://webhook.site/52bb1efc-6e06-41c9-9073-b2685de6f5ed",
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

func mai999() {
	// r := gin.Default()

	// B2C Payment
	// r.POST("/send-b2c", func(c *gin.Context) {
	// 	var input PaymentInput
	// 	if err := c.ShouldBindJSON(&input); err != nil {
	// 		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	// 		return
	// 	}
	// resp, err := sendB2CPayment("254768899729", "10")
	// if err != nil {
	// 	fmt.Println("Error sending B2C payment:", err)
	// }

	// fmt.Println("B2C Payment Response:", resp)

	resp2, err1 := checkBalance()

	if err1 != nil {
		fmt.Println("Error checking balance:", err1)
	}

	fmt.Println("Balance Check Response:", resp2)

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

/*


	conumerKey         = "m99dbV8i4Vm4GgIn9yQ903a5kOZoi9DXClmFkVq4Aepo3ihx"
	conumerSecret      = "x9CJvif8SpRg96qX0cUSyfyCrEkWjIjjwtoH5kGRIUE38orM714VImejkMDPs1EN"
	initiatorName      = "Collins"
	securityCredential = "Pka2rWhBsx3HdUpChiBUBotu47nXf6hoOZi7yNL+IO+hmewQ4v8segW/HjfRflylmIgRBfLD0NMJvtUASB7qDo7JRkHC/7jWAhUi3gJwaAV6X3yk5HNtwfpYm53wZcMqi6dOu1PH9Fj94Q0psg3DN6CiI3SZnxDNeWbeW5uIZPBQMTTOap04Wh0E4k9ygAgnCTXHOjMywQ3y5CgbfKwtvSnErOBzHtbGUvRoqOca66wkH5zdGA585OZtEjK2oJ/oYoxJuQ2K0iV4101Xa3RynS4XO2UOV9WalHXgKNi74E/vUCoSWHZJ80JpJ+myh6ficMuF3x3PB5xAqoN0lLyLmQ=="
	shortCode          = "3008818"
	callbackURL        = "https://webhook.site/7fe72d42-5ae6-493d-8187-e7013699257d"
	b2cURL             = "https://api.safaricom.co.ke/mpesa/b2c/v1/paymentrequest"
	tokenURL           = "https://api.safaricom.co.ke/oauth/v1/generate?grant_type=client_credentials"
	balanceURL         = "https://api.safaricom.co.ke/mpesa/accountbalance/v1/query"
	transactionURL     = "https://api.safaricom.co.ke/mpesa/transactionstatus/v1/query"


*/

// CrayC2BConsumerKey       = "v1PsGtti6d1GaS1JYV9Txo2S5dEI3Ea3V6SdCQ9B98HVuXW5"
// CrayC2BConsumerSecret    = "3JP0v5g0AJm2GEUAVDOD9sbxp1XPpQwNhAoAqZFBbBTt0JMXDHGIs46X7NkrVQAo"
// CrayC2BBusinessShortCode = "4041603"
// )

const (
	conumerKey    = "3RNMVF7lei58Sm3xGGJv4qkTgz3laFZ3zXi7BI7JjE5pasq5"
	conumerSecret = "pe5nTUfjgmMXnA8AQ1OX7vuILL7nPZOGqG9JFrTQPOYtDAuQrQBu9kmOcx0TdcLJ"
	initiatorName = "collins"

	securityCredential = "H3y/unl9dwsviXb6RFQ20Fzdp3DBGWuFuec4tbVgCUQGFZeLVuOILMZLmzYTLGqRCXbxPmlou/VdYrLBwANoFbK53ZSdlW9DsLzWtcRSkrDEoiQU9mDpp4e9T8pPC1Jbg3rISAdTrOP72OBnZPZu5rBkIgMnBPnVa21TJfy3K3xY+Gta+txH4cbguoJ1/ffmhJmMqX0Gcr90N6ozTOWxVsTh4WE904YWxagJrK4iTvHBIAwQ07lnto2dlSMNYAiYwEJF4l5KoNa7v2gtsUr7b3VbQe+4TzQ4KE1N4BHMKIe/tJ7ml2QNn3USyK5gpcKT9zYX75gazfkfg4G3fw9QeA=="
	shortCode          = "3008814"
	callbackURL        = "https://webhook.site/7fe72d42-5ae6-493d-8187-e7013699257d"
	b2cURL             = "https://api.safaricom.co.ke/mpesa/b2c/v1/paymentrequest"
	tokenURL           = "https://api.safaricom.co.ke/oauth/v1/generate?grant_type=client_credentials"
	balanceURL         = "https://api.safaricom.co.ke/mpesa/accountbalance/v1/query"
	transactionURL     = "https://api.safaricom.co.ke/mpesa/transactionstatus/v1/query"
)
