package pesalink

import (
	"bytes"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"
)

const (
	login               = "40411137"
	serviceId           = "1386"
	description         = "Mam-Laka Payment"
	currencyCode        = "KES"
	feeAmount           = "0"
	sourceAccountNumber = "55010160020282"
	sourceBankCode      = "40476000"
	pesalinkURL         = "https://pesalink.mam-laka.com/send_to_account"
)

type PaymentRequest struct {
	RequestID           string `json:"requestId"`
	Login               string `json:"login"`
	ServiceID           string `json:"serviceId"`
	Description         string `json:"description"`
	CurrencyCode        string `json:"currencyCode"`
	CustomerName        string `json:"customername"`
	Purpose             string `json:"purpose"`
	SourceAmount        string `json:"sourceAmount"`
	FeeAmount           string `json:"feeAmount"`
	SourceAccountNumber string `json:"sourceAccountNumber"`
	SourceBankCode      string `json:"sourceBankCode"`
	DestinationAccount  string `json:"destinationAccount"`
	DestinationBankCode string `json:"destinationBankCode"`
}

type PaymentResponse struct {
	Status        string `json:"Status"`
	StatusCode    string `json:"StatusCode"`
	StatusMessage string `json:"StatusMessage"`
}

// Struct for parsing Pesalink response
type PesalinkResponse struct {
	Body struct {
		ProcessPayment struct {
			Status        string `json:"Status"`
			StatusCode    string `json:"StatusCode"`
			StatusMessage string `json:"StatusMessage"`
			Payments      []struct {
				TransactionId string `json:"TransactionId"`
				Status        string `json:"Status"`
				StatusCode    string `json:"StatusCode"`
			} `json:"Payments"`
		} `json:"ProcessPayment"`
	} `json:"Body"`
}

func GenerateSecureID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return base64.URLEncoding.EncodeToString(b)
}

func SendPesalinkPayment(amount, destinationAccount, destinationBankCode, customerName, requestID string) (string, error) {
	// requestID := GenerateSecureID()

	paymentData := PaymentRequest{
		RequestID:           requestID,
		Login:               login,
		ServiceID:           serviceId,
		Description:         description,
		CurrencyCode:        currencyCode,
		CustomerName:        customerName,
		Purpose:             "MAM-LAKA PAYMENT",
		SourceAmount:        amount,
		FeeAmount:           feeAmount,
		SourceAccountNumber: sourceAccountNumber,
		SourceBankCode:      sourceBankCode,
		DestinationAccount:  destinationAccount,
		DestinationBankCode: destinationBankCode,
	}

	jsonData, err := json.Marshal(paymentData)
	if err != nil {
		return "", err
	}

	// Make HTTP request
	req, err := http.NewRequest("POST", pesalinkURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	// Read response body
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	// Parse response
	var pesalinkResponse PesalinkResponse
	err = json.Unmarshal(body, &pesalinkResponse)
	if err != nil {
		return "", err
	}
	fmt.Println("Pesalink Response:", pesalinkResponse)

	// Extract transaction details
	processPayment := pesalinkResponse.Body.ProcessPayment

	// Handle different response cases
	if len(processPayment.Payments) > 0 {
		payment := processPayment.Payments[0]

		if payment.Status == "SUCCESS" {
			return "Transaction Successful: " + payment.TransactionId, nil
		}

		if payment.Status == "ERROR" {
			if payment.StatusCode == "DUPLICATE_MESSAGE" {
				return "Transaction Failed: Duplicate transaction", errors.New("duplicate transaction")
			}
			return "Transaction Failed: " + processPayment.StatusMessage, errors.New(processPayment.StatusMessage)
		}
	}

	// Handle failed response without payments
	if processPayment.Status == "ERROR" {
		return "Transaction Failed: " + processPayment.StatusMessage, errors.New(processPayment.StatusMessage)
	}

	return "Transaction Failed: Unknown Error", errors.New("unknown error")
}
