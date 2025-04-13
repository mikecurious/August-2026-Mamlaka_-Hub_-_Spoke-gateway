package merchants

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"com.mam-laka/transactions"
)

// SendCallback now accepts the callbackBody.Body type directly
func SendCallback(transactionID uint, callbackBody interface{}) error {
	// Step 1: Retrieve the transaction by ID
	log.Println("Getting transaction by id", transactionID)
	transaction, err := transactions.GetTransactionByID(transactionID)
	fmt.Println(transaction)
	if err != nil {
		return fmt.Errorf("failed to retrieve transaction: %v", err)
	}
	log.Println("Transaction found")

	// Step 2: Check if CallbackURL exists
	log.Println("Checking callback")

	// if transaction.CallbackURL == nil || transaction.CallbackURL == "" {
	// 	return fmt.Errorf("callback URL is missing or empty")
	// }
	log.Println("Callback found")

	// Step 3: Marshal the callbackBody to JSON
	responseBody, err := json.Marshal(callbackBody)
	if err != nil {
		return fmt.Errorf("failed to marshal callback body: %v", err)
	}

	// Step 4: Log the response body being sent
	log.Println("Sending the following JSON to callback URL:")
	log.Println(string(responseBody)) // This will print the raw JSON being sent

	// Step 5: Send the raw response body to the CallbackURL
	//send
	log.Println("send  raw response body to the CallbackURL", transaction.CallbackURL)
	resp, err := http.Post(transaction.CallbackURL, "application/json", bytes.NewBuffer(responseBody))
	if err != nil {
		return fmt.Errorf("failed to send callback: %v", err)
	}
	log.Println("Raw response sent to callback URL")
	defer resp.Body.Close()

	// Step 6: Check the response status
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("callback failed with status code: %d", resp.StatusCode)
	}

	return nil
}
