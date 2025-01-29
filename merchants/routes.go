package merchants

import (
	"encoding/base64"
	"fmt"
	"net/http"
	"time"

	// "net/http"

	"strings"

	"com.mam-laka/auth"
	"com.mam-laka/card"
	"com.mam-laka/database"
	"com.mam-laka/mpesa"
	"com.mam-laka/transactions"
	"com.mam-laka/users"

	"github.com/gin-gonic/gin"
)

type STKResponse struct {
	MerchantRequestID   string `json:"MerchantRequestID"`
	CheckoutRequestID   string `json:"CheckoutRequestID"`
	ResponseCode        string `json:"ResponseCode"`
	ResponseDescription string `json:"ResponseDescription"`
	CustomerMessage     string `json:"CustomerMessage"`
}

// MobilePaymentRequest structure to bind incoming JSON request
type MobilePaymentRequest struct {
	ImpalaMerchantId string `json:"impalaMerchantId" binding:"required"`
	Currency         string `json:"currency" binding:"required"`
	Amount           int    `json:"amount" binding:"required"`
	DisplayName      string `json:"displayName" binding:"required"`
	PayerPhone       string `json:"payerPhone" binding:"required"`
	MobileMoneySP    string `json:"mobileMoneySP" binding:"required"`
	ExternalID       string `json:"externalId" binding:"required"`
	CallbackURL      string `json:"callbackUrl" binding:"required"`
}
type CardPaymentRequest struct {
	ImpalaMerchantId string  `json:"impalaMerchantId" binding:"required"`
	Currency         string  `json:"currency" binding:"required"`
	Amount           float32 `json:"amount" binding:"required"`
	MobileMoneySP    string  `json:"mobileMoneySP" binding:"required"`
	ExternalID       string  `json:"externalId" binding:"required"`
	CallbackURL      string  `json:"callbackUrl" binding:"required"`
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

	dateAdded := time.Now().Format("2006-01-02 15:04:05")

	// Replace with actual logic for initiating the M-Pesa request
	stkResponse, errror_stk := mpesa.StkPush(req.PayerPhone, req.Amount, req.CallbackURL, req.DisplayName)
	// StkPush(phoneNumber string, amount int, callbackURL, accountReference string) (*StkPushResponse, error) {

	if errror_stk != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to initiate payment", "details": "test"})
		return
	}
	// Create the transaction record in the database
	newTransaction := &transactions.TransactionModel{
		ImpalaMerchantID:    req.ImpalaMerchantId,
		MerchantRequestID:   &stkResponse.MerchantRequestID,
		CheckoutRequestID:   &stkResponse.CheckoutRequestID,
		ResponseDescription: &stkResponse.ResponseDescription,
		ResponseCode:        &stkResponse.ResponseCode,
		Currency:            req.Currency,
		Amount:              req.Amount,
		Msisdn:              req.PayerPhone,
		NetAmount:           float64(req.Amount), // Adjust if there are transaction fees
		SecureID:            &secureID,
		SourceOfFunds:       req.MobileMoneySP,
		ExternalID:          &req.ExternalID,
		CallbackURL:         &req.CallbackURL,
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
		"transactionId": newTransaction.ID,
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
	fmt.Printf("Payment initiated for user: %s with amount: %d\n", user.Name, req.Amount)

	// Here you would initiate the mobile payment logic, e.g., interacting with a payment API.
	// This is just an example response.
	//call the initiate payment method
	//StkPush(phoneNumber string, amount int, callbackURL, accountReference string

	// Generate secureId and other dynamic fields
	secureID := mpesa.GenerateSecureID()

	dateAdded := time.Now().Format("2006-01-02 15:04:05")

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
		MerchantRequestID:   &secureID,
		CheckoutRequestID:   &secureID,
		ResponseDescription: &cardResponse,
		ResponseCode:        &cardResponseCode,
		Currency:            req.Currency,
		Amount:              int(req.Amount),
		Msisdn:              "Null",
		NetAmount:           float64(req.Amount), // Adjust if there are transaction fees
		SecureID:            &secureID,
		SourceOfFunds:       req.MobileMoneySP,
		ExternalID:          &req.ExternalID,
		CallbackURL:         &req.CallbackURL,
		DateAdded:           dateAdded,
		TransactionReport:   "collection",
		TransactionStatus:   "PENDING", // Set an initial status
	}

	db := database.GetConnection()
	if err := db.Create(newTransaction).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create transaction", "details": err.Error()})
		return
	}
	data := fmt.Sprintf("amount=%sf&merchant=%s&callback=%s&redirect=%s&externalid=%s", req.Amount, req.ImpalaMerchantId, req.CallbackURL, secureID, req.ExternalID)

	// Encode the string in Base64
	encoded := base64.StdEncoding.EncodeToString([]byte(data))

	// Print the Base64 encoded string
	fmt.Println("Base64 Encoded Data:", encoded)
	cardlink := "https://mpgs.cradlevoices.com/mpgs.php?data=" + encoded

	c.JSON(http.StatusOK, gin.H{
		"message":  "card Payment  initiation successful",
		"cardLink": cardlink,
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
	// Make request to function to get user by username
	user_id, err := users.GetUserByMerchantId(username)
	fmt.Println("user id", user_id)

	// Authenticate the user
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "merchant not found"})
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
	token, expirationDate, err := auth.CreateToken(username)
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
func MobileCallbackHandler(c *gin.Context) {
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
						Value interface{} `json:"Value"`
					} `json:"Item"`
				} `json:"CallbackMetadata"`
			} `json:"stkCallback"`
		} `json:"Body"`
	}

	// Parse the incoming JSON request
	if err := c.ShouldBindJSON(&callbackBody); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid callback body", "details": err.Error()})
		return
	}

	stkCallback := callbackBody.Body.StkCallback
	db := database.GetConnection()

	// Retrieve the transaction by MerchantRequestID
	var transaction transactions.TransactionModel
	if err := db.Where("merchantRequestID = ?", stkCallback.MerchantRequestID).First(&transaction).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Transaction not found", "details": err.Error()})
		return
	}

	// Process the ResultCode to determine transaction success or failure
	if stkCallback.ResultCode == 0 { // Success
		// Extract metadata
		metadata := make(map[string]interface{})
		for _, item := range stkCallback.CallbackMetadata.Items {
			metadata[item.Name] = item.Value
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
	} else { // Failure
		// Update the transaction status to FAILED
		if err := db.Model(&transactions.TransactionModel{}).
			Where("id = ?", transaction.ID).
			Updates(map[string]interface{}{
				"transactionStatus":   "FAILED",
				"responseDescription": stkCallback.ResultDesc,
				"callbackStatus":      "SENT",
			}).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update transaction", "details": err.Error()})
			return
		}
	}
	// Call the SendCallback function to send the callback response to the merchant
	// body, err := ioutil.ReadAll(r.Body)

	if err := SendCallback(transaction.ID, callbackBody); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to send callback", "details": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Callback processed and status updated to SENT"})
}

