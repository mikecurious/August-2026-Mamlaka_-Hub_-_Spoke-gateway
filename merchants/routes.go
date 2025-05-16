package merchants

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
	"time"

	// "net/http"

	"strings"

	"com.mam-laka/auth"
	"com.mam-laka/balances"
	"com.mam-laka/database"
	"com.mam-laka/mpesa"
	"com.mam-laka/pesalink"
	"com.mam-laka/transactions"
	"com.mam-laka/users"
	"gorm.io/gorm"

	"github.com/gin-gonic/gin"
)

// remove prfix
func RemovePlusPrefix(phone string) string {
	if strings.HasPrefix(phone, "+") {
		return phone[1:] // Remove the first character
	}
	return phone
}

// MobilePaymentHandler to handle mobile payment initiation
func MobilePaymentHandler(c *gin.Context) {
	// Get the Authorization header
	authHeader := c.GetHeader("Authorization")
	if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization header is required"})
		return
	}

	// Extract the token from the Bearer scheme
	tokenString := strings.TrimPrefix(authHeader, "Bearer ")
	if tokenString == authHeader { // Token not prefixed with "Bearer "
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid authorization token format"})
		return
	}

	// Verify the token
	err := auth.VerifyToken(tokenString)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or expired token", "details": err.Error()})
		return
	}

	// Parse the mobile payment request
	var req MobilePaymentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input"})
		return
	}

	// Verify the merchant ID exists (or perform any business logic)
	userID, err := users.GetUserByMerchantId(req.ImpalaMerchantId)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Merchant not found"})
		return
	}

	// Get user by ID (you can use this data for logging or processing)
	user, err := users.GetUserByID(uint(userID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve user"})
		return
	}
	fmt.Printf("Payment initiated for user: %s with amount: %d\n", user.Name, req.Amount)

	// Here you would initiate the mobile payment logic, e.g., interacting with a payment API.
	// This is just an example response.
	//call the initiate payment method
	//StkPush(phoneNumber string, amount int, callbackURL, accountReference string

	// Generate secureId and other dynamic fields
	secureID := mpesa.GenerateSecureID()

	dateAdded := time.Now().Unix()

	// RemovePlusPrefix removes the '+' sign from the beginning of a phone number if present.

	// Replace with actual logic for initiating the M-Pesa request
	stkResponse, errror_stk := mpesa.StkPush(RemovePlusPrefix(req.PayerPhone), req.Amount, req.CallbackURL, user.Name)
	// StkPush(phoneNumber string, amount int, callbackURL, accountReference string) (*StkPushResponse, error) {

	if errror_stk != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to initiate payment", "details": errror_stk})
		return
	}
	// Create the transaction record in the database
	newTransaction := &transactions.TransactionModel{
		ImpalaMerchantID:    req.ImpalaMerchantId,
		MerchantRequestID:   stkResponse.MerchantRequestID,
		CheckoutRequestID:   stkResponse.CheckoutRequestID,
		ResponseDescription: stkResponse.ResponseDescription,
		ResponseCode:        stkResponse.ResponseCode,
		Currency:            req.Currency,
		Amount:              req.Amount,
		Msisdn:              req.PayerPhone,
		NetAmount:           float64(req.Amount), // Adjust if there are transaction fees
		SecureID:            secureID,
		SourceOfFunds:       req.MobileMoneySP,
		ExternalID:          req.ExternalID,
		CallbackURL:         req.CallbackURL,
		DateAdded:           dateAdded,
		TransactionReport:   "collection",
		TransactionStatus:   "PENDING", // Set an initial status
	}

	db := database.GetConnection()
	if err := db.Create(newTransaction).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create transaction", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":       "Payment initiation successful",
		"transactionId": &stkResponse.MerchantRequestID,
		"secureId":      secureID,
	})

}

// mobile withdrawal
func MobileWithdrawalHandler(c *gin.Context) {
	// Get the Authorization header
	authHeader := c.GetHeader("Authorization")
	if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization header is required"})
		return
	}

	// Extract the token from the Bearer scheme
	tokenString := strings.TrimPrefix(authHeader, "Bearer ")
	if tokenString == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid authorization token format"})
		return
	}

	// Verify the token
	err := auth.VerifyToken(tokenString)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or expired token", "details": err.Error()})
		return
	}

	// Parse the mobile payment request
	var req MobileWithdrawalRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input", "details": err.Error()})
		return
	}

	// Check if the merchant ID exists
	userID, err := users.GetUserByMerchantId(req.ImpalaMerchantId)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Merchant not found"})
		return
	}

	// Get user by ID
	user, err := users.GetUserByID(uint(userID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve user", "details": err.Error()})
		return
	}

	fmt.Printf("Payment initiated for user: %s with amount: %.2f\n", user.Name, req.Amount)

	// Generate secureId
	secureID := mpesa.GenerateSecureID()
	dateAdded := time.Now().Unix()

	// Check merchant's balance
	balance, err := balances.GetMerchantBalance(req.ImpalaMerchantId)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve merchant balance", "details": err.Error()})
		return
	}

	// Insufficient balance check
	if balance.KESBalance < float64(req.Amount) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Insufficient balance", "message": "Please top up your payout wallet"})
		return
	}

	// Deduct the balance
	err = balances.DeductKESBalance(req.ImpalaMerchantId, float64(req.Amount))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to deduct amount", "details": err.Error()})
		return
	}
	fmt.Printf("KES Balance for Merchant %s: %.2f\n", req.ImpalaMerchantId, balance.KESBalance)

	// Initiate payment via M-Pesa
	b2bResponse, err := mpesa.GenerateB2CRequest(RemovePlusPrefix(req.RecipientPhone), float64(req.Amount), req.CallbackURL, req.ExternalID, user.Name)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "Payment initiation failed", "details": err.Error()})
		return
	}

	// Create transaction record
	newTransaction := &transactions.TransactionModel{
		ImpalaMerchantID:    req.ImpalaMerchantId,
		MerchantRequestID:   b2bResponse.OriginatorConversationID,
		CheckoutRequestID:   b2bResponse.ConversationID,
		ResponseDescription: b2bResponse.ResponseDescription,
		ResponseCode:        b2bResponse.ResponseCode,
		Currency:            req.Currency,
		Amount:              int(req.Amount),
		Msisdn:              req.RecipientPhone,
		NetAmount:           float64(req.Amount),
		SecureID:            secureID,
		SourceOfFunds:       req.MobileMoneySP,
		ExternalID:          req.ExternalID,
		CallbackURL:         req.CallbackURL,
		DateAdded:           dateAdded,
		TransactionReport:   "withdraw",
		TransactionStatus:   "PENDING",
	}

	db := database.GetConnection()
	if err := db.Create(newTransaction).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to record transaction", "details": err.Error()})
		return
	}

	// Success response
	c.JSON(http.StatusOK, gin.H{
		"message":       "Payment initiation successful",
		"transactionId": req.ExternalID,
		"secureId":      secureID,
	})
}

