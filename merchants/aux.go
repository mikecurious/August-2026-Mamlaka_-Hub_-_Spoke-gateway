package merchants

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"com.mam-laka/transactions"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// buildMpesaMerchantCallback is the standard webhook payload for KES M-Pesa pay-in/payout.
func buildMpesaMerchantCallback(tx *transactions.TransactionModel, transactionStatus, transactionReport string, amount int, reference string) map[string]interface{} {
	payload := map[string]interface{}{
		"amount":            amount,
		"currency":          tx.Currency,
		"externalId":        tx.ExternalID,
		"secureId":          tx.SecureID,
		"transactionReport": transactionReport,
		"transactionStatus": transactionStatus,
	}
	if reference != "" {
		payload["reference"] = reference
	} else if tx.ProviderReference != "" {
		payload["reference"] = tx.ProviderReference
	}
	return payload
}

func metadataValueString(v interface{}) string {
	switch t := v.(type) {
	case string:
		return t
	case float64:
		return strconv.FormatInt(int64(t), 10)
	case int:
		return strconv.Itoa(t)
	case int64:
		return strconv.FormatInt(t, 10)
	default:
		return fmt.Sprintf("%v", v)
	}
}

func metadataValueInt(v interface{}, fallback int) int {
	switch t := v.(type) {
	case float64:
		return int(t)
	case int:
		return t
	case int64:
		return int(t)
	case string:
		if n, err := strconv.Atoi(t); err == nil {
			return n
		}
	}
	return fallback
}

func mpesaReceiptFromSTKMetadata(metadata map[string]interface{}) string {
	return metadataValueString(metadata["MpesaReceiptNumber"])
}

func mpesaReceiptFromB2CMetadata(metadata map[string]interface{}) string {
	return metadataValueString(metadata["TransactionReceipt"])
}

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
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Post(transaction.CallbackURL, "application/json", bytes.NewBuffer(responseBody))
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

func IsInList(value int, list []int) bool {
	for _, v := range list {
		if v == value {
			return true
		}
	}
	return false
}

func respondCallback(tx *transactions.TransactionModel, amount float64, status string, c *gin.Context) {
	response := gin.H{
		"transactionStatus": status,
		"transactionReport": status,
		"currency":          "XAF",
		"amount":            amount,
		"netAmount":         amount,
		"secureId":          tx.SecureID,
		"externalId":        tx.ExternalID,
	}

	log.Printf("Sending callback response: %+v", response)
}

func updateTransactionStatus(db *gorm.DB, transactionID uint, status, callbackStatus string) error {
	return db.Model(&transactions.TransactionModel{}).
		Where("id = ?", transactionID).
		Updates(map[string]interface{}{
			"transactionStatus": status,
			"callbackStatus":    callbackStatus,
		}).Error
}