func CardCallbackHandler(c *gin.Context) {
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

	// Extract the MerchantRequestID from the RedirectURL
	merchantRequestID := callbackBody.RedirectURL

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

	// Call the SendCallback function to send the callback response to the merchant
	if err := SendCallback(transaction.ID, callbackBody); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to send callback", "details": err.Error()})
		return
	}

	// Respond with success
	c.JSON(http.StatusOK, gin.H{"message": "Callback processed and status updated to SENT"})
}

func TagsHandler(c *gin.Context) {
	var Tag struct {
		Tag      string `json:"tag" binding:"required"`
		Amount   int    `json:"amount" binding:"required"`
		Currency string `json:"currency" binding:"required"`
	}

	if err := c.ShouldBindJSON(&Tag); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format", "details": err.Error()})
		return
	}

	// Generate secureId and other dynamic fields
	secureID := mpesa.GenerateSecureID()

	dateAdded := time.Now().Format("2006-01-02 15:04:05")

	// Replace with actual logic for initiating the M-Pesa request
	// stkResponse, errror_stk := mpesa.StkPush(req.PayerPhone, req.Amount, req.CallbackURL, req.DisplayName)
	cardLinkResponse, card_errror := card.GenerateCardPaymentLink(Tag.Currency, float64(Tag.Amount), secureID, secureID, secureID)
	// StkPush(phoneNumber string, amount int, callbackURL, accountReference string) (*StkPushResponse, error) {

	if card_errror != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to initiate payment", "details": "test"})
		return
	}
	cardResponse := "Card payment"
	cardResponseCode := "200"
	externalId := "200"
	callbackUrl := "200"

	// Create the transaction record in the database
	newTransaction := &transactions.TransactionModel{
		ImpalaMerchantID:    Tag.Tag,
		MerchantRequestID:   &secureID,
		CheckoutRequestID:   &secureID,
		ResponseDescription: &cardResponse,
		ResponseCode:        &cardResponseCode,
		Currency:            Tag.Currency,
		Amount:              Tag.Amount,
		Msisdn:              "Null",
		NetAmount:           float64(Tag.Amount), // Adjust if there are transaction fees
		SecureID:            &secureID,
		SourceOfFunds:       "card",
		ExternalID:          &externalId,
		CallbackURL:         &callbackUrl,
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
		"message":  "card Payment  initiation successful",
		"cardLink": cardLinkResponse,
		"secureId": secureID,
	})

	// fmt.Println(Tag.Amount)

}

// RegisterRoutes registers the USDC-related routes with the router.
func RegisterRoutes(router *gin.RouterGroup) {
	// router.POST("/check-balance", CheckBalanceHandler)
	// router.POST("/send/usdc", SendUSDCHandler)
	router.GET("/", LoginHandler)
	router.POST("mobile/initiate", MobilePaymentHandler)
	router.POST("card/initiate", CardPaymentHandler)
	router.POST("mobile/callback", MobileCallbackHandler)
	router.POST("card/callback", CardCallbackHandler)
	router.POST("links/tags", TagsHandler)

}