// card paymnet hander
func CardPaymentHandler(c *gin.Context) {
	// Get the Authorization header
	authHeader := c.GetHeader("Authorization")
	if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization header is required"})
		return
	}

	// Extract the token from the Bearer scheme
	tokenString := strings.TrimPrefix(authHeader, "Bearer ")
	if tokenString == authHeader { // Token not prefixed with "Bearer "
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid authorization token format"})
		return
	}

	// Verify the token
	err := auth.VerifyToken(tokenString)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or expired token", "details": err.Error()})
		return
	}

	// Parse the mobile payment request
	var req CardPaymentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input"})
		return
	}

	// Verify the merchant ID exists (or perform any business logic)
	fmt.Println(req.ImpalaMerchantId)
	userID, err := users.GetUserByMerchantId(req.ImpalaMerchantId)

	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Merchant not found"})
		return
	}
	fmt.Println(userID)

	// Get user by ID (you can use this data for logging or processing)
	user, err := users.GetUserByID(uint(userID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve user"})
		return
	}
	fmt.Printf("Payment initiated for user: %s with amount: %.2f\n", user.Name, req.Amount)

	// Here you would initiate the mobile payment logic, e.g., interacting with a payment API.
	// This is just an example response.
	//call the initiate payment method
	//StkPush(phoneNumber string, amount int, callbackURL, accountReference string

	// Generate secureId and other dynamic fields
	secureID := mpesa.GenerateSecureID()

	dateAdded := time.Now().Unix()

	// Replace with actual logic for initiating the M-Pesa request
	// stkResponse, errror_stk := mpesa.StkPush(req.PayerPhone, req.Amount, req.CallbackURL, req.DisplayName)
	// cardLinkResponse, card_errror := card.GenerateCardPaymentLink(req.Currency, float64(req.Amount), req.ExternalID, req.CallbackURL, secureID)
	// // StkPush(phoneNumber string, amount int, callbackURL, accountReference string) (*StkPushResponse, error) {

	// if card_errror != nil {
	// 	c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to initiate payment", "details": "test"})
	// 	return
	// }
	cardResponse := "Card payment"
	cardResponseCode := "200"
	// Create the transaction record in the database
	newTransaction := &transactions.TransactionModel{
		ImpalaMerchantID:    req.ImpalaMerchantId,
		MerchantRequestID:   secureID,
		CheckoutRequestID:   secureID,
		ResponseDescription: cardResponse,
		ResponseCode:        cardResponseCode,
		Currency:            req.Currency,
		Amount:              int(req.Amount),
		Msisdn:              "Null",
		NetAmount:           float64(req.Amount), // Adjust if there are transaction fees
		SecureID:            secureID,
		SourceOfFunds:       req.MobileMoneySP,
		ExternalID:          req.ExternalID,
		CallbackURL:         req.CallbackURL,
		DateAdded:           dateAdded,
		TransactionReport:   "collection",
		TransactionStatus:   "PENDING", // Set an initial status
	}

	db := database.GetConnection()
	if err := db.Create(newTransaction).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create transaction", "details": err.Error()})
		return
	}
	data := fmt.Sprintf("amount=%.2f&merchant=%s&callback=%s&redirect=%s&externalid=%s&redirectUrl=%s", req.Amount, req.ImpalaMerchantId, req.CallbackURL, secureID, req.ExternalID, req.RedirectURL)
	fmt.Println(data)

	// Encode the string in Base64
	encoded := base64.StdEncoding.EncodeToString([]byte(data))

	// Print the Base64 encoded string
	fmt.Println("Base64 Encoded Data:", encoded)
	fmt.Println("merchant id ", req.ImpalaMerchantId)
	var cardlink string

	if req.ImpalaMerchantId == "Tallytours" { //kcb mid
		cardlink = "https://v1.mam-laka.com/mpgs.php?data=" + encoded

	} else { //uba mid
		cardlink = "https://collect.commetagri.com/uba.php?data=" + encoded
	}

	c.JSON(http.StatusOK, gin.H{
		"message":  "card Payment  initiation successful",
		"cardLink": cardlink,
		"secureId": secureID,
	})

}

// card paymnet crypto Payment Handler
func UsdcPaymentHandler(c *gin.Context) {
	// Get the Authorization header
	authHeader := c.GetHeader("Authorization")
	if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization header is required"})
		return
	}

	// Extract the token from the Bearer scheme
	tokenString := strings.TrimPrefix(authHeader, "Bearer ")
	if tokenString == authHeader { // Token not prefixed with "Bearer "
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid authorization token format"})
		return
	}

	// Verify the token
	err := auth.VerifyToken(tokenString)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or expired token", "details": err.Error()})
		return
	}

	// Parse the mobile payment request
	var req CardPaymentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input"})
		return
	}

	// Verify the merchant ID exists (or perform any business logic)
	fmt.Println(req.ImpalaMerchantId)
	userID, err := users.GetUserByMerchantId(req.ImpalaMerchantId)

	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Merchant not found"})
		return
	}
	fmt.Println(userID)

	// Get user by ID (you can use this data for logging or processing)
	user, err := users.GetUserByID(uint(userID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve user"})
		return
	}
	fmt.Printf("Payment initiated for user: %s with amount: %.2f\n", user.Name, req.Amount)

	// Here you would initiate the mobile payment logic, e.g., interacting with a payment API.
	// This is just an example response.
	//call the initiate payment method
	//StkPush(phoneNumber string, amount int, callbackURL, accountReference string

	// Generate secureId and other dynamic fields
	secureIDBytes := make([]byte, 5) // 5 bytes = 10 hex characters
	if _, err := rand.Read(secureIDBytes); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to generate secure ID",
			"details": err.Error(),
		})
		return
	}
	secureID := hex.EncodeToString(secureIDBytes) // e.g., "a7f9c23b1d"

	dateAdded := time.Now().Unix()

	// Replace with actual logic for initiating the M-Pesa request
	// stkResponse, errror_stk := mpesa.StkPush(req.PayerPhone, req.Amount, req.CallbackURL, req.DisplayName)
	// cardLinkResponse, card_errror := card.GenerateCardPaymentLink(req.Currency, float64(req.Amount), req.ExternalID, req.CallbackURL, secureID)
	// // StkPush(phoneNumber string, amount int, callbackURL, accountReference string) (*StkPushResponse, error) {

	// if card_errror != nil {
	// 	c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to initiate payment", "details": "test"})
	// 	return
	// }
	cardResponse := "Card payment"
	cardResponseCode := "200"
	fmt.Println("callback", req.CallbackURL)
	// Create the transaction record in the database
	newTransaction := &transactions.TransactionModel{
		ImpalaMerchantID:    req.ImpalaMerchantId,
		MerchantRequestID:   secureID,
		CheckoutRequestID:   secureID,
		ResponseDescription: cardResponse,
		ResponseCode:        cardResponseCode,
		Currency:            req.Currency,
		Amount:              int(req.Amount),
		Msisdn:              "Null",
		NetAmount:           float64(req.Amount), // Adjust if there are transaction fees
		SecureID:            secureID,
		SourceOfFunds:       "CRYPTO",
		ExternalID:          req.ExternalID,
		CallbackURL:         req.CallbackURL,
		DateAdded:           dateAdded,
		TransactionReport:   "collection",
		TransactionStatus:   "PENDING", // Set an initial status
	}

	db := database.GetConnection()
	if err := db.Create(newTransaction).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create transaction", "details": err.Error()})
		return
	}
	memo := fmt.Sprintf("MHS-%s-%s", req.ImpalaMerchantId, secureID)
	// fmt.Println(data)

	// Encode the string in Base64
	// encoded := base64.StdEncoding.EncodeToString([]byte(data))

	// Print the Base64 encoded string
	// fmt.Println("Base64 Encoded Data:", encoded)
	// cardlink := "https://commetagri.mam-laka.com/uba.php?data=" + encoded

	c.JSON(http.StatusOK, gin.H{
		"status":   "success",
		"Memo":     memo,
		"address":  "GACPWTGCILSIZNCQABZLU5LIXUVQMIYMIY7JY3CIBMRHXSQQM5UTXEBX",
		"secureId": secureID,
	})
}

