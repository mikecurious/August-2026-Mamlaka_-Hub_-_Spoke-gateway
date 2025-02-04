package merchants

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	// "net/http"

	"strings"

	"com.mam-laka/auth"
	"com.mam-laka/balances"
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

/*
 go run main.go
VgN8mZljJJg4ycWhVTKxeOrWEqRlResponse: {
            "ConversationID": "AG_20250201_206051c643ca6389833f",
            "OriginatorConversationID": "9021-4f51-8691-8604c07e13de14061111",
            "ResponseCode":"0",
            "ResponseDescription": "Accept the service request successfully."
        }

*/

type B2BResponse struct {
	ConversationID           string `json:"ConversationID"`
	OriginatorConversationID string `json:"OriginatorConversationID"`
	ResponseCode             string `json:"ResponseCode"`
	ResponseDescription      string `json:"ResponseDescription"`
}

//WITHdrawal sample request

/*
	{
	    "impalaMerchantId":"{{username}}",
	    "currency":"KES",
	    "amount":10,
	    "recipientPhone":"254112299271",
	    "mobileMoneySP":"M-Pesa",
	    "externalId":"joeltest4",
	    "callbackUrl":""
	}
*/
type MobileWithdrawalRequest struct {
	ImpalaMerchantId string  `json:"impalaMerchantId" binding:"required"`
	Currency         string  `json:"currency" binding:"required"`
	Amount           float32 `json:"amount" binding:"required"`
	RecipientPhone   string  `json:"recipientPhone" binding:"required"`
	MobileMoneySP    string  `json:"mobileMoneySP" binding:"required"`
	ExternalID       string  `json:"externalId" binding:"required"`
	CallbackURL      string  `json:"callbackUrl" binding:"required"`
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
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to initiate payment", "details": errror_stk})
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
	var req MobileWithdrawalRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input 2"})
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
	fmt.Printf("Payment initiated for user: %s with amount: %.2f\n", user.Name, req.Amount)

	// Here you would initiate the mobile payment logic, e.g., interacting with a payment API.
	// This is just an example response.
	//call the initiate payment method
	//StkPush(phoneNumber string, amount int, callbackURL, accountReference string

	// Generate secureId and other dynamic fields
	secureID := mpesa.GenerateSecureID()

	dateAdded := time.Now().Format("2006-01-02 15:04:05")
	//check the balance for the marchant
	// Call GetMerchantBalance to retrieve the merchant's balance
	balance, err := balances.GetMerchantBalance(req.ImpalaMerchantId)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get  payout balance ", "details": "test"})
		return
	}

	// Print the KES balance
	if balance.KESBalance < float64(req.Amount) {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to initiate transaction kindly top up your payout wallet", "details": "test"})
		return

	}

	// Deduct the balance
	err = balances.DeductKESBalance(req.ImpalaMerchantId, float64(req.Amount))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to deduct amount", "details": err.Error()})
		return
	}
	fmt.Printf("KES Balance for Merchant %s: %.2f\n", req.ImpalaMerchantId, balance.KESBalance)

	// Replace with actual logic for initiating the M-Pesa request
	b2bResponse, errror_b2b := mpesa.GenerateB2CRequest(req.RecipientPhone, float64(req.Amount), req.CallbackURL, req.ExternalID)
	// StkPush(phoneNumber string, amount int, callbackURL, accountReference string) (*StkPushResponse, error) {

	if errror_b2b != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to initiate payment", "details": "test"})
		return
	}
	// Create the transaction record in the database
	newTransaction := &transactions.TransactionModel{
		ImpalaMerchantID:    req.ImpalaMerchantId,
		MerchantRequestID:   &b2bResponse.OriginatorConversationID,
		CheckoutRequestID:   &b2bResponse.ConversationID,
		ResponseDescription: &b2bResponse.ResponseDescription,
		ResponseCode:        &b2bResponse.ResponseCode,
		Currency:            req.Currency,
		Amount:              int(req.Amount),
		Msisdn:              req.RecipientPhone,
		NetAmount:           float64(req.Amount), // Adjust if there are transaction fees
		SecureID:            &secureID,
		SourceOfFunds:       req.MobileMoneySP,
		ExternalID:          &req.ExternalID,
		CallbackURL:         &req.CallbackURL,
		DateAdded:           dateAdded,
		TransactionReport:   "withdraw",
		TransactionStatus:   "PENDING", // Set an initial status
	}

	db := database.GetConnection()
	if err := db.Create(newTransaction).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create transaction", "details": err.Error()})
		return
	}

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
	data := fmt.Sprintf("amount=%.2f&merchant=%s&callback=%s&redirect=%s&externalid=%s", req.Amount, req.ImpalaMerchantId, req.CallbackURL, secureID, req.ExternalID)
	fmt.Println(data)

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
		Result struct {
			ResultCode               int    `json:"ResultCode"`
			ResultDesc               string `json:"ResultDesc"`
			OriginatorConversationID string `json:"OriginatorConversationID"`
			TransactionID            string `json:"TransactionID"`
			ResultParameters         struct {
				ResultParameter []struct {
					Key   string `json:"Key"`
					Value string `json:"Value"`
				} `json:"ResultParameter"`
			} `json:"ResultParameters"`
		} `json:"Result"`
	}

	// Parse the incoming JSON request
	if err := c.ShouldBindJSON(&callbackBody); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid callback body", "details": err.Error()})
		return
	}

	db := database.GetConnection()
	var transaction transactions.TransactionModel

	// Check if the callback is a mobile payment initialization (stkCallback)
	if callbackBody.Body.StkCallback.MerchantRequestID != "" {
		// Retrieve the transaction by MerchantRequestID for mobile payment initialization
		if err := db.Where("merchantRequestID = ?", callbackBody.Body.StkCallback.MerchantRequestID).First(&transaction).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Transaction not found", "details": err.Error()})
			return
		}

		stkCallback := callbackBody.Body.StkCallback

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
			// Call the SendCallback function to send the callback response to the merchant
			callbackResponse := map[string]interface{}{
				"transactionStatus": "COMPLETE",
				"transactionReport": callbackBody.Result.ResultDesc,
				"currency":          transaction.Currency,
				"amount":            transaction.Amount,
				"netAmount":         transaction.NetAmount,
				"secureId":          transaction.SecureID,
				"externalId":        transaction.ExternalID, // Get from DB, not callback
			}

			if err := SendCallback(transaction.ID, callbackResponse); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to send callback", "details": err.Error()})
				return
			}

			c.JSON(http.StatusOK, gin.H{"message": "Callback processed and status updated to SENT"})
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
			// Call the SendCallback function to send the callback response to the merchant
			callbackResponse := map[string]interface{}{
				"transactionStatus": "FAILED",
				"transactionReport": callbackBody.Result.ResultDesc,
				"currency":          transaction.Currency,
				"amount":            transaction.Amount,
				"netAmount":         transaction.NetAmount,
				"secureId":          transaction.SecureID,
				"externalId":        transaction.ExternalID, // Get from DB, not callback
			}

			if err := SendCallback(transaction.ID, callbackResponse); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to send callback", "details": err.Error()})
				return
			}
		}
	} else if callbackBody.Result.OriginatorConversationID != "" {
		// Retrieve the transaction by OriginatorConversationID for withdrawal
		if err := db.Where("merchantRequestID = ?", callbackBody.Result.OriginatorConversationID).First(&transaction).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Transaction not found", "details": err.Error()})
			return
		}

		// Process the ResultCode to determine transaction success or failure
		if callbackBody.Result.ResultCode == 0 { // Success
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
			callbackResponse := map[string]interface{}{
				"transactionStatus": "COMPLETE",
				"transactionReport": callbackBody.Result.ResultDesc,
				"currency":          transaction.Currency,
				"amount":            transaction.Amount,
				"netAmount":         transaction.NetAmount,
				"secureId":          transaction.SecureID,
				"externalId":        transaction.ExternalID, // Get from DB, not callback
			}

			if err := SendCallback(transaction.ID, callbackResponse); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to send callback", "details": err.Error()})
				return
			}
		} else { // Failure
			// Update the transaction status to FAILED
			if err := db.Model(&transactions.TransactionModel{}).
				Where("id = ?", transaction.ID).
				Updates(map[string]interface{}{
					"transactionStatus":   "FAILED",
					"responseDescription": callbackBody.Result.ResultDesc,
					"callbackStatus":      "SENT",
				}).Error; err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update transaction", "details": err.Error()})
				return
			}
			callbackResponse := map[string]interface{}{
				"transactionStatus": "",
				"transactionReport": callbackBody.Result.ResultDesc,
				"currency":          transaction.Currency,
				"amount":            transaction.Amount,
				"netAmount":         transaction.NetAmount,
				"secureId":          transaction.SecureID,
				"externalId":        transaction.ExternalID, // Get from DB, not callback
			}

			if err := SendCallback(transaction.ID, callbackResponse); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to send callback", "details": err.Error()})
				return
			}
		}
	} else {
		// If neither MerchantRequestID nor OriginatorConversationID is present
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid callback: Missing MerchantRequestID or OriginatorConversationID"})
		return
	}

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

	dateAdded := time.Now().Format("2006-01-02 15:04:05")

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
		MerchantRequestID:   &secureID,
		CheckoutRequestID:   &secureID,
		ResponseDescription: &cardResponse,
		ResponseCode:        &cardResponseCode,
		Currency:            Tag.Currency,
		Amount:              int(Tag.Amount),
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
	data := fmt.Sprintf("amount=%.2f&merchant=%s&callback=%s&redirect=%s&externalid=%s", Tag.Amount, Tag.Tag, callbackUrl, secureID, externalId)
	// fmt.Println(data)

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

	// c.JSON(http.StatusOK, gin.H{
	// 	"message":  "card Payment  initiation successful",
	// 	"cardLink": cardLinkResponse,
	// 	"secureId": secureID,
	// })

	// fmt.Println(Tag.Amount)

}

// RegisterRoutes registers the USDC-related routes with the router.
func RegisterRoutes(router *gin.RouterGroup) {
	// router.POST("/check-balance", CheckBalanceHandler)
	// router.POST("/send/usdc", SendUSDCHandler)
	router.GET("/", LoginHandler)
	router.POST("mobile/initiate", MobilePaymentHandler)
	router.POST("mobile/transfer", MobileWithdrawalHandler)
	router.POST("card/initiate", CardPaymentHandler)
	router.POST("mobile/callback", MobileCallbackHandler)
	router.POST("card/callback", CardCallbackHandler)
	router.POST("links/tags", TagsHandler)

}
