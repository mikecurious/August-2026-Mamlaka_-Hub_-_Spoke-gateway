package uganda

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
)

type ApiResponsee struct {
	Message string `json:"message"`
	Error   string `json:"error"`
}

// Function to send money to a phone
func SendMoneyToPhoneReal(phone string, amount float64) (bool, string, error) {
	fmt.Println("Sending money to phone:", phone, "Amount:", amount)
	// Prepare the payload
	payload := map[string]interface{}{
		"phone":    phone,
		"amount":   amount,
		"currency": "UGX",
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return false, "", errors.New("failed to marshal JSON")
	}

	apiURL := "https://uganda.commetagri.com/send"
	req, err := http.NewRequest("POST", apiURL, bytes.NewBuffer(data))
	if err != nil {
		return false, "", fmt.Errorf("failed to create request: %v", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return false, "", fmt.Errorf("failed to make request: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return false, "", fmt.Errorf("failed to read response: %v", err)
	}
	log.Println(body)

	var apiResponse ApiResponsee
	if err := json.Unmarshal(body, &apiResponse); err != nil {
		return false, "", fmt.Errorf("failed to unmarshal response: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		return false, apiResponse.Error, nil
	}

	return true, apiResponse.Message, nil
}

// Function to handle sending money to Uganda
// func sendToUgandaReal(c *gin.Context, db *DB, phoneNumber string, amount float64, currency string, userID uint) error {
// 	var user User

// 	// Fetch user details
// 	if err := db.First(&user, userID).Error; err != nil {
// 		c.JSON(http.StatusBadRequest, gin.H{"error": "user not found"})
// 		return errors.New("user not found")
// 	}

// 	// Validate the currency
// 	if currency != "UGX" {
// 		c.JSON(http.StatusBadRequest, gin.H{"error": "unsupported currency for Uganda"})
// 		return errors.New("unsupported currency for Uganda")
// 	}

// 	// Check user balance
// 	if user.UGXBalance < amount {
// 		c.JSON(http.StatusBadRequest, gin.H{"error": "insufficient UGX balance"})
// 		return errors.New("insufficient UGX balance")
// 	}

// 	// Deduct balance before sending money
// 	user.UGXBalance -= amount
// 	if err := db.Save(&user).Error; err != nil {
// 		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to deduct user balance"})
// 		return errors.New("failed to deduct user balance")
// 	}

// 	// Attempt to send money
// 	success, apiMessage, err := sendMoneyToPhoneReal(phoneNumber, amount)
// 	if !success || err != nil {
// 		// Rollback balance deduction if sending money fails
// 		user.UGXBalance += amount
// 		if rollbackErr := db.Save(&user).Error; rollbackErr != nil {
// 			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to rollback balance after error"})
// 			return fmt.Errorf("failed to rollback balance after error: %v, original error: %v", rollbackErr, err)
// 		}
// 		// Marshal the response and return
// 		response := gin.H{"error": apiMessage}
// 		c.JSON(http.StatusBadRequest, response)
// 		return err
// 	}

// 	// Log the transaction
// 	transaction := Transaction{
// 		UserID:     userID,
// 		Amount:     amount,
// 		Phone:      phoneNumber,
// 		Status:     "completed",
// 		Currency:   currency,
// 		ExternalID: uuid.New().String(),
// 		Action:     "Send Money",
// 	}

// 	if dbErr := db.Create(&transaction).Error; dbErr != nil {
// 		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to log transaction in database"})
// 		return fmt.Errorf("failed to log transaction in database: %v", dbErr)
// 	}

// 	// Return success response with marshaled JSON
// 	response := gin.H{
// 		"message": "Transaction successful",
// 		"status":  "completed",
// 	}
// 	c.JSON(http.StatusOK, response)

// 	return nil
// }

type ApiResponseCollectionReal struct {
	Message   string `json:"message"`
	Reference string `json:"reference"`
	Status    string `json:"status"`
}

func CollectFromPhoneReal(phone string, amount float64) (string, string, error) {
	// Prepare the payload
	payload := map[string]interface{}{
		"phone":    phone,
		"amount":   amount,
		"currency": "UGX",
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return "", "", errors.New("failed to marshal JSON")
	}

	apiURL := "https://uganda.commetagri.com/collect"
	req, err := http.NewRequest("POST", apiURL, bytes.NewBuffer(data))
	if err != nil {
		return "", "", fmt.Errorf("failed to create request: %v", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", "", fmt.Errorf("failed to make request: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", "", fmt.Errorf("failed to read response: %v", err)
	}

	var apiResponse ApiResponseCollectionReal
	if err := json.Unmarshal(body, &apiResponse); err != nil {
		return "", "", fmt.Errorf("failed to unmarshal response: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", apiResponse.Message, fmt.Errorf("transaction failed: %s", apiResponse.Message)
	}

	return apiResponse.Reference, apiResponse.Message, nil
}

// func collectFromUgandaReal(c *gin.Context, db *DB, phoneNumber string, amount float64, currency string, userID uint) error {
// 	var user User
// 	if err := db.First(&user, userID).Error; err != nil {
// 		c.JSON(http.StatusBadRequest, gin.H{"error": "user not found"})
// 		return errors.New("user not found")
// 	}

// 	if currency != "UGX" {
// 		c.JSON(http.StatusBadRequest, gin.H{"error": "unsupported currency for Uganda"})
// 		return errors.New("unsupported currency for Uganda")
// 	}

// 	reference, message, err := collectFromPhoneReal(phoneNumber, amount)
// 	transactionStatus := "failed"

// 	if err != nil {
// 		c.JSON(http.StatusInternalServerError, gin.H{"error": message})
// 		return err
// 	}

// 	transactionStatus = "completed"

// 	// Update user balance
// 	user.UGXBalance += amount
// 	if err := db.Save(&user).Error; err != nil {
// 		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update user balance"})
// 		return errors.New("failed to update user balance")
// 	}

// 	// Log the transaction
// 	transaction := Transaction{
// 		UserID:     userID,
// 		Amount:     amount,
// 		Status:     transactionStatus,
// 		Phone:      phoneNumber,
// 		Currency:   currency,
// 		ExternalID: uuid.New().String(),
// 		Action:     "Top Up",
// 	}

// 	if dbErr := db.Create(&transaction).Error; dbErr != nil {
// 		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to log transaction in database"})
// 		return errors.New("failed to log transaction in database")
// 	}

// 	// Return success message
// 	c.JSON(http.StatusOK, gin.H{
// 		"message":   message,
// 		"status":    transactionStatus,
// 		"reference": reference,
// 	})

// 	return nil
// }