// usdt payment handler
func UsdtPaymentHandler(c *gin.Context) {
	// Get the Authorization header
	authHeader := c.GetHeader("Authorization")
	if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization header is required"})
		return
	}

	// Extract the token from the Bearer scheme
	tokenString := strings.TrimPrefix(authHeader, "Bearer ")
	if tokenString == authHeader { // Token not prefixed with "Bearer "
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid authorization token format"})
		return
	}

	// Verify the token
	err := auth.VerifyToken(tokenString)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or expired token", "details": err.Error()})
		return
	}

	// Parse the mobile payment request
	var req CardPaymentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input"})
		return
	}

	// Verify the merchant ID exists (or perform any business logic)
	fmt.Println(req.ImpalaMerchantId)
	userID, err := users.GetUserByMerchantId(req.ImpalaMerchantId)

	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Merchant not found"})
		return
	}
	fmt.Println(userID)

	// Get user by ID (you can use this data for logging or processing)
	user, err := users.GetUserByID(uint(userID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve user"})
		return
	}
	fmt.Printf("Payment initiated for user: %s with amount: %.2f\n", user.Name, req.Amount)

	// Here you would initiate the mobile payment logic, e.g., interacting with a payment API.
	// This is just an example response.
	//call the initiate payment method
	//StkPush(phoneNumber string, amount int, callbackURL, accountReference string

	// Generate secureId and other dynamic fields
	secureIDBytes := make([]byte, 5) // 5 bytes = 10 hex characters
	if _, err := rand.Read(secureIDBytes); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to generate secure ID",
			"details": err.Error(),
		})
		return
	}
	secureID := hex.EncodeToString(secureIDBytes) // e.g., "a7f9c23b1d"

	dateAdded := time.Now().Unix()

	// Replace with actual logic for initiating the M-Pesa request
	// stkResponse, errror_stk := mpesa.StkPush(req.PayerPhone, req.Amount, req.CallbackURL, req.DisplayName)
	// cardLinkResponse, card_errror := card.GenerateCardPaymentLink(req.Currency, float64(req.Amount), req.ExternalID, req.CallbackURL, secureID)
	// // StkPush(phoneNumber string, amount int, callbackURL, accountReference string) (*StkPushResponse, error) {

	// if card_errror != nil {
	// 	c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to initiate payment", "details": "test"})
	// 	return
	// }
	cardResponse := "Card payment"
	cardResponseCode := "200"
	fmt.Println("callback", req.CallbackURL)
	// Create the transaction record in the database
	newTransaction := &transactions.TransactionModel{
		ImpalaMerchantID:    req.ImpalaMerchantId,
		MerchantRequestID:   secureID,
		CheckoutRequestID:   secureID,
		ResponseDescription: cardResponse,
		ResponseCode:        cardResponseCode,
		Currency:            "USDT",
		Amount:              int(req.Amount),
		Msisdn:              "Null",
		NetAmount:           float64(req.Amount), // Adjust if there are transaction fees
		SecureID:            secureID,
		SourceOfFunds:       "CRYPTO",
		ExternalID:          req.ExternalID,
		CallbackURL:         req.CallbackURL,
		DateAdded:           dateAdded,
		TransactionReport:   "collection",
		TransactionStatus:   "PENDING", // Set an initial status
	}

	db := database.GetConnection()
	if err := db.Create(newTransaction).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create transaction", "details": err.Error()})
		return
	}
	memo := fmt.Sprintf("MHS-%s-%s", req.ImpalaMerchantId, secureID)
	// fmt.Println(data)

	// Encode the string in Base64
	// encoded := base64.StdEncoding.EncodeToString([]byte(data))

	// Print the Base64 encoded string
	// fmt.Println("Base64 Encoded Data:", encoded)
	// cardlink := "https://commetagri.mam-laka.com/uba.php?data=" + encoded

	c.JSON(http.StatusOK, gin.H{
		"status":   "success",
		"Memo":     memo,
		"address":  "GACPWTGCILSIZNCQABZLU5LIXUVQMIYMIY7JY3CIBMRHXSQQM5UTXEBX",
		"secureId": secureID,
	})
}

func LoginHandler(c *gin.Context) {
	// Get the Authorization header
	authHeader := c.GetHeader("Authorization")
	if authHeader == "" || !strings.HasPrefix(authHeader, "Basic ") {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization header is required"})
		return
	}

	// Decode Basic Auth credentials
	encodedCredentials := strings.TrimPrefix(authHeader, "Basic ")
	decodedBytes, err := base64.StdEncoding.DecodeString(encodedCredentials)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid Authorization header"})
		return
	}
	credentials := string(decodedBytes)
	parts := strings.SplitN(credentials, ":", 2)
	if len(parts) != 2 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid Authorization header"})
		return
	}

	username := parts[0]
	password := parts[1]
	merchantID := username
	// Make request to function to get user by username
	user_id, err := users.GetUserByMerchantId(username)
	fmt.Println("user id", user_id)
	//get user details using the id

	// Authenticate the user
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "merchant not found 2"})
		return
	}
	// Get user by user id
	user, err := users.GetUserByID(uint(user_id))
	fmt.Println(user)

	// Check password based on the user id
	if err := user.CheckPassword(password); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid password"})
		return
	}

	// Generate a JWT token and get expiration date
	token, expirationDate, err := auth.CreateToken(user.Name, merchantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
		return
	}

	// Respond with the token and expiration date
	c.JSON(http.StatusOK, gin.H{
		"token":      token,
		"expires_at": expirationDate,
	})
}

