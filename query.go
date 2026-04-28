package main

// import (
// 	"bytes"
// 	"encoding/base64"
// 	"encoding/json"
// 	"io"
// 	"log"
// 	"net/http"

// 	"github.com/gin-gonic/gin"
// )

// type AccessTokenResponse struct {
// 	AccessToken string `json:"access_token"`
// 	ExpiresIn   string `json:"expires_in"`
// }

// type TransactionStatusInput struct {
// 	TransactionID string `json:"transaction_id" binding:"required"`
// }

// const (
// 	conumerKey         = "Lj4StGZWiCQRbgSQmGZV9FMEd0ZRzdkJ7h8yAhgiEdpKWFaO"
// 	conumerSecret      = "FYRQ59r2imH0Kw89SQH9qZKKJOupFTNWwEd0bt5zazqZUl8ie75bAwSSLuAtNxRh"
// 	initiatorName      = "collins"
// 	securityCredential = "P8tMtdc3GunIzpxqKoF4FC8SbPIGvV5lMtoqWMZvdQ5aKeRzWqQItVqAs8BDXSHoJS+kUZpjJ7xAH4q0W/L3Wv421RiI7VpTHk7TfcSFVPnwL1dYYUVPaSSuZPSMAgedkGIfL/kR8Uc9Q+pE4CMS7bCkDUdHfm61YiNgFQ+mabs0aFPynI4hYXEYGAXc4VIMQ+hevhKDs4cKvddrVGoGVYapi+vjqdzMe25gnyp6pZdHT+y6Lkm09WhIkmgRsVOSZnXixmPLEIgTjdw+xLHGFIqYn+QDE0Mk2jahNVXi82P2BGZ4ouORSHPrwE7nLOCKyiZUcfZKbok44uO+9BTjgg=="
// 	shortCode          = "3008826"
// 	transactionURL     = "https://api.safaricom.co.ke/mpesa/transactionstatus/v1/query"
// 	tokenURL           = "https://api.safaricom.co.ke/oauth/v1/generate?grant_type=client_credentials"
// )

// func getAccessToken() (string, error) {
// 	client := &http.Client{}
// 	req, err := http.NewRequest("GET", tokenURL, nil)
// 	if err != nil {
// 		return "", err
// 	}

// 	auth := base64.StdEncoding.EncodeToString([]byte(conumerKey + ":" + conumerSecret))
// 	req.Header.Add("Authorization", "Basic "+auth)

// 	resp, err := client.Do(req)
// 	if err != nil {
// 		return "", err
// 	}
// 	defer resp.Body.Close()

// 	body, err := io.ReadAll(resp.Body)
// 	if err != nil {
// 		return "", err
// 	}

// 	var tokenResp AccessTokenResponse
// 	if err := json.Unmarshal(body, &tokenResp); err != nil {
// 		return "", err
// 	}

// 	return tokenResp.AccessToken, nil
// }

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
// 		"ResultURL":          "https://webhook.site/b93adfee-a03c-4b01-bb30-681bd319fc99",
// 		"QueueTimeOutURL":    "https://webhook.site/b93adfee-a03c-4b01-bb30-681bd319fc99",
// 		"Remarks":            "Check transaction status",
// 	}

// 	payloadBytes, err := json.Marshal(payload)
// 	if err != nil {
// 		return nil, err
// 	}

// 	req, err := http.NewRequest("POST", transactionURL, bytes.NewBuffer(payloadBytes))
// 	if err != nil {
// 		return nil, err
// 	}
// 	req.Header.Set("Authorization", "Bearer "+token)
// 	req.Header.Set("Content-Type", "application/json")

// 	client := &http.Client{}
// 	resp, err := client.Do(req)
// 	if err != nil {
// 		return nil, err
// 	}
// 	defer resp.Body.Close()

// 	body, err := io.ReadAll(resp.Body)
// 	if err != nil {
// 		return nil, err
// 	}

// 	var pretty map[string]interface{}
// 	if err := json.Unmarshal(body, &pretty); err != nil {
// 		return map[string]interface{}{"raw_response": string(body)}, nil
// 	}
// 	return pretty, nil
// }

// func main() {
// 	r := gin.Default()

// 	r.POST("/transaction-status", func(c *gin.Context) {
// 		var input TransactionStatusInput
// 		if err := c.ShouldBindJSON(&input); err != nil {
// 			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
// 			return
// 		}

// 		resp, err := checkTransactionStatus(input.TransactionID)
// 		if err != nil {
// 			log.Println("MPESA transaction status error:", err)
// 			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
// 			return
// 		}

// 		c.JSON(http.StatusOK, resp)
// 	})

// 	if err := r.Run(":8081"); err != nil {
// 		log.Fatal(err)
// 	}
// }