// MobileCallbackHandler processes M-Pesa callback responses
// MobileCallbackHandler processes M-Pesa callback responses
// MobileCallbackHandler processes M-Pesa callback responses
func MobileCallbackHandler(c *gin.Context) {

	log.Println("inside a callback level 1")

	// Read the raw request body
	rawBody, err := c.GetRawData()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to read request body", "details": err.Error()})
		return
	}

	// Log the raw request body
	log.Println("Raw request body:", string(rawBody))

	// Parse the incoming JSON request into a map
	var response map[string]interface{}
	if err := json.Unmarshal(rawBody, &response); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid callback body", "details": err.Error()})
		return
	}

	// Debug the parsed response
	log.Println("Parsed response:", response)

	// Check if this is an STK callback
	body, ok := response["Body"].(map[string]interface{})
	// if !ok {
	// 	c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid callback body: missing Body"})
	// 	return
	// }

	stkCallback, ok := body["stkCallback"].(map[string]interface{})
	if ok { //stk push
		// c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid STK callback: missing stkCallback"})
		// return
		// Extract fields from STK callback
		merchantRequestID, _ := stkCallback["MerchantRequestID"].(string)
		checkoutRequestID, _ := stkCallback["CheckoutRequestID"].(string)
		resultCode, _ := stkCallback["ResultCode"].(float64)
		resultDesc, _ := stkCallback["ResultDesc"].(string)

		// Debug the extracted fields
		log.Println("MerchantRequestID:", merchantRequestID)
		log.Println("CheckoutRequestID:", checkoutRequestID)
		log.Println("ResultCode:", resultCode)
		log.Println("ResultDesc:", resultDesc)

		// Get database connection
		db := database.GetConnection()
		log.Println("iGetting database connection")

		// Retrieve the transaction by MerchantRequestID
		transaction, err := transactions.GetTransactionByMerchantRequestID(merchantRequestID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Transaction not found", "details": err.Error()})
			return
		}

		// Print the transaction details
		fmt.Printf("Transaction received: %+v\n", transaction)

		// Extract CallbackMetadata (if present)
		callbackMetadata, ok := stkCallback["CallbackMetadata"].(map[string]interface{})
		if ok {
			items, ok := callbackMetadata["Item"].([]interface{})
			if ok {
				for _, item := range items {
					itemMap, ok := item.(map[string]interface{})
					if ok {
						name, _ := itemMap["Name"].(string)
						value := itemMap["Value"]
						log.Printf("CallbackMetadata Item - Name: %s, Value: %v\n", name, value)
					}
				}
			}
		}

		// Process the ResultCode to determine transaction success or failure
		if resultCode == 0 { // Success
			// Extract metadata
			fmt.Println("inside success")
			metadata := make(map[string]interface{})
			if callbackMetadata != nil {
				items, ok := callbackMetadata["Item"].([]interface{})
				if ok {
					for _, item := range items {
						itemMap, ok := item.(map[string]interface{})
						if ok {
							name, _ := itemMap["Name"].(string)
							value := itemMap["Value"]
							metadata[name] = value
						}
					}
				}
			}

			// Update the transaction status to SUCCESS
			if err := db.Model(&transactions.TransactionModel{}).
				Where("id = ?", transaction.ID).
				Updates(map[string]interface{}{
					"transactionStatus": "SUCCESS",
					"callbackStatus":    "SENT",
				}).Error; err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update transaction", "details": err.Error()})
				return
			}
			//update them amounts
			// // 2. Retrieve the updated transaction to get impalaMerchantId and amount
			// var updatedTx transactions.TransactionModel
			// if err := db.First(&updatedTx, transaction.ID).Error; err != nil {
			// 	c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch updated transaction", "details": err.Error()})
			// 	return
			// }

			// 3. Update merchant collection balance
			fmt.Println("Updating collection balance ...")
			if err := db.Model(&balances.MerchantCollectionBalance{}).
				Where("impalaMerchantId = ?", transaction.ImpalaMerchantID).
				Update("kesBalance", gorm.Expr("kesBalance + ?", transaction.Amount)).Error; err != nil {
				fmt.Println("error updating the balance")
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update merchant balance", "details": err.Error()})
				return
			}
			fmt.Println("finished .. collection balance ...")

			// Process the callback response to match your required format
			callbackResponse := map[string]interface{}{
				"transactionStatus": "COMPLETE",
				"transactionReport": "COMPLETE",
				"currency":          "KES",              // Assuming KES is the default currency
				"amount":            metadata["Amount"], // Extract the correct amount from metadata
				"netAmount":         metadata["Amount"], // Assuming the net amount is same as amount
				"secureId":          transaction.SecureID,
				"externalId":        transaction.ExternalID, // Get from DB, not callback
			}
			log.Println("callback response", callbackResponse)

			// Call the SendCallback function to send the callback response to the merchant
			if err := SendCallback(transaction.ID, callbackResponse); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to send callback", "details": err.Error()})
				return
			}
			//call the crypto functon
			sendTokenTransfer(strconv.Itoa(transaction.Amount), "GBR7COBB5T5WPEYI7PN2XXLIYC2VUE22VIF4BHRKA3TPLUSZS6TQY7BE")

			c.JSON(http.StatusOK, gin.H{"message": "Callback processed and status updated to SENT"})
		} else { // Failure or Canceled
			// Update the transaction status to FAILED
			if err := db.Model(&transactions.TransactionModel{}).
				Where("id = ?", transaction.ID).
				Updates(map[string]interface{}{
					"transactionStatus":   "FAILED",
					"responseDescription": resultDesc, // Include the failure reason
					"callbackStatus":      "SENT",
				}).Error; err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update transaction", "details": err.Error()})
				return
			}

			// Extract metadata (if available)
			metadata := make(map[string]interface{})
			if callbackMetadata != nil {
				items, ok := callbackMetadata["Item"].([]interface{})
				if ok {
					for _, item := range items {
						itemMap, ok := item.(map[string]interface{})
						if ok {
							name, _ := itemMap["Name"].(string)
							value := itemMap["Value"]
							metadata[name] = value
						}
					}
				}
			}

			// Process the callback response to match your required format
			callbackResponse := map[string]interface{}{
				"transactionStatus": "FAILED",
				"transactionReport": "FAILED",
				"currency":          "KES", // Default to KES, adjust if necessary
				"amount":            transaction.Amount,
				"netAmount":         transaction.Amount,
				"secureId":          transaction.SecureID,
				"externalId":        transaction.ExternalID, // Get from DB, not callback
			}
			fmt.Println(callbackResponse)
			log.Println("call the callback", callbackResponse)

			// Call the SendCallback function to send the callback response to the merchant
			if err := SendCallback(transaction.ID, callbackResponse); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to send callback", "details": err.Error()})
				return
			}

			c.JSON(http.StatusOK, gin.H{"message": "Callback processed and status updated to FAILED"})

		}
	} else { //hanlde pull request
		fmt.Println("this is a pull request ")
		log.Println("this is a pull request")
	}
	// Check if this is a withdrawal response
	if result, ok := response["Result"].(map[string]interface{}); ok {
		log.Println("Processing withdrawal response")

		// Extract fields from the withdrawal response
		resultType, _ := result["ResultType"].(float64)
		resultCode, _ := result["ResultCode"].(float64)
		resultDesc, _ := result["ResultDesc"].(string)
		originatorConversationID, _ := result["OriginatorConversationID"].(string)
		conversationID, _ := result["ConversationID"].(string)
		transactionID, _ := result["TransactionID"].(string)

		// Debug the extracted fields
		log.Println("ResultType:", resultType)
		log.Println("ResultCode:", resultCode)
		log.Println("ResultDesc:", resultDesc)
		log.Println("OriginatorConversationID:", originatorConversationID)
		log.Println("ConversationID:", conversationID)
		log.Println("TransactionID:", transactionID)

		// Extract ResultParameters
		resultParameters, ok := result["ResultParameters"].(map[string]interface{})
		if ok {
			resultParameter, ok := resultParameters["ResultParameter"].([]interface{})
			if ok {
				for _, item := range resultParameter {
					itemMap, ok := item.(map[string]interface{})
					if ok {
						key, _ := itemMap["Key"].(string)
						value := itemMap["Value"]
						log.Printf("ResultParameter - Key: %s, Value: %v\n", key, value)
					}
				}
			}
		}

		// Fetch the transaction from the database using the TransactionID
		db := database.GetConnection()
		// var transaction transactions.TransactionModel
		// if err := db.Where("transaction_id = ?", transactionID).First(&transaction).Error; err != nil {
		// 	c.JSON(http.StatusNotFound, gin.H{"error": "Transaction not found", "details": err.Error()})
		// 	return
		// }
		transaction, err := transactions.GetTransactionByMerchantRequestID(originatorConversationID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Transaction not found", "details": err.Error()})
			return
		}

		// Debug the fetched transaction
		log.Println("Fetched transaction:", transaction)

		// Extract metadata from ResultParameters
		metadata := make(map[string]interface{})
		if resultParameters != nil {
			resultParameter, ok := resultParameters["ResultParameter"].([]interface{})
			if ok {
				for _, item := range resultParameter {
					itemMap, ok := item.(map[string]interface{})
					if ok {
						key, _ := itemMap["Key"].(string)
						value := itemMap["Value"]
						metadata[key] = value
					}
				}
			}
		}

		// Debug the extracted metadata
		log.Println("Withdrawal metadata:", metadata)

		// Process the withdrawal response
		if resultCode == 0 { // Success
			log.Println("Withdrawal successful")

			// Update the transaction status to SUCCESS
			if err := db.Model(&transactions.TransactionModel{}).
				Where("id = ?", transaction.ID).
				Updates(map[string]interface{}{
					"transactionStatus": "SUCCESS",
					"callbackStatus":    "SENT",
				}).Error; err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update transaction", "details": err.Error()})
				return
			}
			/*
			 {"amount":10,"currency":"KES",
			 "externalId":"ImpadlTdest25",
			 "netAmount":10,"secureId":"OgaEKPUnxToNfiTahW7uAw==",
			 "transactionReport":"COMPLETE",
			 "transactionStatus":"COMPLETE"}
			*/

			// Process the callback response to match your required format
			callbackResponse := map[string]interface{}{
				"transactionStatus":            "COMPLETE",
				"transactionReport":            "COMPLETE",
				"currency":                     "KES",                         // Assuming KES is the default currency
				"amount":                       metadata["TransactionAmount"], // Extract the transaction amount
				"netAmount":                    metadata["TransactionAmount"], // Assuming the net amount is same as amount
				"transactionReceipt":           metadata["TransactionReceipt"],
				"receiverPartyPublicName":      metadata["ReceiverPartyPublicName"],
				"transactionCompletedDateTime": metadata["TransactionCompletedDateTime"],
				"secureId":                     transaction.SecureID,   // Fetch from the database
				"externalId":                   transaction.ExternalID, // Fetch from the database
			}
			log.Println("callback response", callbackResponse)

			// Call the SendCallback function to send the callback response to the merchant
			if err := SendCallback(transaction.ID, callbackResponse); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to send callback", "details": err.Error()})
				return
			}
			//call the crypto functon
			DeductTokenTransfer(strconv.Itoa(transaction.Amount), "GBR7COBB5T5WPEYI7PN2XXLIYC2VUE22VIF4BHRKA3TPLUSZS6TQY7BE")

			c.JSON(http.StatusOK, gin.H{"message": "Withdrawal callback processed and status updated to SENT"})
		} else { // Failure
			log.Println("Withdrawal failed")

			// Update the transaction status to FAILED
			if err := db.Model(&transactions.TransactionModel{}).
				Where("id = ?", transaction.ID).
				Updates(map[string]interface{}{
					"transactionStatus":   "FAILED",
					"responseDescription": resultDesc, // Include the failure reason
					"callbackStatus":      "SENT",
				}).Error; err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update transaction", "details": err.Error()})
				return
			}

			// Process the callback response for a failed withdrawal
			callbackResponse := map[string]interface{}{
				"transactionStatus":            "FAILED",
				"transactionReport":            "FAILED",
				"currency":                     "KES",
				"amount":                       metadata["TransactionAmount"],
				"netAmount":                    metadata["TransactionAmount"],
				"transactionReceipt":           metadata["TransactionReceipt"],
				"receiverPartyPublicName":      metadata["ReceiverPartyPublicName"],
				"transactionCompletedDateTime": metadata["TransactionCompletedDateTime"],
				"errorMessage":                 resultDesc,             // Include the failure reason
				"secureId":                     transaction.SecureID,   // Fetch from the database
				"externalId":                   transaction.ExternalID, // Fetch from the database
			}
			log.Println("callback response", callbackResponse)

			// Call the SendCallback function to send the callback response to the merchant
			if err := SendCallback(transaction.ID, callbackResponse); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to send callback", "details": err.Error()})
				return
			}

			c.JSON(http.StatusOK, gin.H{"message": "Withdrawal callback processed and status updated to FAILED"})
		}
	} else {
		// Handle other callback types (STK, C2B, etc.)
		log.Println("Unknown callback type")
		c.JSON(http.StatusBadRequest, gin.H{"error": "Unknown callback type"})
		return
	}
	//call the token function here

}

func MobileCallbackHandler2(c *gin.Context) {
	var callbackBody struct {
		Body struct {
			StkCallback struct {
				MerchantRequestID string `json:"MerchantRequestID"`
				CheckoutRequestID string `json:"CheckoutRequestID"`
				ResultCode        int    `json:"ResultCode"`
				ResultDesc        string `json:"ResultDesc"`
				CallbackMetadata  struct {
					Items []struct {
						Name  string      `json:"Name"`
						Value interface{} `json:"Value,omitempty"` // Make Value optional
					} `json:"Item"`
				} `json:"CallbackMetadata"`
			} `json:"stkCallback"`
		} `json:"Body"`
	}

	// Debugging: Print received JSON
	bodyBytes, _ := ioutil.ReadAll(c.Request.Body)
	fmt.Println("Received JSON:", string(bodyBytes))

	// Parse JSON request
	if err := json.Unmarshal(bodyBytes, &callbackBody); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid callback body", "details": err.Error()})
		return
	}

	db := database.GetConnection()
	var transaction transactions.TransactionModel

	merchantRequestID := callbackBody.Body.StkCallback.MerchantRequestID
	resultCode := callbackBody.Body.StkCallback.ResultCode
	resultDesc := callbackBody.Body.StkCallback.ResultDesc

	if merchantRequestID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing MerchantRequestID"})
		return
	}

	// Check if transaction exists in the database
	if err := db.Where("merchantRequestID = ?", merchantRequestID).First(&transaction).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Transaction not found", "details": err.Error()})
		return
	}

	// Determine transaction status
	transactionStatus := "FAILED"
	if resultCode == 0 {
		transactionStatus = "COMPLETED"
	}

	// Prepare update data for database
	updateData := map[string]interface{}{
		"transactionStatus":   transactionStatus,
		"responseDescription": resultDesc,
		"callbackStatus":      "SENT",
	}

	if err := db.Model(&transactions.TransactionModel{}).
		Where("id = ?", transaction.ID).
		Updates(updateData).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update transaction", "details": err.Error()})
		return
	}

	// Extract additional metadata (Amount)
	var amount float64
	for _, item := range callbackBody.Body.StkCallback.CallbackMetadata.Items {
		if item.Name == "Amount" {
			if val, ok := item.Value.(float64); ok {
				amount = val
			}
		}
	}

	// Construct callback response (externalId comes from the database)
	callbackResponse := map[string]interface{}{
		"transactionStatus": transactionStatus,
		"transactionReport": resultDesc,
		"currency":          transaction.Currency,
		"amount":            amount,
		"netAmount":         transaction.NetAmount,
		"secureId":          transaction.SecureID,
		"externalId":        transaction.ExternalID, // Get from DB, not callback
	}

	// Debugging: Print outgoing response
	fmt.Println("Sending callback:", callbackResponse)

	// Send callback to external system
	if err := SendCallback(transaction.ID, callbackResponse); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to send callback", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, callbackResponse)
}

func CardCallbackHandler(c *gin.Context) {
	var callbackBody struct {
		TransactionStatus string `json:"transactionStatus"`
		TransactionReport string `json:"transactionReport"`
		Currency          string `json:"currency"`
		Amount            string `json:"amount"`
		NetAmount         string `json:"netAmount"`
		SecureID          string `json:"redirect"`
		ExternalID        string `json:"externalId"`
		RedirectURL       string `json:"redirectUrl"`
	}

	// Parse the incoming JSON request
	if err := c.ShouldBindJSON(&callbackBody); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid callback body", "details": err.Error()})
		return
	}
	//testing
	//Convert struct to JSON (map representation)
	jsonData, _ := json.Marshal(callbackBody)

	// Convert JSON to map[string]interface{}
	var callbackMap map[string]interface{}
	json.Unmarshal(jsonData, &callbackMap)

	// Print key-value pairs
	for key, value := range callbackMap {
		fmt.Printf("%s: %v\n", key, value)
	}

	// Extract the MerchantRequestID from the RedirectURL
	merchantRequestID := callbackBody.SecureID
	// fmt.Println("callback data: ", callbackBody)
	// fmt.Print("merchantRequestID: ", callbackBody.SecureID)

	db := database.GetConnection()

	// Retrieve the transaction by RedirectURL (MerchantRequestID)
	var transaction transactions.TransactionModel
	if err := db.Where("merchantRequestID = ?", merchantRequestID).First(&transaction).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Transaction not found", "details": err.Error()})
		return
	}

	// Determine transaction success or failure
	transactionStatus := "FAILED"
	callbackStatus := "SENT"
	if callbackBody.TransactionStatus == "COMPLETED" {
		transactionStatus = "SUCCESS"
	}

	// Update the transaction status in the database
	fmt.Println("Updating transaction status")
	if err := db.Model(&transactions.TransactionModel{}).
		Where("id = ?", transaction.ID).
		Updates(map[string]interface{}{
			"transactionStatus":   transactionStatus,
			"responseDescription": callbackBody.TransactionReport,
			"callbackStatus":      callbackStatus,
		}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update transaction", "details": err.Error()})
		return
	}

	// If the transaction was successful, update the merchant's balance
	if callbackBody.TransactionStatus == "COMPLETED" {
		log.Println("Transaction successful. Updating merchant balance for:", transaction.ImpalaMerchantID)

		// Retrieve the merchant's balance
		var balance balances.MerchantBalance
		if err := db.Where("impalaMerchantId = ?", transaction.ImpalaMerchantID).First(&balance).Error; err != nil {
			log.Println("Error retrieving merchant balance:", err)
			c.JSON(http.StatusNotFound, gin.H{"error": "Merchant balance not found", "details": err.Error()})
			return
		}
		log.Printf("Current balance retrieved: %+v\n", balance)

		// Convert the amount to a float
		amount, err := strconv.ParseFloat(callbackBody.NetAmount, 64)
		if err != nil {
			log.Println("Invalid amount format:", err)
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid amount format", "details": err.Error()})
			return
		}
		log.Println("Transaction amount:", amount, "Currency:", callbackBody.Currency)

		// Update the correct currency balance
		updateData := make(map[string]interface{})
		switch callbackBody.Currency {
		case "KES":
			updateData["kesBalance"] = balance.KESBalance + amount
		case "USD":
			updateData["usdBalance"] = balance.USDBalance + amount
		case "USDC":
			updateData["usdcBalance"] = balance.USDCBalance + amount
		case "IMPA":
			updateData["impaBalance"] = balance.ImpaBalance + amount
		case "LUMEN":
			updateData["lumenBalance"] = balance.LumenBalance + amount
		case "USDT":
			updateData["usdtBalance"] = balance.USDTBalance + amount
		case "EUR":
			updateData["eurBalance"] = balance.EURBalance + amount
		case "GBP":
			updateData["gbpBalance"] = balance.GBPBalance + amount
		case "TZS":
			updateData["tzsBalance"] = balance.TZSBalance + amount
		case "UGX":
			updateData["ugxBalance"] = balance.UGXBalance + amount
		default:
			log.Println("Unsupported currency:", callbackBody.Currency)
			c.JSON(http.StatusBadRequest, gin.H{"error": "Unsupported currency"})
			return
		}
		log.Printf("Updated balance data: %+v\n", updateData)

		// Update the merchant's balance in the database
		if err := db.Model(&balances.MerchantBalance{}).
			Where("impalaMerchantId = ?", transaction.ImpalaMerchantID).
			Updates(updateData).Error; err != nil {
			log.Println("Failed to update merchant balance:", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update merchant balance", "details": err.Error()})
			return
		}

		log.Println("Merchant balance successfully updated for:", transaction.ImpalaMerchantID)
	}

	// Call the SendCallback function to notify the merchant
	if err := SendCallback(transaction.ID, callbackBody); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to send callback", "details": err.Error()})
		return
	}

	// Respond with success
	c.JSON(http.StatusOK, gin.H{"message": "Callback processed and status updated to SENT"})
	// initiate the token transfre based  on status
	if callbackBody.TransactionStatus == "COMPLETED" {
		sendTokenTransfer(strconv.Itoa(transaction.Amount), "GBR7COBB5T5WPEYI7PN2XXLIYC2VUE22VIF4BHRKA3TPLUSZS6TQY7BE")

	}
}

func CryptoCallbackHandler(c *gin.Context) {
	var callbackBody struct {
		TransactionStatus string `json:"transactionStatus"`
		TransactionReport string `json:"transactionReport"`
		Currency          string `json:"currency"`
		Amount            string `json:"amount"`
		NetAmount         string `json:"netAmount"`
		SecureID          string `json:"secureId"`
		ExternalID        string `json:"externalId"`
		RedirectURL       string `json:"redirectUrl"`
	}

	// Parse the incoming JSON request
	if err := c.ShouldBindJSON(&callbackBody); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid callback body", "details": err.Error()})
		return
	}
	//testing
	//Convert struct to JSON (map representation)
	jsonData, _ := json.Marshal(callbackBody)

	// Convert JSON to map[string]interface{}
	var callbackMap map[string]interface{}
	json.Unmarshal(jsonData, &callbackMap)

	// Print key-value pairs
	for key, value := range callbackMap {
		fmt.Printf("%s: %v\n", key, value)
	}

	// Extract the MerchantRequestID from the RedirectURL
	merchantRequestID := callbackBody.SecureID
	// fmt.Println("callback data: ", callbackBody)
	fmt.Print("merchantRequestID: ", callbackBody.SecureID)

	db := database.GetConnection()

	// Retrieve the transaction by RedirectURL (MerchantRequestID)
	var transaction transactions.TransactionModel
	if err := db.Where("secureId = ?", merchantRequestID).First(&transaction).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Transaction not found", "details": err.Error()})
		return
	}

	// Determine transaction success or failure
	transactionStatus := "FAILED"
	callbackStatus := "SENT"
	if callbackBody.TransactionStatus == "COMPLETED" {
		transactionStatus = "SUCCESS"
	}

	// Update the transaction status in the database
	fmt.Println("Updating transaction status")
	if err := db.Model(&transactions.TransactionModel{}).
		Where("id = ?", transaction.ID).
		Updates(map[string]interface{}{
			"transactionStatus":   transactionStatus,
			"responseDescription": callbackBody.TransactionReport,
			"callbackStatus":      callbackStatus,
		}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update transaction", "details": err.Error()})
		return
	}

	// If the transaction was successful, update the merchant's balance
	if callbackBody.TransactionStatus == "COMPLETED" {
		log.Println("Transaction successful. Updating merchant balance for:", transaction.ImpalaMerchantID)

		// Retrieve the merchant's balance
		var balance balances.MerchantBalance
		if err := db.Where("impalaMerchantId = ?", transaction.ImpalaMerchantID).First(&balance).Error; err != nil {
			log.Println("Error retrieving merchant balance:", err)
			c.JSON(http.StatusNotFound, gin.H{"error": "Merchant balance not found", "details": err.Error()})
			return
		}
		log.Printf("Current balance retrieved: %+v\n", balance)

		// Convert the amount to a float
		amount, err := strconv.ParseFloat(callbackBody.NetAmount, 64)
		if err != nil {
			log.Println("Invalid amount format:", err)
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid amount format", "details": err.Error()})
			return
		}
		log.Println("Transaction amount:", amount, "Currency:", callbackBody.Currency)

		// Update the correct currency balance
		updateData := make(map[string]interface{})
		switch callbackBody.Currency {
		case "KES":
			updateData["kesBalance"] = balance.KESBalance + amount
		case "USD":
			updateData["usdBalance"] = balance.USDBalance + amount
		case "USDC":
			updateData["usdcBalance"] = balance.USDCBalance + amount
		case "IMPA":
			updateData["impaBalance"] = balance.ImpaBalance + amount
		case "LUMEN":
			updateData["lumenBalance"] = balance.LumenBalance + amount
		case "USDT":
			updateData["usdtBalance"] = balance.USDTBalance + amount
		case "EUR":
			updateData["eurBalance"] = balance.EURBalance + amount
		case "GBP":
			updateData["gbpBalance"] = balance.GBPBalance + amount
		case "TZS":
			updateData["tzsBalance"] = balance.TZSBalance + amount
		case "UGX":
			updateData["ugxBalance"] = balance.UGXBalance + amount
		default:
			log.Println("Unsupported currency:", callbackBody.Currency)
			c.JSON(http.StatusBadRequest, gin.H{"error": "Unsupported currency"})
			return
		}
		log.Printf("Updated balance data: %+v\n", updateData)

		// Update the merchant's balance in the database
		if err := db.Model(&balances.MerchantBalance{}).
			Where("impalaMerchantId = ?", transaction.ImpalaMerchantID).
			Updates(updateData).Error; err != nil {
			log.Println("Failed to update merchant balance:", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update merchant balance", "details": err.Error()})
			return
		}

		log.Println("Merchant balance successfully updated for:", transaction.ImpalaMerchantID)
	}

	// Call the SendCallback function to notify the merchant
	if err := SendCallback(transaction.ID, callbackBody); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to send callback", "details": err.Error()})
		return
	}

	// Respond with success
	c.JSON(http.StatusOK, gin.H{"message": "Callback processed and status updated to SENT"})
}

func TagsHandler(c *gin.Context) {
	var Tag struct {
		Tag      string  `json:"tag" binding:"required"`
		Amount   float32 `json:"amount" binding:"required"`
		Currency string  `json:"currency" binding:"required"`
	}

	if err := c.ShouldBindJSON(&Tag); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format", "details": err.Error()})
		return
	}

	// Generate secureId and other dynamic fields
	secureID := mpesa.GenerateSecureID()

	dateAdded := time.Now().Unix()

	// Replace with actual logic for initiating the M-Pesa request
	// stkResponse, errror_stk := mpesa.StkPush(req.PayerPhone, req.Amount, req.CallbackURL, req.DisplayName)
	// cardLinkResponse, card_errror := card.GenerateCardPaymentLink(Tag.Currency, float64(Tag.Amount), secureID, secureID, secureID)
	// // StkPush(phoneNumber string, amount int, callbackURL, accountReference string) (*StkPushResponse, error) {

	// if card_errror != nil {
	// 	c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to initiate payment", "details": "test"})
	// 	return
	// }
	cardResponse := "Card payment"
	cardResponseCode := "200"
	externalId := "200"
	callbackUrl := "200"

	// Create the transaction record in the database
	newTransaction := &transactions.TransactionModel{
		ImpalaMerchantID:    Tag.Tag,
		MerchantRequestID:   secureID,
		CheckoutRequestID:   secureID,
		ResponseDescription: cardResponse,
		ResponseCode:        cardResponseCode,
		Currency:            Tag.Currency,
		Amount:              int(Tag.Amount),
		Msisdn:              "Null",
		NetAmount:           float64(Tag.Amount), // Adjust if there are transaction fees
		SecureID:            secureID,
		SourceOfFunds:       "card",
		ExternalID:          externalId,
		CallbackURL:         callbackUrl,
		DateAdded:           dateAdded,
		TransactionReport:   "collection",
		TransactionStatus:   "PENDING", // Set an initial status
	}

	db := database.GetConnection()
	if err := db.Create(newTransaction).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create transaction", "details": err.Error()})
		return
	}
	data := fmt.Sprintf("amount=%.2f&merchant=%s&callback=%s&redirect=%s&externalid=%s", Tag.Amount, Tag.Tag, callbackUrl, secureID, externalId)
	// fmt.Println(data)

	// Encode the string in Base64
	encoded := base64.StdEncoding.EncodeToString([]byte(data))

	// Print the Base64 encoded string
	var cardlink string
	if Tag.Tag == "Tallytours" { //kcb mid
		cardlink = "https://v1.mam-laka.com/mpgs.php?data=" + encoded

	} else { //uba mid
		cardlink = "https://collect.commetagri.com/uba.php?data=" + encoded
	}

	c.JSON(http.StatusOK, gin.H{
		"message":  "card Payment  initiation successful",
		"cardLink": cardlink,
		"secureId": secureID,
	})
}

func GetTransactionHandler(c *gin.Context) {
	// Extract query parameters
	merchantID := c.Query("merchant")
	secureID := c.Query("secureId") // Ensure the key matches the actual query parameter

	// Retrieve transaction
	transaction, err := transactions.GetTransactionByMerchantIDAndSecureID(merchantID, secureID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve transaction", "details": err.Error()})
		return
	}

	// Serialize transaction
	serializer := transactions.NewTransactionSerializer(c, transaction)
	response := serializer.Response()

	// Send response
	c.JSON(http.StatusOK, gin.H{"transaction": response})
}

// handle using pasa pal ...
func BankTransferHandler(c *gin.Context) {
	fmt.Println(c)
	merchantID, merchantExists := c.Get("merchantID")
	fmt.Println("merchantID", merchantID)
	username, userExists := c.Get("username")
	fmt.Println("username", username)

	if !merchantExists || !userExists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"status":  "error",
			"message": "Missing authentication details",
		})
		return
	}

	// Get the Authorization header
	authHeader := c.GetHeader("Authorization")
	if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization header is required"})
		return
	}

	// Extract the token from the Bearer scheme
	tokenString := strings.TrimPrefix(authHeader, "Bearer ")
	if tokenString == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid authorization token format"})
		return
	}

	// Verify the token
	err := auth.VerifyToken(tokenString)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or expired token", "details": err.Error()})
		return
	}

	// Parse the mobile payment request
	var req PesalinkBankTransferRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input", "details": err.Error()})
		return
	}

	// Check if the merchant ID exists

	userID, err := users.GetUserByMerchantId(merchantID.(string))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Merchant not found"})
		return
	}

	// Get user by ID
	user, err := users.GetUserByID(uint(userID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve user", "details": err.Error()})
		return
	}

	fmt.Printf("Payment initiated for user: %s with amount: %.2f\n", user.Name, req.Amount)

	// Generate secureId
	// secureID := mpesa.GenerateSecureID()
	dateAdded := time.Now().Unix()

	// Check merchant's balance
	balance, err := balances.GetMerchantBalance(merchantID.(string))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve merchant balance", "details": err.Error()})
		return
	}

	// Insufficient balance check
	// floatAmount, err := strconv.ParseFloat(req.Amount, 64)
	// if err != nil {
	// 	fmt.Println("Error converting string to float:", err)
	// 	return
	// }

	if balance.KESBalance < float64(req.Amount) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Insufficient balance", "message": "Please top up your payout wallet"})
		return
	}

	// Deduct the balance
	err = balances.DeductKESBalance(merchantID.(string), float64(req.Amount))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to deduct amount", "details": err.Error()})
		return
	}
	fmt.Printf("KES Balance for Merchant %s: %.2f\n", merchantID, balance.KESBalance)

	// Initiate payment via M-Pesa
	// b2bResponse, err := mpesa.GenerateB2CRequest(RemovePlusPrefix(req.RecipientPhone), float64(req.Amount), req.CallbackURL, req.ExternalID, user.Name)
	requestID := pesalink.GenerateSecureID()
	strVal := fmt.Sprintf("%.2f", req.Amount)
	bankTransferResponse, err := pesalink.SendPesalinkPayment(strVal, req.DestinationAccount, req.DestinationBankCode, username.(string), requestID)
	var transactionStatus string

	if err != nil {
		transactionStatus = "FAILED"
		fmt.Println("error ", err)
		// c.JSON(http.StatusBadGateway, gin.H{"error": "Payment initiation failed", "details": err.Error()})
	} else {
		transactionStatus = "SUCCESS"

	}
	// Check if the transaction was successful
	// if  == "SUCCESS" && bankTransferResponse.StatusCode == "0" {

	// 	fmt.Println("Transaction Successful:", bankTransferResponse.StatusMessage)
	// } else {
	// 	transactionStatus = "FAILED"
	// 	fmt.Println("Transaction Failed:", bankTransferResponse.StatusMessage)
	// }
	fmt.Println("Transaction Status:", transactionStatus)
	fmt.Println("Transaction Response:", bankTransferResponse)

	// Create transaction record
	newTransaction := &transactions.TransactionModel{
		ImpalaMerchantID:    merchantID.(string),
		MerchantRequestID:   requestID,
		CheckoutRequestID:   requestID,
		ResponseDescription: bankTransferResponse,
		ResponseCode:        "200",
		Currency:            "KES",
		Amount:              int(req.Amount),
		Msisdn:              req.DestinationAccount,
		NetAmount:           float64(req.Amount),
		SecureID:            requestID,
		SourceOfFunds:       "MAM-LAKA",
		ExternalID:          requestID,
		CallbackURL:         "NULL",
		DateAdded:           dateAdded,
		TransactionReport:   "withdraw",
		TransactionStatus:   transactionStatus,
	}

	db := database.GetConnection()
	if err := db.Create(newTransaction).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to record transaction", "details": err.Error()})
		return
	}

	// Success response
	c.JSON(http.StatusOK, gin.H{
		"message":           bankTransferResponse,
		"transactionId":     requestID,
		"transactionStatus": transactionStatus,
	})
}

// get merchant balances
func GetMerchantBalanceHandler(c *gin.Context) {
	merchantID, merchantExists := c.Get("merchantID")
	fmt.Println("merchantID", merchantID)
	username, userExists := c.Get("username")
	fmt.Println("username", username)

	if !merchantExists || !userExists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"status":  "error",
			"message": "Missing authentication details",
		})
		return
	}

	// Get the Authorization header
	authHeader := c.GetHeader("Authorization")
	if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization header is required"})
		return
	}

	// Extract the token from the Bearer scheme
	tokenString := strings.TrimPrefix(authHeader, "Bearer ")
	if tokenString == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid authorization token format"})
		return
	}

	// Verify the token
	err := auth.VerifyToken(tokenString)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or expired token", "details": err.Error()})
		return
	}

	// Parse the mobile payment request
	// var req PesalinkBankTransferRequest
	// if err := c.ShouldBindJSON(&req); err != nil {
	// 	c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input", "details": err.Error()})
	// 	return
	// }

	// Check if the merchant ID exists

	userID, err := users.GetUserByMerchantId(merchantID.(string))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Merchant not found"})
		return
	}

	// Get user by ID
	user, err := users.GetUserByID(uint(userID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve user", "details": err.Error()})
		return
	}
	fmt.Println(user)

	// fmt.Printf("Payment initiated for user: %s with amount: %.2f\n", user.Name, req.Amount)

	// Generate secureId
	// secureID := mpesa.GenerateSecureID()
	// dateAdded := time.Now().Unix()

	// Check merchant's balance
	balance, err := balances.GetMerchantBalance(merchantID.(string))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve merchant balance", "details": err.Error()})
		return
	}

	// Success response
	c.JSON(http.StatusOK, gin.H{
		"message":  "success",
		"balances": balance,
	})
}

func ConvertMerchantBalancesHandler(c *gin.Context) {
	merchantID, merchantExists := c.Get("merchantID")
	fmt.Println("merchantID", merchantID)
	username, userExists := c.Get("username")
	fmt.Println("username", username)

	if !merchantExists || !userExists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"status":  "error",
			"message": "Missing authentication details",
		})
		return
	}

	// Get the Authorization header
	authHeader := c.GetHeader("Authorization")
	if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization header is required"})
		return
	}

	// Extract the token from the Bearer scheme
	tokenString := strings.TrimPrefix(authHeader, "Bearer ")
	if tokenString == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid authorization token format"})
		return
	}

	// Verify the token
	err := auth.VerifyToken(tokenString)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or expired token", "details": err.Error()})
		return
	}

	// Parse the mobile payment request
	var req ConvertBalance
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input", "details": err.Error()})
		return
	}

	// Check if the merchant ID exists

	// userID, err := users.GetUserByMerchantId(merchantID.(string))
	// if err != nil {
	// 	c.JSON(http.StatusNotFound, gin.H{"error": "Merchant not found"})
	// 	return
	// }

	// Get user by ID
	// user, err := users.GetUserByID(uint(userID))
	// if err != nil {
	// 	c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve user", "details": err.Error()})
	// 	return
	// }
	// fmt.Println(user)

	// fmt.Printf("Payment initiated for user: %s with amount: %.2f\n", user.Name, req.Amount)
	err1 := balances.ConvertBalance(merchantID.(string), req.OriginCurrency, req.DestinationCurrency, req.Amount, req.Amount)
	if err1 != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to convert balance", "details": err.Error()})
		return
	}

	// Generate secureId
	// secureID := mpesa.GenerateSecureID()
	// dateAdded := time.Now().Unix()

	// Check merchant's balance
	balance, err := balances.GetMerchantBalance(merchantID.(string))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve merchant balance", "details": err.Error()})
		return
	}

	// Success response
	c.JSON(http.StatusOK, gin.H{
		"message":  "balance updated successfully",
		"balances": balance,
	})
}

// RegisterRoutes registers the USDC-related routes with the router.
func RegisterRoutes(router *gin.RouterGroup) {
	router.GET("/", LoginHandler)
	router.POST("mobile/initiate", MobilePaymentHandler)
	router.POST("mobile/transfer", MobileWithdrawalHandler)
	router.POST("card/initiate", CardPaymentHandler)
	router.POST("usdc/initiate", UsdcPaymentHandler)
	router.POST("mobile/callback", MobileCallbackHandler)
	router.POST("card/callback", CardCallbackHandler)
	router.POST("usdc/callback", CryptoCallbackHandler)
	router.POST("links/tags", TagsHandler)
	router.GET("transaction", GetTransactionHandler)
	// add a route for bank transfers
	//these are protected routes
	protected := router.Group("/")
	protected.Use(auth.AuthMiddleware())
	protected.POST("bank/transfer", BankTransferHandler)
	//add a protected route to get all merchant balance in differnt currency
	protected.GET("wallet/balances", GetMerchantBalanceHandler)
	// converts the merchant balance
	protected.POST("wallet/convert", ConvertMerchantBalancesHandler)

}
