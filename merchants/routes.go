package merchants

import (
	"bytes"
	"context"
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
	"com.mam-laka/cameroon"
	"com.mam-laka/database"
	"com.mam-laka/flutterwave"
	"com.mam-laka/korapay"
	"com.mam-laka/mpesa"
	"com.mam-laka/payaza"
	"com.mam-laka/pesalink"
	"com.mam-laka/transactions"
	"com.mam-laka/uganda"
	"com.mam-laka/users"
	virtualcards "com.mam-laka/virtulcards"
	"com.mam-laka/westafrica"
	"gorm.io/gorm"
	"mam-laka.com/merchant/drawings"

	"github.com/gin-gonic/gin"
)

// west africa

// AirtimeCallbackRequest represents the incoming callback payload
type AirtimeCallbackRequest struct {
	TransactionID           string  `json:"transaction_id"`
	Amount                  int     `json:"amount"`
	Benefice                int     `json:"benefice"`
	Commission              int     `json:"comission"`
	Destination             string  `json:"destination"`
	Fee                     int     `json:"fee"`
	Response                string  `json:"response"`
	Error                   *string `json:"error"`
	ServiceID               int     `json:"service_id"`
	CustomerName            string  `json:"customer_name"`
	State                   string  `json:"state"`
	CustomData              string  `json:"custom_data"`
	IPNUrl                  string  `json:"ipn_url"`
	TransactionChannel      string  `json:"transaction_channel"`
	ProviderID              string  `json:"provider_id"`
	SMSLink                 int     `json:"sms_link"`
	CreatedAt               string  `json:"created_at"`
	UpdatedAt               string  `json:"updated_at"`
	IPNState                int     `json:"ipn_state"`
	WAmountAfterTransaction string  `json:"w_amount_after_transaction"`
	PLastWalletAmount       int     `json:"p_last_wallet_amount"`
	PNewWalletAmount        int     `json:"p_new_wallet_amount"`
	PID                     int     `json:"p_id"`
	Hash                    string  `json:"hash"`
	Currency                string  `json:"currency"`
}

// CallbackResponse represents the response sent back to merchant
type CallbackResponse struct {
	Amount            int    `json:"amount"`
	Currency          string `json:"currency"`
	ExternalID        string `json:"externalId"`
	NetAmount         int    `json:"netAmount"`
	SecureID          string `json:"secureId"`
	TransactionReport string `json:"transactionReport"`
	TransactionStatus string `json:"transactionStatus"`
	Reason            string `json:"reason,omitempty"` // Only included for failed transactions
}

// TransactionType represents the type of transaction
type TransactionType string

const (
	TransactionTypePayin  TransactionType = "DEPOSIT"
	TransactionTypePayout TransactionType = "WITHDRAW"
)

// WestAfricaCallbackHandler handles airtime transaction callbacks
func WestAfricaCallbackHandler(c *gin.Context) {
	log.Println("📞 Received West Africa callback")

	// Parse the callback request
	var callbackReq AirtimeCallbackRequest
	if err := c.ShouldBindJSON(&callbackReq); err != nil {
		log.Printf(" Failed to parse callback request: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid callback payload", "details": err.Error()})
		return
	}

	// Log the callback details
	log.Printf("🔍 Processing callback for transaction: %s", callbackReq.TransactionID)
	log.Printf("📊 State: %s, Amount: %d, Destination: %s", callbackReq.State, callbackReq.Amount, callbackReq.Destination)

	// Get database connection
	db := database.GetConnection()
	if db == nil {
		log.Println(" Failed to get database connection")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database connection failed"})
		return
	}

	// Retrieve the transaction by external transaction ID
	transaction, err := getTransactionByExternalID(db, callbackReq.TransactionID)
	if err != nil {
		log.Printf(" Transaction not found: %s, Error: %v", callbackReq.TransactionID, err)
		c.JSON(http.StatusNotFound, gin.H{"error": "Transaction not found", "details": err.Error()})
		return
	}

	log.Printf("🔍 Found transaction: ID=%d, Type=%s, Amount=%d", transaction.ID, transaction.TransactionReport, transaction.Amount)
	fmt.Println("STATE", callbackReq.State)
	// Process based on transaction state
	switch strings.ToUpper(callbackReq.State) {
	case "SUCCESSFUL":
		if err := processSuccessfulTransaction(db, transaction, &callbackReq); err != nil {
			log.Printf(" Failed to process successful transaction: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to process successful transaction", "details": err.Error()})
			return
		}
		log.Println(" Successfully processed SUCCESSFUL transaction")
		c.JSON(http.StatusOK, gin.H{"message": "Callback processed successfully - SUCCESSFUL"})

	case "FAILED":
		if err := processFailedTransaction(db, transaction, &callbackReq); err != nil {
			log.Printf(" Failed to process failed transaction: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to process failed transaction", "details": err.Error()})
			return
		}
		log.Println(" Successfully processed FAILED transaction")
		c.JSON(http.StatusOK, gin.H{"message": "Callback processed successfully - FAILED"})

	default:
		log.Printf("⚠️  Unknown transaction state: %s", callbackReq.State)
		c.JSON(http.StatusOK, gin.H{"message": "Callback received but state not processed", "state": callbackReq.State})
	}
}

// processSuccessfulTransaction handles successful transactions
func processSuccessfulTransaction(db *gorm.DB, transaction *transactions.TransactionModel, callback *AirtimeCallbackRequest) error {
	log.Printf(" Processing successful transaction: %s", callback.TransactionID)

	// Update transaction status
	updates := map[string]interface{}{
		"transactionStatus": "SUCCESSFUL",
		"callbackStatus":    "SENT",
	}

	if err := db.Model(&transactions.TransactionModel{}).
		Where("id = ?", transaction.ID).
		Updates(updates).Error; err != nil {
		return fmt.Errorf("failed to update transaction status: %w", err)
	}

	// Handle balance updates based on transaction type
	if err := handleBalanceUpdates(db, transaction, callback, true); err != nil {
		return fmt.Errorf("failed to handle balance updates: %w", err)
	}

	// Send callback to merchant
	callbackResponse := buildCallbackResponse(transaction, callback, "COMPLETED")
	if err := SendCallback(transaction.ID, callbackResponse); err != nil {
		log.Printf("⚠️  Failed to send callback to merchant: %v", err)
		// Don't fail the transaction if callback fails
	}

	// Send token transfer for successful transactions
	// if callback.Amount > 0 {
	// 	go sendTokenTransfer(strconv.Itoa(callback.Amount), "GBR7COBB5T5WPEYI7PN2XXLIYC2VUE22VIF4BHRKA3TPLUSZS6TQY7BE")
	// }

	log.Printf(" Successfully processed successful transaction: %s", callback.TransactionID)
	return nil
}

// processFailedTransaction handles failed transactions
func processFailedTransaction(db *gorm.DB, transaction *transactions.TransactionModel, callback *AirtimeCallbackRequest) error {
	log.Printf(" Processing failed transaction: %s", callback.TransactionID)

	// errorMsg := "FAILED"
	// if callback.Error != nil && *callback.Error != "" {
	// 	errorMsg = *callback.Error
	// }

	// Update transaction status
	updates := map[string]interface{}{
		"transactionStatus": "FAILED",
		"callbackStatus":    "SENT",
	}

	if err := db.Model(&transactions.TransactionModel{}).
		Where("id = ?", transaction.ID).
		Updates(updates).Error; err != nil {
		return fmt.Errorf("failed to update transaction status: %w", err)
	}

	// Handle balance updates for failed transactions (usually reversals)
	if err := handleBalanceUpdates(db, transaction, callback, false); err != nil {
		return fmt.Errorf("failed to handle balance updates: %w", err)
	}

	// Send callback to merchant
	callbackResponse := buildCallbackResponse(transaction, callback, "FAILED")
	// callbackResponse.ErrorMessage = errorMsg
	if err := SendCallback(transaction.ID, callbackResponse); err != nil {
		log.Printf("⚠️  Failed to send callback to merchant: %v", err)
		// Don't fail the transaction if callback fails
	}

	log.Printf(" Successfully processed failed transaction: %s", callback.TransactionID)
	return nil
}

// handleBalanceUpdates handles balance updates based on transaction type and success status
func handleBalanceUpdates(db *gorm.DB, transaction *transactions.TransactionModel, callback *AirtimeCallbackRequest, isSuccessful bool) error {
	transactionType := TransactionType(strings.ToUpper(transaction.TransactionReport))

	log.Printf("Handling balance updates - Type: %s, Successful: %v, Amount: %d", transactionType, isSuccessful, callback.Amount)

	fmt.Println("Transaction Type", transactionType)
	switch transactionType {
	case TransactionTypePayin:
		return handlePayinBalanceUpdate(db, transaction, callback, isSuccessful)
	case TransactionTypePayout:
		return handlePayoutBalanceUpdate(db, transaction, callback, isSuccessful)
	default:
		log.Printf("⚠️  Unknown transaction type: %s", transaction.TransactionReport)
		return nil
	}
}

// handlePayinBalanceUpdate handles balance updates for payin transactions
func handlePayinBalanceUpdate(db *gorm.DB, transaction *transactions.TransactionModel, callback *AirtimeCallbackRequest, isSuccessful bool) error {
	// if !isSuccessful {
	// 	log.Println("Skipping payin balance update for failed transaction")
	// 	return nil
	// }

	log.Printf("Updating merchant collection balance for payin - MerchantID: %s, Amount: %d",
		transaction.ImpalaMerchantID, callback.Benefice)

	// For successful payin, add the benefice (amount after fees/commission) to merchant balance

	if err := balances.AddXOFBalance(transaction.ImpalaMerchantID, transaction.NetAmount); err != nil {
		return fmt.Errorf("failed to update merchant collection balance: %w", err)
	}

	log.Printf(" Successfully updated merchant collection balance")
	return nil
}

// handlePayoutBalanceUpdate handles balance updates for payout transactions
func handlePayoutBalanceUpdate(db *gorm.DB, transaction *transactions.TransactionModel, callback *AirtimeCallbackRequest, isSuccessful bool) error {
	fmt.Println("In payout____________")

	if isSuccessful {
		log.Println("Payout successful - balance already deducted during initiation")
		return nil
	}

	log.Printf("Reversing payout balance for failed transaction - MerchantID: %s, Amount: %d",
		transaction.ImpalaMerchantID, transaction.Amount)

	// For failed payout, reverse the deduction by adding back the original amount
	if err := db.Model(&balances.MerchantBalance{}).
		Where("impalaMerchantId = ?", transaction.ImpalaMerchantID).
		Update("impaBalance", gorm.Expr("impaBalance + ?", transaction.Amount)).Error; err != nil {
		return fmt.Errorf("failed to reverse payout balance: %w", err)
	}

	log.Printf(" Successfully reversed payout balance")
	return nil
}

// buildCallbackResponse builds the callback response for the merchant
func buildCallbackResponse(transaction *transactions.TransactionModel, callback *AirtimeCallbackRequest, status string) CallbackResponse {

	response := CallbackResponse{
		TransactionStatus: status,
		TransactionReport: status,
		Currency:          callback.Currency,
		Amount:            callback.Amount,
		NetAmount:         callback.Benefice,
		SecureID:          transaction.SecureID,
		ExternalID:        transaction.ExternalID,
	}

	// Set default currency if not provided
	if response.Currency == "" {
		response.Currency = "XOF" // Default West African currency
	}

	return response
}

// getTransactionByExternalID retrieves transaction by external ID
func getTransactionByExternalID(db *gorm.DB, externalID string) (*transactions.TransactionModel, error) {
	var transaction transactions.TransactionModel

	// Try to find by external ID first
	if err := db.Where("secureId = ?", externalID).First(&transaction).Error; err != nil {
		// If not found by external ID, try by merchant request ID as fallback
		if err := db.Where("secureId = ?", externalID).First(&transaction).Error; err != nil {
			return nil, fmt.Errorf("transaction not found: %w", err)
		}
	}

	return &transaction, nil
}

// Helper function to log callback details (can be used for debugging)
func logCallbackDetails(callback *AirtimeCallbackRequest) {
	log.Printf("🔍 Callback Details:")
	log.Printf("  Transaction ID: %s", callback.TransactionID)
	log.Printf("  State: %s", callback.State)
	log.Printf("  Amount: %d", callback.Amount)
	log.Printf("  Benefice: %d", callback.Benefice)
	log.Printf("  Fee: %d", callback.Fee)
	log.Printf("  Commission: %d", callback.Commission)
	log.Printf("  Destination: %s", callback.Destination)
	log.Printf("  Provider ID: %s", callback.ProviderID)
	log.Printf("  Customer Name: %s", callback.CustomerName)
	log.Printf("  Currency: %s", callback.Currency)
	if callback.Error != nil {
		log.Printf("  Error: %s", *callback.Error)
	}
}

// remove prfix
func RemovePlusPrefix(phone string) string {
	if strings.HasPrefix(phone, "+") {
		return phone[1:] // Remove the first character
	}
	return phone
}

// MobilePaymentHandler to handle mobile payment initiation
func MobilePaymentHandler(c *gin.Context) {
	// Authrorization already dont on anothr page before this handler is called

	// Parse the mobile payment request
	var req MobilePaymentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input3"})
		return
	}

	// Prevent duplicate externalId for this merchant
	if req.ExternalID != "" {
		if exists, err := transactions.ExternalIDExists(req.ImpalaMerchantId, req.ExternalID); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to validate externalId", "details": err.Error()})
			return
		} else if exists {
			c.JSON(http.StatusConflict, gin.H{
				"error":   "DUPLICATE_EXTERNAL_ID",
				"message": "A transaction with this externalId already exists for this merchant",
			})
			return
		}
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

	// Generate secureId and other dynamic fields
	secureID := mpesa.GenerateSecureID()

	dateAdded := time.Now().Unix()
	// RemovePlusPrefix removes the '+' sign from the beginning of a phone number if present.
	// check which currency the payment is being made from
	var stkResponse *mpesa.StkPushResponse
	var errror_stk error
	var merchantRequestID string
	var checkoutRequestID string
	var responseDescription string
	var responseCode string

	if req.Currency == "KES" {

		stkResponse, errror_stk = mpesa.StkPush(RemovePlusPrefix(req.PayerPhone), req.Amount, req.CallbackURL, user.Name)
		merchantRequestID = stkResponse.MerchantRequestID
		checkoutRequestID = stkResponse.CheckoutRequestID
		responseDescription = stkResponse.ResponseDescription
		responseCode = stkResponse.ResponseCode

	} else if req.Currency == "XOF" || req.Currency == "UGX" { // WE USING PAYAZA FOR THIS

		// get country code from phone
		countryCode := payaza.DetectCountryCode(req.PayerPhone)
		CustomerBankCode := payaza.GetBankCode(countryCode, req.MobileMoneySP)

		fmt.Println("Determined Bank Code:", CustomerBankCode)
		// prepare payload
		payload := payaza.PayazaPayload{
			Amount:                 req.Amount,
			CustomerNumber:         req.PayerPhone,
			TransactionReference:   secureID,
			TransactionDescription: "Test Payment",
			CustomerBankCode:       CustomerBankCode,
			CurrencyCode:           req.Currency,
			CustomerEmail:          "bigmaitre@blondmail.com",
			CustomerFirstName:      "Robert",
			CustomerLastName:       "Stones",
			CustomerPhoneNumber:    "2290196289492",
			CountryCode:            countryCode,
		}

		stkResponse, errror_stk := payaza.SendPayazaRequest(payload)
		if errror_stk != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Payment initiation failed", "details": stkResponse})
			return
		}

		if stkResponse.IsSuccess() {
			fmt.Println("SUCCESS (PENDING):")
			fmt.Println("Transaction Ref:", stkResponse.TransactionReference)
			fmt.Println("Payment Token:", stkResponse.PaymentToken)
		} else {
			fmt.Println("FAILED / DECLINED:")
			fmt.Println("Code:", stkResponse.ResponseCode)
			fmt.Println("Message:", stkResponse.ResponseMessage)
		}
		merchantRequestID = stkResponse.TransactionReference
		checkoutRequestID = stkResponse.PaymentToken
		responseDescription = "Merchant initiated XOF payment via Payaza"
		responseCode = stkResponse.ResponseCode

	}

	// Replace with actual logic for initiating the M-Pesa request
	// StkPush(phoneNumber string, amount int, callbackURL, accountReference string) (*StkPushResponse, error) {

	if errror_stk != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to initiate payment", "details": errror_stk})
		return
	}
	// Create the transaction record in the database
	newTransaction := &transactions.TransactionModel{
		ImpalaMerchantID:    req.ImpalaMerchantId,
		MerchantRequestID:   merchantRequestID,
		CheckoutRequestID:   checkoutRequestID,
		ResponseDescription: responseDescription,
		ResponseCode:        responseCode,
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
		"transactionId": merchantRequestID,
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

	// Prevent duplicate externalId for this merchant (if provided)
	if req.ExternalID != "" {
		if exists, err := transactions.ExternalIDExists(req.ImpalaMerchantId, req.ExternalID); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to validate externalId", "details": err.Error()})
			return
		} else if exists {
			c.JSON(http.StatusConflict, gin.H{
				"error":   "DUPLICATE_EXTERNAL_ID",
				"message": "A transaction with this externalId already exists for this merchant",
			})
			return
		}
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

	// Check merchant's balance FIRST before initiating any payout
	balance, err := balances.GetMerchantBalance(req.ImpalaMerchantId)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve merchant balance", "details": err.Error()})
		return
	}

	// print
	fmt.Println("currency", req.Currency)

	//check the currenvy from the request
	switch req.Currency {
	case "KES":
		// Check if merchant has sufficient KES balance BEFORE initiating payout
		if balance.KESBalance < float64(req.Amount) {
			c.JSON(http.StatusBadRequest, gin.H{
				"error":   "INSUFFICIENT_BALANCE",
				"message": fmt.Sprintf("Insufficient KES balance. Available: %.2f KES, Required: %.2f KES", balance.KESBalance, float64(req.Amount)),
			})
			return
		}

		log.Printf("✅ Balance check passed - Available: %.2f KES, Required: %.2f KES", balance.KESBalance, float64(req.Amount))

		// Do NOT deduct at initiation. We only initiate the B2C payout here.
		// Balance will be deducted in the callback handler after Safaricom confirms success.

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
	case "UGX":
		ugxBalance := balance.UGXBalance

		if ugxBalance < float64(req.Amount) {
			c.JSON(http.StatusOK, gin.H{
				"error":   "INSUFFICIENT_BALANCE",
				"message": fmt.Sprintf("Insufficient balance. Available: %.2f UGX", ugxBalance),
			})
			return
		}

		// Deduct balance before attempting payment
		err = balances.DeductUGXBalance(req.ImpalaMerchantId, float64(req.Amount))
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":   "BALANCE_DEDUCTION_FAILED",
				"message": "Failed to deduct amount from balance",
				"details": err.Error(),
			})
			return
		}

		fmt.Printf("UGX Balance deducted for Merchant %s: %.2f\n", req.ImpalaMerchantId, float64(req.Amount))

		// Initiate payment via reize remit
		status, message, err := uganda.SendMoneyToPhoneReal(req.RecipientPhone, float64(req.Amount))
		fmt.Printf("payment status: %s, message: %s, error: %v\n", status, message, err)

		// Determine transaction status and prepare response
		var transactionStatus, responseStatus, responseMessage string
		var responseCode int

		if err == nil {
			transactionStatus = "COMPLETE"
			responseStatus = "SUCCESS"
			responseMessage = "Payment completed successfully"
			responseCode = http.StatusOK
		} else {
			transactionStatus = "FAILED"
			responseStatus = "FAILED"
			responseMessage = "Payment initiation failed"
			responseCode = http.StatusOK

			// Refund the balance since payment failed
			if refundErr := balances.AddUGXBalance(req.ImpalaMerchantId, float64(req.Amount)); refundErr != nil {
				log.Printf("Failed to refund balance for merchant %s: %v", req.ImpalaMerchantId, refundErr)
			}
		}

		// Create transaction record (for both success and failure)
		newTransaction := &transactions.TransactionModel{
			ImpalaMerchantID:    req.ImpalaMerchantId,
			MerchantRequestID:   secureID,
			CheckoutRequestID:   secureID,
			ResponseDescription: fmt.Sprintf("UGX withdrawal - %s", transactionStatus),
			ResponseCode:        message,
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
			TransactionStatus:   transactionStatus,
		}

		// Save transaction to database
		db := database.GetConnection()
		if dbErr := db.Create(newTransaction).Error; dbErr != nil {
			log.Printf("Failed to save transaction: %v", dbErr)
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":   "DATABASE_ERROR",
				"message": "Failed to record transaction",
				"details": dbErr.Error(),
			})
			return
		}

		// Get the transaction ID from the saved record
		transactionID := newTransaction.ID

		// Send API response
		c.JSON(responseCode, gin.H{
			"status":     responseStatus,
			"message":    responseMessage,
			"externalId": req.ExternalID,
			"secureId":   secureID,
		})

		// Prepare callback response
		callbackStatus := "FAILED"
		if transactionStatus == "SUCCESS" {
			callbackStatus = "COMPLETE"
		}

		callbackResponse := map[string]interface{}{
			"transactionStatus": callbackStatus,
			"transactionReport": callbackStatus,
			"currency":          req.Currency, // Use actual currency from request
			"amount":            req.Amount,
			"netAmount":         req.Amount,
			"secureId":          secureID,
			"externalId":        req.ExternalID,
		}

		log.Printf("Sending callback response: %+v", callbackResponse)

		// Send callback to merchant
		if callbackErr := SendCallback(transactionID, callbackResponse); callbackErr != nil {
			log.Printf("Failed to send callback for transaction %d: %v", transactionID, callbackErr)
			// Don't return error to client since the main transaction processing is complete
		}
	case "XOF":
		// ugxBalance := balance.UGXBalance
		XOFBalance := balance.ImpaBalance

		serviceId := func() int {
			id, err := strconv.Atoi(req.MobileMoneySP)
			if err != nil {
				log.Printf("Failed to convert MobileMoneySP to int: %v", err)
				return 0 // default value
			}
			return id
		}()

		// Ensure there’s enough balance before proceeding

		// Define known cash-in and cash-out service IDs
		cashinIDs := []int{170, 174, 172, 8, 152, 150, 154, 162, 166, 168, 164, 50, 148}
		cashoutIDs := []int{171, 175, 173, 7, 153, 151, 155, 163, 167, 169, 165, 49, 149}

		var transactionReport string

		if IsInList(serviceId, cashoutIDs) {
			transactionReport = "deposit"
		} else if IsInList(serviceId, cashinIDs) {
			transactionReport = "withdraw"
			if XOFBalance < float64(req.Amount) {
				c.JSON(http.StatusOK, gin.H{
					"error":   "INSUFFICIENT_BALANCE",
					"message": fmt.Sprintf("Insufficient balance. Available: %.2f XOF", XOFBalance),
				})
				return
			}

			fmt.Println("am here .....")
			// Deduct balance only for withdrawals
			err := balances.DeductXOFBalance(req.ImpalaMerchantId, float64(req.Amount))
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{
					"error":   "BALANCE_DEDUCTION_FAILED",
					"message": "Failed to deduct amount from balance",
					"details": err.Error(),
				})
				return
			}
		} else {
			c.JSON(http.StatusBadRequest, gin.H{
				"error":   "UNKNOWN_SERVICE_ID",
				"message": "Unknown service ID. Please verify and try again.",
			})
			return
		}

		// initiatae west africa client
		client := westafrica.NewAirtimeClient()

		// Initiate payment via reize remit
		westAfricaRequest := &westafrica.AirtimeRequest{
			Amount:      int(req.Amount),
			Destination: req.RecipientPhone,
			APIKey:      "PIX_737219e4-4980-4000-b0a9-a0393bbcaf28",
			IPNUrl:      "https://payments.mam-laka.com/api/v1/west-africa/callback",
			ServiceID: func() int {
				id, err := strconv.Atoi(req.MobileMoneySP)
				if err != nil {
					log.Printf("Failed to convert MobileMoneySP to int: %v", err)
					return 0 // Default value in case of error
				}
				return id
			}(),
			OMOTP:      strconv.Itoa(req.OMOTP),
			CustomData: "your_custom_data",
		}

		// Send request with context and timeout
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		response, err := client.SendAirtimeTransaction(ctx, westAfricaRequest)

		// log the raw response as JSON (if not nil)
		if response != nil {
			if respJSON, errMarshal := json.MarshalIndent(response, "", "  "); errMarshal == nil {
				log.Printf("WestAfrica Response:\n%s", string(respJSON))
			} else {
				log.Printf("Failed to marshal WestAfrica response: %v", errMarshal)
			}
		}

		if err != nil {
			details := ""
			if response != nil {
				details = response.Message
			}

			c.JSON(http.StatusInternalServerError, gin.H{
				"status":  500,
				"error":   "1991",
				"message": "Failed to initiate payment",
				"details": details,
				// "resp":    response,
				"err": err.Error(),
			})
			return
		}

		// ✅ Success
		// c.JSON(http.StatusOK, gin.H{
		// 	"status":  response.StatusCode,
		// 	"state":   response.Data.State,
		// 	"message": response.Message,
		// 	// "resp":    response,
		// })

		// status, message, err := uganda.SendMoneyToPhoneReal(req.RecipientPhone, float64(req.Amount))
		// fmt.Printf("payment status: %s, message: %s, error: %v\n", status, message, err)

		// Determine transaction status and prepare response
		var transactionStatus, responseStatus, responseMessage string
		var responseCode int

		if err == nil {
			transactionStatus = "PENDING"
			responseStatus = "SUCCESS"
			responseMessage = "Payment initiated  successfully"
			responseCode = http.StatusOK
		} else {
			transactionStatus = "FAILED"
			responseStatus = "FAILED"
			responseMessage = "Payment initiation failed"
			responseCode = http.StatusOK

			// Refund the balance since payment failed
			if refundErr := balances.AddXOFBalance(req.ImpalaMerchantId, float64(req.Amount)); refundErr != nil {
				log.Printf("Failed to refund balance for merchant %s: %v", req.ImpalaMerchantId, refundErr)
			}
		}

		// Create transaction record (for both success and failure)
		newTransaction := &transactions.TransactionModel{
			ImpalaMerchantID:    req.ImpalaMerchantId,
			MerchantRequestID:   response.Data.TransactionID,
			CheckoutRequestID:   response.Data.TransactionID,
			ResponseDescription: response.Message,
			ResponseCode:        response.Data.State,
			Currency:            req.Currency,
			Amount:              int(req.Amount),
			Msisdn:              req.RecipientPhone,
			NetAmount:           float64(req.Amount),
			SecureID:            response.Data.TransactionID,
			SourceOfFunds:       req.MobileMoneySP,
			ExternalID:          req.ExternalID,
			CallbackURL:         req.CallbackURL,
			DateAdded:           dateAdded,
			TransactionReport:   transactionReport,
			TransactionStatus:   transactionStatus,
		}

		// Save transaction to database
		db := database.GetConnection()
		if dbErr := db.Create(newTransaction).Error; dbErr != nil {
			log.Printf("Failed to save transaction: %v", dbErr)
			c.JSON(http.StatusInternalServerError, gin.H{
				"status":  401,
				"error":   "1994",
				"message": "Failed to record transaction",
				"details": dbErr.Error(),
			})
			return
		}

		// Get the transaction ID from the saved record
		// transactionID := newTransaction.ID

		// Send API response
		c.JSON(responseCode, gin.H{
			"status":     responseStatus,
			"message":    responseMessage,
			"error":      "1995",
			"externalId": req.ExternalID,
			"secureId":   response.Data.TransactionID,
		})
	case "XAF":
		// check the mobile service sp
		switch req.MobileMoneySP {
		case "200": //cameroon collect
			token, err := cameroon.GetAccessToken()
			if err != nil {
				log.Fatalf("Error getting token: %v", err)
			}
			secureID := mpesa.GenerateSecureID()

			err = cameroon.SendCollectRequest(token, req.RecipientPhone, float64(req.Amount), secureID, "https://webhook.site/c24e095f-d9af-4b30-a2ad-3ae5dc048400")
			if err != nil {
				c.JSON(http.StatusOK, gin.H{
					"details":       err,
					"message":       "Payment initiation successful",
					"code":          "1996",
					"transactionId": req.ExternalID,
					"secureId":      secureID,
				})
				return
			}

			// Create transaction record
			newTransaction := &transactions.TransactionModel{
				ImpalaMerchantID:    req.ImpalaMerchantId,
				MerchantRequestID:   secureID,
				CheckoutRequestID:   secureID,
				ResponseDescription: "Payment initiated via Cameroon Collect",
				ResponseCode:        "0",
				Currency:            req.Currency,
				Amount:              int(req.Amount),
				Msisdn:              req.RecipientPhone,
				NetAmount:           float64(req.Amount),
				SecureID:            secureID,
				SourceOfFunds:       req.MobileMoneySP,
				ExternalID:          req.ExternalID,
				CallbackURL:         req.CallbackURL,
				DateAdded:           dateAdded,
				TransactionReport:   "deposit",
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
				"code":          "1996",
				"transactionId": req.ExternalID,
				"secureId":      secureID,
			})

		case "201":
			// cameroon collect
			token, err := cameroon.GetAccessToken()
			if err != nil {
				log.Fatalf("Error getting token: %v", err)
			}
			secureID := mpesa.GenerateSecureID()
			// check xaf balance
			balance, err := balances.GetMerchantBalance(req.ImpalaMerchantId)
			if balance.XAFBalance < float64(req.Amount) {
				c.JSON(http.StatusOK, gin.H{
					"error":   "INSUFFICIENT_BALANCE",
					"message": fmt.Sprintf("Insufficient balance. Available: %.2f XAF", balance.XAFBalance),
				})
				return
			}

			err = cameroon.SendDisburseRequest(token, req.RecipientPhone, float64(req.Amount), secureID, "https://webhook.site/c24e095f-d9af-4b30-a2ad-3ae5dc048400")
			if err != nil {
				c.JSON(http.StatusOK, gin.H{
					"details":       err,
					"message":       "Payment initiation successful",
					"code":          "1996",
					"transactionId": req.ExternalID,
					"secureId":      secureID,
				})
				return
			}

			// Create transaction record
			newTransaction := &transactions.TransactionModel{
				ImpalaMerchantID:    req.ImpalaMerchantId,
				MerchantRequestID:   secureID,
				CheckoutRequestID:   secureID,
				ResponseDescription: "Payment initiated via Cameroon Collect",
				ResponseCode:        "0",
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
				"message":       "Withdrawal initiation successful",
				"code":          "1997",
				"transactionId": req.ExternalID,
				"secureId":      secureID,
			})
		}

	}
}

// west africa callback handler

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

	// Prevent duplicate externalId for this merchant (if provided)
	if req.ExternalID != "" {
		if exists, err := transactions.ExternalIDExists(req.ImpalaMerchantId, req.ExternalID); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to validate externalId", "details": err.Error()})
			return
		} else if exists {
			c.JSON(http.StatusConflict, gin.H{
				"error":   "DUPLICATE_EXTERNAL_ID",
				"message": "A transaction with this externalId already exists for this merchant",
			})
			return
		}
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
	data := fmt.Sprintf("amount=%.2f&merchant=%s&callback=%s&redirect=%s&externalid=%s&redirectUrl=%s&currency=%s", req.Amount, req.ImpalaMerchantId, req.CallbackURL, secureID, req.ExternalID, req.RedirectURL, req.Currency)
	fmt.Println(data)

	// Encode the string in Base64
	encoded := base64.StdEncoding.EncodeToString([]byte(data))

	// Print the Base64 encoded string
	fmt.Println("Base64 Encoded Data:", encoded)
	fmt.Println("merchant id ", req.ImpalaMerchantId)
	var cardlink string

	if req.ImpalaMerchantId == "Tallytours" || req.ImpalaMerchantId == "plugin" { //kcb mid
		cardlink = "https://process.mam-laka.com/mpgs.php?data=" + encoded

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

	// Prevent duplicate externalId for this merchant (if provided)
	if req.ExternalID != "" {
		if exists, err := transactions.ExternalIDExists(req.ImpalaMerchantId, req.ExternalID); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to validate externalId", "details": err.Error()})
			return
		} else if exists {
			c.JSON(http.StatusConflict, gin.H{
				"error":   "DUPLICATE_EXTERNAL_ID",
				"message": "A transaction with this externalId already exists for this merchant",
			})
			return
		}
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

	// Prevent duplicate externalId for this merchant (if provided)
	if req.ExternalID != "" {
		if exists, err := transactions.ExternalIDExists(req.ImpalaMerchantId, req.ExternalID); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to validate externalId", "details": err.Error()})
			return
		} else if exists {
			c.JSON(http.StatusConflict, gin.H{
				"error":   "DUPLICATE_EXTERNAL_ID",
				"message": "A transaction with this externalId already exists for this merchant",
			})
			return
		}
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

			// Update the transaction status to COMPLETE
			if err := db.Model(&transactions.TransactionModel{}).
				Where("id = ?", transaction.ID).
				Updates(map[string]interface{}{
					"transactionStatus":   "COMPLETE",
					"callbackStatus":      "SENT",
					"responseDescription": resultDesc, // Include the success reason

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
				"report":            resultDesc,             // Include the success reason
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
				"report":            resultDesc,             // Include the success reason
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

			// Deduct merchant float NOW (after Safaricom confirms success)
			if err := balances.DeductKESBalance(transaction.ImpalaMerchantID, float64(transaction.Amount)); err != nil {
				// If deduction fails due to insufficient float, return error code 101 and do NOT mark SUCCESS
				log.Printf("Float deduction failed for merchant %s: %v", transaction.ImpalaMerchantID, err)
				c.JSON(http.StatusInternalServerError, gin.H{
					"error":   "INTERNAL_ERROR",
					"message": "Insufficient float, please try again",
					"code":    101,
				})
				return
			}

			// Update the transaction status to COMPLETE
			if err := db.Model(&transactions.TransactionModel{}).
				Where("id = ?", transaction.ID).
				Updates(map[string]interface{}{
					"transactionStatus": "COMPLETE",
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

// B2CCallbackHandler handles M-Pesa B2C (withdrawal) callback responses
func B2CCallbackHandler(c *gin.Context) {
	log.Println("📞 Received M-Pesa B2C callback")

	// Read the raw request body
	rawBody, err := c.GetRawData()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to read request body", "details": err.Error()})
		return
	}

	// Log the raw request body
	log.Println("Raw B2C callback body:", string(rawBody))

	// Parse the incoming JSON request
	var response map[string]interface{}
	if err := json.Unmarshal(rawBody, &response); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid callback body", "details": err.Error()})
		return
	}

	// Check if this is a B2C Result response
	result, ok := response["Result"].(map[string]interface{})
	if !ok {
		log.Println("⚠️ Invalid B2C callback format: missing Result")
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid callback format: missing Result"})
		return
	}

	log.Println("✅ Processing B2C withdrawal response")

	// Extract fields from the withdrawal response
	resultType, _ := result["ResultType"].(float64)
	resultCode, _ := result["ResultCode"].(float64)
	resultDesc, _ := result["ResultDesc"].(string)
	originatorConversationID, _ := result["OriginatorConversationID"].(string)
	conversationID, _ := result["ConversationID"].(string)
	transactionID, _ := result["TransactionID"].(string)

	// Debug the extracted fields
	log.Printf("B2C Callback Details:")
	log.Printf("  ResultType: %.0f", resultType)
	log.Printf("  ResultCode: %.0f", resultCode)
	log.Printf("  ResultDesc: %s", resultDesc)
	log.Printf("  OriginatorConversationID: %s", originatorConversationID)
	log.Printf("  ConversationID: %s", conversationID)
	log.Printf("  TransactionID: %s", transactionID)

	// Get database connection
	db := database.GetConnection()
	if db == nil {
		log.Println("❌ Failed to get database connection")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database connection failed"})
		return
	}

	// Find transaction by OriginatorConversationID (MerchantRequestID)
	transaction, err := transactions.GetTransactionByMerchantRequestID(originatorConversationID)
	if err != nil {
		log.Printf("❌ Transaction not found for OriginatorConversationID: %s, Error: %v", originatorConversationID, err)
		c.JSON(http.StatusNotFound, gin.H{"error": "Transaction not found", "details": err.Error()})
		return
	}

	log.Printf("✅ Found transaction: ID=%d, MerchantID=%s, Amount=%d, Currency=%s, Status=%s, CallbackStatus=%s", 
		transaction.ID, transaction.ImpalaMerchantID, transaction.Amount, transaction.Currency, 
		transaction.TransactionStatus, transaction.CallbackStatus)

	// Check if callback was already sent (prevent duplicate processing)
	if transaction.CallbackStatus == "SENT" {
		log.Printf("⚠️ Callback already sent for transaction ID=%d, skipping duplicate processing", transaction.ID)
		c.JSON(http.StatusOK, gin.H{"message": "Callback already processed"})
		return
	}

	// Extract metadata from ResultParameters
	metadata := make(map[string]interface{})
	resultParameters, ok := result["ResultParameters"].(map[string]interface{})
	if ok {
		resultParameter, ok := resultParameters["ResultParameter"].([]interface{})
		if ok {
			for _, item := range resultParameter {
				itemMap, ok := item.(map[string]interface{})
				if ok {
					key, _ := itemMap["Key"].(string)
					value := itemMap["Value"]
					metadata[key] = value
					log.Printf("  ResultParameter - Key: %s, Value: %v", key, value)
				}
			}
		}
	}

	// Process based on ResultCode
	if resultCode == 0 { // Success
		log.Println("✅ B2C withdrawal successful")

		// Step 1: Check available KES balance in merchant balance table
		merchantBalance, err := balances.GetMerchantBalance(transaction.ImpalaMerchantID)
		if err != nil {
			log.Printf("❌ Failed to retrieve merchant balance: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve merchant balance", "details": err.Error()})
			return
		}

		log.Printf("💰 Merchant KES Balance: %.2f, Required: %.2f", merchantBalance.KESBalance, float64(transaction.Amount))

		// Store original transaction status for balance deduction check
		originalStatus := transaction.TransactionStatus

		// Step 2: Update transaction status to COMPLETE (before sending callback)
		updates := map[string]interface{}{
			"transactionStatus":   "COMPLETE",
			"responseDescription": resultDesc,
			// Don't set callbackStatus to SENT yet - we'll do that after callback is sent
		}

		if err := db.Model(&transactions.TransactionModel{}).
			Where("id = ?", transaction.ID).
			Updates(updates).Error; err != nil {
			log.Printf("❌ Failed to update transaction status: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update transaction", "details": err.Error()})
			return
		}

		// Step 3: Get transaction amount from metadata or use transaction amount
		amount := transaction.Amount
		if transactionAmount, ok := metadata["TransactionAmount"].(float64); ok {
			amount = int(transactionAmount)
		}

		// Step 4: Prepare and send callback to merchant
		callbackResponse := map[string]interface{}{
			"amount":            amount,
			"currency":          transaction.Currency,
			"externalId":        transaction.ExternalID,
			"netAmount":         amount,
			"secureId":          transaction.SecureID,
			"transactionReport": resultDesc, // Use ResultDesc as transactionReport
			"transactionStatus": "COMPLETE",
		}

		log.Printf("📤 Sending callback to merchant: %+v", callbackResponse)

		// Send callback to merchant
		callbackErr := SendCallback(transaction.ID, callbackResponse)
		if callbackErr != nil {
			log.Printf("⚠️ Failed to send callback to merchant: %v", callbackErr)
			// Update callback status to FAILED if callback fails
			db.Model(&transactions.TransactionModel{}).
				Where("id = ?", transaction.ID).
				Update("callbackStatus", "FAILED")
		} else {
			// Step 5: Mark callback as SENT after successful callback
			db.Model(&transactions.TransactionModel{}).
				Where("id = ?", transaction.ID).
				Update("callbackStatus", "SENT")
			log.Printf("✅ Callback sent successfully to merchant")
		}

		// Step 6: Deduct balance AFTER callback is sent (only if transaction was PENDING)
		if originalStatus == "PENDING" || originalStatus == "" {
			if merchantBalance.KESBalance >= float64(transaction.Amount) {
				if err := balances.DeductKESBalance(transaction.ImpalaMerchantID, float64(transaction.Amount)); err != nil {
					log.Printf("❌ Balance deduction failed for merchant %s: %v", transaction.ImpalaMerchantID, err)
					// Log error but don't fail - Safaricom already processed the payment
				} else {
					log.Printf("✅ Successfully deducted balance for merchant %s: %.2f KES", transaction.ImpalaMerchantID, float64(transaction.Amount))
				}
			} else {
				log.Printf("⚠️ Insufficient balance for merchant %s: available %.2f, required %.2f (Safaricom already processed payment)", 
					transaction.ImpalaMerchantID, merchantBalance.KESBalance, float64(transaction.Amount))
				// Still process the callback since Safaricom already processed it
			}
		} else {
			log.Printf("⚠️ Transaction already processed (status: %s), skipping balance deduction", originalStatus)
		}

		log.Printf("✅ Successfully processed B2C callback for transaction: %s", originatorConversationID)
		c.JSON(http.StatusOK, gin.H{"message": "B2C callback processed successfully"})

	} else { // Failure
		log.Printf("❌ B2C withdrawal failed - ResultCode: %.0f, ResultDesc: %s", resultCode, resultDesc)

		// Step 1: Update transaction status to FAILED (before sending callback)
		updates := map[string]interface{}{
			"transactionStatus":   "FAILED",
			"responseDescription": resultDesc,
			// Don't set callbackStatus to SENT yet - we'll do that after callback is sent
		}

		if err := db.Model(&transactions.TransactionModel{}).
			Where("id = ?", transaction.ID).
			Updates(updates).Error; err != nil {
			log.Printf("❌ Failed to update transaction status: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update transaction", "details": err.Error()})
			return
		}

		// Step 2: Get transaction amount
		amount := transaction.Amount
		if transactionAmount, ok := metadata["TransactionAmount"].(float64); ok {
			amount = int(transactionAmount)
		}

		// Step 3: Prepare and send callback to merchant
		callbackResponse := map[string]interface{}{
			"amount":            amount,
			"currency":          transaction.Currency,
			"externalId":        transaction.ExternalID,
			"netAmount":         amount,
			"secureId":          transaction.SecureID,
			"transactionReport": resultDesc, // Use ResultDesc as transactionReport
			"transactionStatus": "FAILED",
		}

		log.Printf("📤 Sending failed callback to merchant: %+v", callbackResponse)

		// Send callback to merchant
		callbackErr := SendCallback(transaction.ID, callbackResponse)
		if callbackErr != nil {
			log.Printf("⚠️ Failed to send callback to merchant: %v", callbackErr)
			// Update callback status to FAILED if callback fails
			db.Model(&transactions.TransactionModel{}).
				Where("id = ?", transaction.ID).
				Update("callbackStatus", "FAILED")
		} else {
			// Mark callback as SENT after successful callback
			db.Model(&transactions.TransactionModel{}).
				Where("id = ?", transaction.ID).
				Update("callbackStatus", "SENT")
			log.Printf("✅ Callback sent successfully to merchant")
		}

		// For failed transactions, no balance deduction needed (payment wasn't processed)

		log.Printf("✅ Successfully processed failed B2C callback for transaction: %s", originatorConversationID)
		c.JSON(http.StatusOK, gin.H{"message": "B2C callback processed successfully"})
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
	data := fmt.Sprintf("amount=%.2f&merchant=%s&callback=%s&redirect=%s&externalid=%s&currency=%s", Tag.Amount, Tag.Tag, callbackUrl, secureID, externalId, Tag.Currency)
	// fmt.Println(data)

	// Encode the string in Base64
	encoded := base64.StdEncoding.EncodeToString([]byte(data))

	// Print the Base64 encoded string
	var cardlink string
	if Tag.Tag == "Tallytours" || Tag.Tag == "plugin" { //kcb mid
		cardlink = "https://process.mam-laka.com/mpgs.php?data=" + encoded

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

// SearchTransactionsHandler handles transaction search with multiple filters
func SearchTransactionsHandler(c *gin.Context) {
	// Get merchant ID from context (if authenticated) or from query
	merchantID, merchantExists := c.Get("merchantID")
	merchantIDStr := ""
	if merchantExists {
		if str, ok := merchantID.(string); ok {
			merchantIDStr = str
		}
	}
	// Also allow merchant ID from query parameter
	if merchantIDStr == "" {
		merchantIDStr = c.Query("merchantId")
	}

	// Build search parameters
	params := transactions.SearchTransactionsParams{
		MerchantID: merchantIDStr,
	}

	// Extract optional search parameters
	if phone := c.Query("phone"); phone != "" {
		params.Phone = phone
	}

	if amountStr := c.Query("amount"); amountStr != "" {
		if amount, err := strconv.Atoi(amountStr); err == nil {
			params.Amount = &amount
		}
	}

	if currency := c.Query("currency"); currency != "" {
		params.Currency = currency
	}

	if status := c.Query("status"); status != "" {
		params.TransactionStatus = status
	}

	if report := c.Query("report"); report != "" {
		params.TransactionReport = report
	}

	if externalID := c.Query("externalId"); externalID != "" {
		params.ExternalID = externalID
	}

	if secureID := c.Query("secureId"); secureID != "" {
		params.SecureID = secureID
	}

	if sourceOfFunds := c.Query("sourceOfFunds"); sourceOfFunds != "" {
		params.SourceOfFunds = sourceOfFunds
	}

	// Date range filters
	if startDateStr := c.Query("startDate"); startDateStr != "" {
		if startDate, err := strconv.ParseInt(startDateStr, 10, 64); err == nil {
			params.StartDate = &startDate
		}
	}

	if endDateStr := c.Query("endDate"); endDateStr != "" {
		if endDate, err := strconv.ParseInt(endDateStr, 10, 64); err == nil {
			params.EndDate = &endDate
		}
	}

	// Pagination
	page := 1
	if pageParam := c.DefaultQuery("page", "1"); pageParam != "" {
		if parsedPage, err := strconv.Atoi(pageParam); err == nil && parsedPage > 0 {
			page = parsedPage
		}
	}
	params.Page = page

	pageSize := 20
	if pageSizeParam := c.DefaultQuery("page_size", "20"); pageSizeParam != "" {
		if parsedPageSize, err := strconv.Atoi(pageSizeParam); err == nil && parsedPageSize > 0 {
			if parsedPageSize > 100 {
				pageSize = 100
			} else {
				pageSize = parsedPageSize
			}
		}
	}
	params.PageSize = pageSize

	// Perform search
	results, total, err := transactions.SearchTransactions(params)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to search transactions", "details": err.Error()})
		return
	}

	// Serialize results
	serializer := transactions.NewTransactionSerializerList(c, results)

	totalPages := 0
	if total > 0 {
		totalPages = int((total + int64(pageSize) - 1) / int64(pageSize))
	}

	c.JSON(http.StatusOK, gin.H{
		"transactions": serializer.Response(),
		"pagination": gin.H{
			"current_page": page,
			"per_page":     pageSize,
			"total_pages":  totalPages,
			"total_items":  total,
		},
	})
}

func ListTransactionsHandler(c *gin.Context) {
	merchantID, exists := c.Get("merchantID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Missing authentication details"})
		return
	}

	merchantIDStr, ok := merchantID.(string)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid merchantID type"})
		return
	}

	page := 1
	if pageParam := c.DefaultQuery("page", "1"); pageParam != "" {
		if parsedPage, err := strconv.Atoi(pageParam); err == nil && parsedPage > 0 {
			page = parsedPage
		}
	}

	pageSize := 20
	if pageSizeParam := c.DefaultQuery("page_size", "20"); pageSizeParam != "" {
		if parsedPageSize, err := strconv.Atoi(pageSizeParam); err == nil && parsedPageSize > 0 {
			if parsedPageSize > 100 {
				pageSize = 100
			} else {
				pageSize = parsedPageSize
			}
		}
	}

	transactionList, total, err := transactions.GetTransactionsPaginated(merchantIDStr, page, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve transactions", "details": err.Error()})
		return
	}

	totalPages := 0
	if total > 0 {
		totalPages = int((total + int64(pageSize) - 1) / int64(pageSize))
	}

	serializer := transactions.NewTransactionSerializerList(c, transactionList)

	c.JSON(http.StatusOK, gin.H{
		"transactions": serializer.Response(),
		"pagination": gin.H{
			"current_page": page,
			"per_page":     pageSize,
			"total_pages":  totalPages,
			"total_items":  total,
		},
	})
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
		transactionStatus = "COMPLETE"

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
			"message": "Missing authentication details 2",
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

// balance routes
// get balance from base currency
func GetTotalBalanceHandler(c *gin.Context) {
	// Extract merchant ID and base currency from query parameters
	// baseCurrency, baseCurrencyExists := c.Get("baseCurrency")
	merchantID, merchantExists := c.Get("merchantID")

	if !merchantExists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Missing authentication details"})
		return
	}

	// Validate input
	// if merchantID == "" || baseCurrency == "" {
	// 	c.JSON(http.StatusBadRequest, gin.H{"error": "Missing required parameters"})
	// 	return
	// }

	// Retrieve the total balance
	totalBalance, err := balances.GetTotalBalance(merchantID.(string), "KES")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve balance", "details": err.Error()})
		return
	}

	// Return response
	c.JSON(http.StatusOK, gin.H{
		"merchantId":   merchantID,
		"baseCurrency": "KES",
		"Balances":     totalBalance,
	})
}

// get payout balance
func GetTotalPayinBalanceHandler(c *gin.Context) {
	// Extract merchant ID and base currency from query parameters
	// baseCurrency, baseCurrencyExists := c.Get("baseCurrency")
	merchantID, merchantExists := c.Get("merchantID")
	fmt.Println("merchant id: ", merchantID)

	if !merchantExists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Missing authentication details"})
		return
	}

	// Validate input
	// if merchantID == "" || baseCurrency == "" {
	// 	c.JSON(http.StatusBadRequest, gin.H{"error": "Missing required parameters"})
	// 	return
	// }
	fmt.Println("merchantID", merchantID)

	// Retrieve the total balance
	totalBalance, err := balances.GetTotalCollectionBalance(merchantID.(string), "KES")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve balance", "details": err.Error()})
		return
	}

	// Return response
	c.JSON(http.StatusOK, gin.H{
		"merchantId":   merchantID,
		"baseCurrency": "KES",
		"Balances":     totalBalance,
	})
}

// card routes

// CreateCardHolder creates a new card holder
func CreateCardHolderHandler(c *gin.Context) {
	merchantID, merchantExists := c.Get("merchantID")
	if !merchantExists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Missing authentication details"})
		return
	}

	var req virtualcards.CreateHolderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body", "details": err.Error()})
		return
	}

	req.MerchantOrderNo = fmt.Sprintf("HOLDER-%s-%d", merchantID, time.Now().Unix())

	// Call the simple request function
	resp, err := virtualcards.SimpleCreateCardHolder("https://kcb-buni.mam-laka.com", req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "API request failed", "details": err.Error()})
		return
	}

	// Check nested response for failure
	if resp == nil || !resp.Data.Success {
		c.JSON(http.StatusBadRequest, gin.H{
			"success":  false,
			"message":  "Failed to create card holder",
			"api_msg":  resp.Data.Msg,
			"api_code": resp.Data.Code,
		})
		return
	}

	// Save to DB (example structure, adjust as needed)
	holderData := resp.Data.Data
	dbHolder := virtualcards.CardHolderModel{
		MerchantOrderNo: holderData.MerchantOrderNo,
		HolderID:        fmt.Sprintf("%d", holderData.HolderID),
		CardTypeID:      strconv.Itoa(req.CardTypeID),
		AreaCode:        req.AreaCode,
		Mobile:          req.Mobile,
		Email:           req.Email,
		FirstName:       req.FirstName,
		LastName:        req.LastName,
		BirthDay:        req.BirthDay,
		Country:         req.Country,
		Town:            req.Town,
		Address:         req.Address,
		PostCode:        req.PostCode,
		Status:          holderData.Status,
		StatusStr:       holderData.StatusStr,
		Message:         holderData.Message,
		UserID:          404, // Placeholder
		MerchantID:      merchantID.(string),
	}

	savedHolder, err := virtualcards.CreateCardHolderWithUser(404, dbHolder)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save holder"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    savedHolder,
	})
}

// get all card holders
// GetCardHoldersHandler handles fetching card holders for the authenticated merchant
func GetCardHoldersHandler(c *gin.Context) {
	// Retrieve merchant ID from context (set by authentication middleware)
	merchantID, exists := c.Get("merchantID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized: merchant ID not found in token"})
		return
	}

	// Convert merchantID to uint if needed
	// merchantIDUint, ok := merchantID.(uint)
	// if !ok {
	// 	// In case merchantID is a string and needs conversion
	// 	if strID, ok := merchantID.(string); ok {
	// 		parsedID, err := strconv.ParseUint(strID, 10, 64)
	// 		if err != nil {
	// 			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid merchant ID format"})
	// 			return
	// 		}
	// 		merchantIDUint = uint(parsedID)
	// 	} else {
	// 		c.JSON(http.StatusBadRequest, gin.H{"error": "Unable to interpret merchant ID"})
	// 		return
	// 	}
	// }

	// Fetch card holders for this merchant
	merchantIDStr, ok := merchantID.(string)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid merchant ID format"})
		return
	}
	holders, err := virtualcards.GetCardHoldersByMerchantID(merchantIDStr)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve card holders", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "data": holders})
}

// create a virtual card
// CreateCardHandler creates a new virtual card and stores it in the DB
func CreateCardHandler(c *gin.Context) {
	var req virtualcards.CreateCardRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body", "details": err.Error()})
		return
	}

	// userID := c.GetUint("user_id")
	merchantID, _ := c.Get("merchantID")

	// Optional: Validate card holder belongs to this merchant
	holder, err := virtualcards.GetCardHolderByHolderID(req.HolderID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Card holder not found"})
		return
	}
	if holder.MerchantID != merchantID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied to holder"})
		return
	}

	// Call external API to create card
	baseURL := "https://kcb-buni.mam-laka.com"
	apiResp, err := virtualcards.CallCreateCard(baseURL, req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "Failed to create card",
			"error":   err.Error(),
		})
		return
	}

	if !apiResp.Data.Success {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": apiResp.Data.Msg,
			"code":    apiResp.Data.Code,
		})
		return
	}

	// Take the first card response in the data array
	if len(apiResp.Data.Data) == 0 {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "API response missing card data"})
		return
	}
	card := apiResp.Data.Data[0]

	// Build DB record
	dbCard := virtualcards.VirtualCardModel{
		OrderNo:          card.OrderNo,
		MerchantOrderNo:  card.MerchantOrderNo,
		CardTypeID:       req.CardTypeID,
		HolderID:         req.HolderID,
		CardNo:           card.OrderNo, // Placeholder
		Currency:         card.Currency,
		Amount:           card.Amount,
		Fee:              card.Fee,
		ReceivedAmount:   card.ReceivedAmount,
		ReceivedCurrency: card.ReceivedCurrency,
		Type:             card.Type,
		Status:           card.Status,
		TransactionTime:  card.TransactionTime,
		Balance:          10,
		UserID:           404,
		CardHolderID:     holder.ID,
		MerchantID:       merchantID.(string),
		CallbackURL:      req.CallbackUrl,
		CallbackStatus:   "PENDING",
	}

	// Save card
	savedCard, err := virtualcards.CreateVirtualCardWithTransaction(404, dbCard)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save card to database", "details": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"message": "Virtual card created successfully",
		"data":    savedCard,
	})
}

// get virtual card by merchant id
func ListVirtualCardsByMerchant(c *gin.Context) {
	merchantID, exists := c.Get("merchantID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized: merchant ID not found in token"})
		return
	}

	// if err != nil {
	// 	c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
	// 	return
	// }

	merchantIDStr, ok := merchantID.(string)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid merchant ID format"})
		return
	}
	cards, err := virtualcards.GetVirtualCardsByMerchantID(merchantIDStr)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve cards", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    cards,
	})
}

// card callback handler
func GetVirtualCardByIDCardCallbackHandler(c *gin.Context) {
	var callback map[string]string
	if err := c.ShouldBindJSON(&callback); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request", "details": err.Error()})
		return
	}

	card, err := virtualcards.UpdateVirtualCardFromCallback(callback)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update card", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Card updated successfully",
		"data":    card,
	})
}

// get the card details handler
func RetrieveCardInfoHandler(c *gin.Context) {
	var req virtualcards.CardInfoRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body", "details": err.Error()})
		return
	}

	// userID := c.GetUint("user_id")
	merchantID, merchantExists := c.Get("merchantID")
	if !merchantExists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Missing authentication details"})
		return
	}
	fmt.Println("merchantID", merchantID)

	// Optional: Validate card holder belongs to this merchant

	// Call external API to create card
	baseURL := "https://kcb-buni.mam-laka.com"
	apiResp, err := virtualcards.CallGetCardInfo(baseURL, req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "Failed to create card",
			"error":   err.Error(),
		})
		return
	}

	if !apiResp.Data.Success {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": apiResp.Data.Msg,
			"code":    apiResp.Data.Code,
		})
		return
	}

	// Take the first card response in the data array
	// if len(apiResp.Data.Data) == 0 {
	// 	c.JSON(http.StatusInternalServerError, gin.H{"error": "API response missing card data"})
	// 	return
	// }
	// card := apiResp.Data.Data[0]

	// // Build DB record
	// dbCard := virtualcards.VirtualCardModel{
	// 	OrderNo:          card.OrderNo,
	// 	MerchantOrderNo:  card.MerchantOrderNo,
	// 	CardTypeID:       req.CardTypeID,
	// 	HolderID:         req.HolderID,
	// 	CardNo:           card.OrderNo, // Placeholder
	// 	Currency:         card.Currency,
	// 	Amount:           card.Amount,
	// 	Fee:              card.Fee,
	// 	ReceivedAmount:   card.ReceivedAmount,
	// 	ReceivedCurrency: card.ReceivedCurrency,
	// 	Type:             card.Type,
	// 	Status:           card.Status,
	// 	TransactionTime:  card.TransactionTime,
	// 	Balance:          10,
	// 	UserID:           404,
	// 	CardHolderID:     holder.ID,
	// 	MerchantID:       merchantID.(string),
	// 	CallbackURL:      req.CallbackUrl,
	// 	CallbackStatus:   "PENDING",
	// }

	// // Save card
	// savedCard, err := virtualcards.CreateVirtualCardWithTransaction(404, dbCard)
	// if err != nil {
	// 	c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save card to database", "details": err.Error()})
	// 	return
	// }

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"message": "Virtual card Retrieved successfully",
		"data":    apiResp.Data.Data,
	})
}

// get card balance
func GetCardBalance(c *gin.Context) {
	var req virtualcards.CardBalanceRequest
	if err := c.ShouldBindJSON(&req); err != nil || req.CardNo == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "cardNo is required"})
		return
	}

	baseURL := "https://kcb-buni.mam-laka.com"
	apiResp, err := virtualcards.CallCardBalanceAPI(baseURL, req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve card balance", "details": err.Error()})
		return
	}

	// check if nested "data.success" is false
	if !apiResp.Data.Success {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"message": apiResp.Data.Msg,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"balance": apiResp.Data.Data,
	})
}

// recharge api
func RechargeCardHandler(c *gin.Context) {
	var req virtualcards.CardRechargeRequest
	if err := c.ShouldBindJSON(&req); err != nil || req.CardNo == "" || req.Amount <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "cardNo and valid amount are required"})
		return
	}

	baseURL := "https://kcb-buni.mam-laka.com"
	apiResp, err := virtualcards.CallCardRechargeAPI(baseURL, req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to recharge card", "details": err.Error()})
		return
	}

	if !apiResp.Data.Success {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": apiResp.Data.Msg,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    apiResp.Data.Data,
	})
}

//camerooncallback

// CallbackPayload defines the structure of the callback data
type CallbackPayload struct {
	Message                     string  `json:"message"`
	Status                      int     `json:"status"`
	SenderCountry               string  `json:"senderCountry"`
	SenderName                  string  `json:"senderName"`
	CollectionStatus            string  `json:"collectionStatus"`
	CollectionType              string  `json:"collectionType"`
	PayoutType                  string  `json:"payoutType"`
	BeneficiaryCountry          string  `json:"beneficiaryCountry"`
	BeneficiaryCurrency         string  `json:"beneficiaryCurrency"`
	Reference                   string  `json:"reference"`
	TransactionDate             string  `json:"transactionDate"`
	BeneficiaryName             string  `json:"beneficiaryName"`
	PayoutAmount                float64 `json:"payoutAmount"`
	FeeChargedToPartnerAmount   float64 `json:"feeChargedToPartnerAmount,omitempty"`
	FeeChargedToPartnerCurrency string  `json:"feeChargedToPartnerCurrency,omitempty"`
}

// HandleCallback processes payment callbacks using gin.Context
func CameroonXAFCallback(c *gin.Context) {
	var payload CallbackPayload
	var transaction *transactions.TransactionModel
	var callbackResponse map[string]interface{}

	if err := c.ShouldBindJSON(&payload); err != nil {
		log.Println(" Invalid JSON payload:", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid JSON payload"})
		return
	}

	db := database.GetConnection()
	log.Println(" Getting database connection")

	// Get transaction before the switch
	txn, err := transactions.GetTransactionByMerchantRequestID(payload.Reference)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Transaction not found", "details": err.Error()})
		return
	}
	transaction = &txn

	switch payload.CollectionStatus {
	case "COMPLETED":
		fmt.Println("✔️ Collection COMPLETED for reference:", payload.Reference)

		// Update transaction status
		if err := db.Model(&transactions.TransactionModel{}).
			Where("id = ?", transaction.ID).
			Updates(map[string]interface{}{
				"transactionStatus": "SUCCESS",
				"callbackStatus":    "SENT",
			}).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update transaction", "details": err.Error()})
			return
		}

		// Update balance
		if updateErr := balances.AddXAFBalance(transaction.ImpalaMerchantID, payload.PayoutAmount); updateErr != nil {
			log.Println("Error updating XAF balance:", updateErr)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update merchant balance", "details": updateErr.Error()})
			return
		}

		// Fetch updated balance (optional logging/debug)
		var updatedBalance balances.MerchantCollectionBalance
		if err := db.Where("impalaMerchantId = ?", transaction.ImpalaMerchantID).
			First(&updatedBalance).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch updated balance", "details": err.Error()})
			return
		}
		log.Printf(" Updated XAF balance for %s: %.2f", transaction.ImpalaMerchantID, updatedBalance.XAFBalance)

		// Prepare callback response
		callbackResponse = map[string]interface{}{
			"transactionStatus": "COMPLETE",
			"transactionReport": "COMPLETE",
			"currency":          "XAF",
			"amount":            payload.PayoutAmount,
			"netAmount":         payload.PayoutAmount,
			"secureId":          transaction.SecureID,
			"externalId":        transaction.ExternalID,
		}

	case "FAILED":
		fmt.Println("Collection FAILED for reference:", payload.Reference)

		if err := db.Model(&transactions.TransactionModel{}).
			Where("id = ?", transaction.ID).
			Updates(map[string]interface{}{
				"transactionStatus": "FAILED",
				"callbackStatus":    "SENT",
			}).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update transaction", "details": err.Error()})
			return
		}

		callbackResponse = map[string]interface{}{
			"transactionStatus": "FAILED",
			"transactionReport": "FAILED",
			"currency":          "XAF",
			"amount":            payload.PayoutAmount,
			"netAmount":         payload.PayoutAmount,
			"secureId":          transaction.SecureID,
			"externalId":        transaction.ExternalID,
		}

	default:
		log.Printf(" Unknown Collection Status: '%s' for Reference: %s", payload.CollectionStatus, payload.Reference)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Unknown collection status"})
		return
	}

	// Send the callback to the client
	if err := SendCallback(transaction.ID, callbackResponse); err != nil {
		log.Println(" Failed to send callback00000:", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to send callback", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "received"})
}

// DisburseCallbackPayload defines the payload for a disbursement callback
type DisburseCallbackPayload struct {
	Message                     string  `json:"message"`
	Status                      int     `json:"status"`
	SenderCountry               string  `json:"senderCountry"`
	SenderName                  string  `json:"senderName"`
	PayoutType                  string  `json:"payoutType"`
	PayoutStatus                string  `json:"payoutStatus"`
	BeneficiaryCountry          string  `json:"beneficiaryCountry"`
	BeneficiaryCurrency         string  `json:"beneficiaryCurrency"`
	Reference                   string  `json:"reference"`
	PayoutRef                   string  `json:"payoutRef"`
	TransactionDate             string  `json:"transactionDate"`
	BeneficiaryName             string  `json:"beneficiaryName"`
	PayoutAmount                float64 `json:"payoutAmount"`
	FeeChargedToPartnerAmount   float64 `json:"feeChargedToPartnerAmount"`
	FeeChargedToPartnerCurrency string  `json:"feeChargedToPartnerCurrency"`
}

// HandleDisburseCallback processes the disbursement callback
func CameroonXAFDisburseCallback(c *gin.Context) {
	var payload DisburseCallbackPayload
	var transaction *transactions.TransactionModel
	var callbackResponse map[string]interface{}

	db := database.GetConnection()
	log.Println("Getting database connection")

	if err := c.ShouldBindJSON(&payload); err != nil {
		log.Println("Invalid disbursement callback payload:", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid JSON"})
		return
	}

	// Fetch transaction early (needed in both cases)
	txn, err := transactions.GetTransactionByMerchantRequestID(payload.Reference)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Transaction not found", "details": err.Error()})
		return
	}
	transaction = &txn

	switch payload.PayoutStatus {
	case "COMPLETED":
		// Update the transaction status
		if err := db.Model(&transactions.TransactionModel{}).
			Where("id = ?", transaction.ID).
			Updates(map[string]interface{}{
				"transactionStatus": "SUCCESS",
				"callbackStatus":    "SENT",
			}).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update transaction", "details": err.Error()})
			return
		}

		// Deduct balance
		if err := db.Model(&balances.MerchantBalance{}).
			Where("impalaMerchantId = ?", transaction.ImpalaMerchantID).
			Update("xafBalance", gorm.Expr("xafBalance - ?", transaction.Amount)).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update merchant balance", "details": err.Error()})
			return
		}

		callbackResponse = map[string]interface{}{
			"transactionStatus": "COMPLETE",
			"transactionReport": "COMPLETE",
			"currency":          "XAF",
			"amount":            payload.PayoutAmount,
			"netAmount":         payload.PayoutAmount,
			"secureId":          transaction.SecureID,
			"externalId":        transaction.ExternalID,
		}

	case "FAILED":
		// Update the transaction status
		if err := db.Model(&transactions.TransactionModel{}).
			Where("id = ?", transaction.ID).
			Updates(map[string]interface{}{
				"transactionStatus": "FAILED",
				"callbackStatus":    "SENT",
			}).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update transaction", "details": err.Error()})
			return
		}

		callbackResponse = map[string]interface{}{
			"transactionStatus": "FAILED",
			"transactionReport": "FAILED",
			"currency":          "XAF",
			"amount":            payload.PayoutAmount,
			"netAmount":         payload.PayoutAmount,
			"secureId":          transaction.SecureID,
			"externalId":        transaction.ExternalID,
		}

	default:
		log.Printf("⚠️ Unknown Status '%s' for Reference: %s", payload.PayoutStatus, payload.Reference)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Unknown payout status"})
		return
	}

	// Now Send the Callback
	if err := SendCallback(transaction.ID, callbackResponse); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to send callback", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "received"})
}

// payment request
type GlobpayCardPaymentRequest struct {
	FirstName       string  `json:"first_name" binding:"required"`
	LastName        string  `json:"last_name" binding:"required"`
	Email           string  `json:"email" binding:"required,email"`
	Address         string  `json:"address" binding:"required"`
	Country         string  `json:"country" binding:"required"`
	City            string  `json:"city" binding:"required"`
	State           string  `json:"state" binding:"required"`
	Zip             string  `json:"zip" binding:"required"`
	IPAddress       string  `json:"ip_address" binding:"required,ip"`
	PhoneNumber     string  `json:"phone_number" binding:"required"`
	Amount          float64 `json:"amount" binding:"required"`
	Currency        string  `json:"currency" binding:"required"`
	CardNumber      string  `json:"card_number" binding:"required"`
	CardExpiryMonth string  `json:"card_expiry_month" binding:"required"`
	CardExpiryYear  string  `json:"card_expiry_year" binding:"required"`
	CardCVV         string  `json:"card_cvv" binding:"required"`
	RedirectURL     string  `json:"redirect_url" binding:"required,url"`
	WebhookURL      string  `json:"webhook_url" binding:"required,url"`
	OrderID         string  `json:"order_id" binding:"required"`
}

// globpay handlers

// Helper to get message
func msgOr(m map[string]interface{}, key string) string {
	if val, ok := m[key].(string); ok {
		return val
	}
	return ""
}

// globpay card callback
func GlobpayCardCallbackHandler(c *gin.Context) {
	var callbackBody struct {
		Status  string `json:"status"` // "SUCCESS" or "FAILED"
		Message string `json:"message"`
		Data    struct {
			Amount        float64 `json:"amount"`
			Currency      string  `json:"currency"`
			OrderID       string  `json:"order_id"`
			TransactionID string  `json:"transaction_id"`
			Customer      struct {
				FirstName string `json:"first_name"`
				LastName  string `json:"last_name"`
				Email     string `json:"email"`
			} `json:"customer"`
			Refund struct {
				Status     bool   `json:"status"`
				RefundDate string `json:"refund_date"`
			} `json:"refund"`
			Chargeback struct {
				Status         bool   `json:"status"`
				ChargebackDate string `json:"chargeback_date"`
			} `json:"chargeback"`
		} `json:"data"`
	}

	if err := c.ShouldBindJSON(&callbackBody); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid callback body", "details": err.Error()})
		return
	}

	log.Printf("Received callback: %+v", callbackBody)

	// Lookup transaction using transaction_id
	db := database.GetConnection()
	var transaction transactions.TransactionModel
	if err := db.Where("merchantRequestID = ?", callbackBody.Data.TransactionID).First(&transaction).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Transaction not found", "details": err.Error()})
		return
	}

	// Determine transaction status
	transactionStatus := "FAILED"
	if callbackBody.Status == "SUCCESS" {
		transactionStatus = "COMPLETE"
	}

	// Update the transaction
	if err := db.Model(&transactions.TransactionModel{}).
		Where("id = ?", transaction.ID).
		Updates(map[string]interface{}{
			"transactionStatus":   transactionStatus,
			"responseDescription": callbackBody.Message,
			"callbackStatus":      "SENT",
		}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update transaction", "details": err.Error()})
		return
	}

	// Handle successful payment: update merchant balance
	if callbackBody.Status == "SUCCESS" {
		log.Println("Transaction successful. Updating merchant balance for:", transaction.ImpalaMerchantID)

		var balance balances.MerchantBalance
		if err := db.Where("impalaMerchantId = ?", transaction.ImpalaMerchantID).First(&balance).Error; err != nil {
			log.Println("Error retrieving merchant balance:", err)
			c.JSON(http.StatusNotFound, gin.H{"error": "Merchant balance not found", "details": err.Error()})
			return
		}

		// Update correct currency
		updateData := make(map[string]interface{})
		switch callbackBody.Data.Currency {
		case "KES":
			updateData["kesBalance"] = balance.KESBalance + callbackBody.Data.Amount
		case "USD":
			updateData["usdBalance"] = balance.USDBalance + callbackBody.Data.Amount
		case "USDC":
			updateData["usdcBalance"] = balance.USDCBalance + callbackBody.Data.Amount
		case "IMPA":
			updateData["impaBalance"] = balance.ImpaBalance + callbackBody.Data.Amount
		case "LUMEN":
			updateData["lumenBalance"] = balance.LumenBalance + callbackBody.Data.Amount
		case "USDT":
			updateData["usdtBalance"] = balance.USDTBalance + callbackBody.Data.Amount
		case "EUR":
			updateData["eurBalance"] = balance.EURBalance + callbackBody.Data.Amount
		case "GBP":
			updateData["gbpBalance"] = balance.GBPBalance + callbackBody.Data.Amount
		case "TZS":
			updateData["tzsBalance"] = balance.TZSBalance + callbackBody.Data.Amount
		case "UGX":
			updateData["ugxBalance"] = balance.UGXBalance + callbackBody.Data.Amount
		default:
			log.Println("Unsupported currency:", callbackBody.Data.Currency)
			c.JSON(http.StatusBadRequest, gin.H{"error": "Unsupported currency"})
			return
		}

		// Save balance update
		if err := db.Model(&balances.MerchantBalance{}).
			Where("impalaMerchantId = ?", transaction.ImpalaMerchantID).
			Updates(updateData).Error; err != nil {
			log.Println("Failed to update merchant balance:", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update merchant balance", "details": err.Error()})
			return
		}

		log.Println("Merchant balance updated successfully")

		// Initiate token transfer
		go sendTokenTransfer(strconv.Itoa(transaction.Amount), "GBR7COBB5T5WPEYI7PN2XXLIYC2VUE22VIF4BHRKA3TPLUSZS6TQY7BE")
	}

	// Notify the merchant
	if err := SendCallback(transaction.ID, callbackBody); err != nil {
		log.Println("Callback notification failed:", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to send callback", "details": err.Error()})
		return
	}

	// Success response
	c.JSON(http.StatusOK, gin.H{"message": "Callback processed and transaction updated"})
}

// ECitizenValidateHandler handles the eCitizen validation request
func ECitizenValidateHandler(c *gin.Context) {
	var req ECitizenValidateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Check if transaction already exists
	db := database.GetConnection()
	var existingTransaction transactions.TransactionModel
	err := db.Where("externalId = ? AND sourceOfFunds = ?", req.RefNo, "ECITIZEN").First(&existingTransaction).Error
	if err == nil {
		// Transaction already exists, return bill found
		response := ECitizenValidateResponse{
			Status: "200",
			Desc:   "Bill Found",
		}
		response.Data.Name = req.RefNo
		response.Data.Currency = req.Currency
		response.Data.Amount = fmt.Sprintf("%.2f", req.Amount)
		c.JSON(http.StatusOK, response)
		return
	}

	// Generate secure ID for the transaction
	secureID := transactions.GenerateSecureID()

	// Create transaction record with PENDING status
	newTransaction := &transactions.TransactionModel{
		ImpalaMerchantID:    "ecitizen", // Default merchant ID for eCitizen
		MerchantRequestID:   secureID,
		CheckoutRequestID:   secureID,
		ResponseDescription: "eCitizen validation initiated",
		ResponseCode:        "200",
		Currency:            req.Currency,
		Amount:              int(req.Amount),
		Msisdn:              "ecitizen",
		NetAmount:           float64(req.Amount),
		SecureID:            secureID,
		SourceOfFunds:       "ECITIZEN",
		ExternalID:          req.RefNo,
		CallbackURL:         req.CallbackURL, // Save the callback URL
		TransactionStatus:   "PENDING",
		DateAdded:           time.Now().Unix(),
	}

	// Save transaction to database
	if err := transactions.SaveTransaction(newTransaction); err != nil {
		log.Printf("Failed to save eCitizen transaction: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save transaction"})
		return
	}

	log.Printf("eCitizen transaction saved successfully with ID: %d, RefNo: %s", newTransaction.ID, req.RefNo)

	// Return successful validation response
	response := ECitizenValidateResponse{
		Status: "200",
		Desc:   "Bill Found",
	}
	response.Data.Name = req.RefNo
	response.Data.Currency = req.Currency
	response.Data.Amount = fmt.Sprintf("%.2f", req.Amount)

	c.JSON(http.StatusOK, response)
}

// ECitizenConfirmHandler handles the eCitizen confirmation request
func ECitizenConfirmHandler(c *gin.Context) {
	var req ECitizenConfirmRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Find the transaction by external ID (ref_no)
	db := database.GetConnection()
	var transaction transactions.TransactionModel
	err := db.Where("externalId = ? AND sourceOfFunds = ?", req.RefNo, "ECITIZEN").First(&transaction).Error
	if err != nil {
		log.Printf("Transaction not found for ref_no: %s, error: %v", req.RefNo, err)
		c.JSON(http.StatusNotFound, ECitizenConfirmResponse{
			Status: "404",
			Desc:   "Transaction not found",
		})
		return
	}

	log.Printf("Found eCitizen transaction with ID: %d, Status: %s", transaction.ID, transaction.TransactionStatus)

	// Update transaction with confirmation details
	updates := map[string]interface{}{
		"transactionStatus":   "SUCCESS",
		"callbackStatus":      "SENT",
		"responseDescription": fmt.Sprintf("eCitizen payment confirmed - %s", req.CustomerName),
		"merchantRequestID":   req.GatewayTransactionID,
		"checkoutRequestID":   req.GatewayTransactionID,
	}

	if err := db.Model(&transactions.TransactionModel{}).
		Where("id = ?", transaction.ID).
		Updates(updates).Error; err != nil {
		log.Printf("Failed to update transaction: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update transaction"})
		return
	}

	log.Printf("Transaction updated successfully for ID: %d", transaction.ID)

	// Send callback if callback URL is available
	if transaction.CallbackURL != "" && transaction.CallbackURL != "NULL" {
		callbackResponse := gin.H{
			"transactionStatus":     "SUCCESS",
			"transactionReport":     "collection",
			"currency":              req.Currency,
			"amount":                req.Amount,
			"netAmount":             req.Amount,
			"externalId":            req.RefNo,
			"gatewayTransactionId":  req.GatewayTransactionID,
			"customerName":          req.CustomerName,
			"customerAccountNumber": req.CustomerAccountNumber,
			"transactionDate":       req.GatewayTransactionDate,
		}

		log.Printf("Sending callback to URL: %s", transaction.CallbackURL)
		if err := SendCallback(transaction.ID, callbackResponse); err != nil {
			log.Printf("Failed to send callback for eCitizen transaction %d: %v", transaction.ID, err)
			// Don't fail the transaction if callback fails, but update callback status
			db.Model(&transactions.TransactionModel{}).
				Where("id = ?", transaction.ID).
				Update("callbackStatus", "FAILED")
		} else {
			log.Printf("Callback sent successfully for transaction %d", transaction.ID)
		}
	} else {
		log.Printf("No callback URL available for transaction %d", transaction.ID)
	}

	// Return success response
	response := ECitizenConfirmResponse{
		Status: "200",
		Desc:   "Success",
	}

	c.JSON(http.StatusOK, response)
}

// KorapayPaymentHandler handles Korapay payment initiation
func KorapayPaymentHandler(c *gin.Context) {
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

	// Parse the Korapay payment request
	var req KorapayPaymentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input"})
		return
	}

	// Prevent duplicate externalId for this merchant
	if req.ExternalID != "" {
		if exists, err := transactions.ExternalIDExists(req.ImpalaMerchantId, req.ExternalID); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to validate externalId", "details": err.Error()})
			return
		} else if exists {
			c.JSON(http.StatusConflict, gin.H{
				"error":   "DUPLICATE_EXTERNAL_ID",
				"message": "A transaction with this externalId already exists for this merchant",
			})
			return
		}
	}

	// Verify the merchant ID exists
	userID, err := users.GetUserByMerchantId(req.ImpalaMerchantId)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Merchant not found"})
		return
	}

	// Get user by ID
	user, err := users.GetUserByID(uint(userID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve user"})
		return
	}

	fmt.Printf("Korapay payment initiated for user: %s with amount: %d\n", user.Name, req.Amount)

	// Generate secure ID for reference
	secureID := korapay.GenerateSecureID()

	// Initiate Korapay payment
	korapayResponse, err := korapay.InitiateKorapayPayment(
		req.PayerPhone,
		req.CustomerName,
		req.CustomerEmail,
		req.Amount,
		req.Currency,
		req.Description,
		req.CallbackURL,
		req.RedirectURL,
	)

	if err != nil {
		log.Printf("Failed to initiate Korapay payment: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to initiate payment"})
		return
	}

	// Create transaction record
	transaction := transactions.TransactionModel{
		SecureID:          secureID,
		ExternalID:        req.ExternalID,
		ImpalaMerchantID:  req.ImpalaMerchantId,
		Amount:            req.Amount,
		Currency:          req.Currency,
		TransactionStatus: "processing",
		TransactionReport: "collection",
		SourceOfFunds:     "korapay",
		CallbackURL:       req.CallbackURL,
		RedirectURL:       req.RedirectURL,
		DateAdded:         time.Now().Unix(),
		MerchantRequestID: korapayResponse.Data.TransactionReference, // Store Korapay transaction reference
		CheckoutRequestID: korapayResponse.Data.PaymentReference,     // Store Korapay payment reference
	}

	// Save transaction to database
	if err := database.GetConnection().Create(&transaction).Error; err != nil {
		log.Printf("Failed to save transaction: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save transaction"})
		return
	}

	// Prepare response
	response := gin.H{
		"status":            korapayResponse.Status,
		"message":           korapayResponse.Message,
		"secureId":          secureID,
		"externalId":        req.ExternalID,
		"amount":            req.Amount,
		"currency":          req.Currency,
		"provider":          "korapay",
		"transactionStatus": "processing",
	}

	if korapayResponse.Data != nil {
		response["transactionReference"] = korapayResponse.Data.TransactionReference
		response["paymentReference"] = korapayResponse.Data.PaymentReference
		response["fee"] = korapayResponse.Data.Fee
		response["narration"] = korapayResponse.Data.Narration
		response["authModel"] = korapayResponse.Data.AuthModel
	}

	c.JSON(http.StatusOK, response)
}

// KorapayCallbackHandler handles Korapay payment callbacks
func KorapayCallbackHandler(c *gin.Context) {
	log.Println("📞 Received Korapay callback")

	// Parse the callback request
	var callbackReq KorapayCallbackRequest
	if err := c.ShouldBindJSON(&callbackReq); err != nil {
		log.Printf("Failed to parse Korapay callback request: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid callback payload", "details": err.Error()})
		return
	}

	// Log the callback details
	log.Printf("🔍 Processing Korapay callback for reference: %s", callbackReq.Data.Reference)

	// Process the callback
	err := korapay.ProcessKorapayCallback(korapay.KorapayCallbackRequest{
		Event: callbackReq.Event,
		Data: korapay.KorapayCallbackData{
			Reference:        callbackReq.Data.Reference,
			PaymentReference: callbackReq.Data.PaymentReference,
			Currency:         callbackReq.Data.Currency,
			Amount:           callbackReq.Data.Amount,
			Fee:              callbackReq.Data.Fee,
			PaymentMethod:    callbackReq.Data.PaymentMethod,
			Status:           callbackReq.Data.Status,
		},
	})
	if err != nil {
		log.Printf("Failed to process Korapay callback: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to process callback"})
		return
	}

	// Find the transaction by Korapay payment reference
	var transaction transactions.TransactionModel
	db := database.GetConnection()

	// Log what we're searching for
	log.Printf("Searching for transaction with Korapay payment reference: %s", callbackReq.Data.PaymentReference)

	// Try to find by Korapay payment reference (stored in CheckoutRequestID)
	err = db.Where("checkoutRequestID = ?", callbackReq.Data.PaymentReference).First(&transaction).Error
	if err != nil {
		log.Printf("Transaction not found for Korapay reference: %s", callbackReq.Data.PaymentReference)
		c.JSON(http.StatusNotFound, gin.H{"error": "Transaction not found"})
		return
	}

	// Update transaction status based on callback event
	var newStatus string
	var transactionStatus string
	var transactionReport string
	var failureReason string

	switch callbackReq.Event {
	case "charge.success":
		newStatus = "success"
		transactionStatus = "COMPLETE"
		transactionReport = "COMPLETE"
	case "charge.failed":
		newStatus = "failed"
		transactionStatus = "FAILED"
		transactionReport = "FAILED"
		// Extract failure reason from status or use default message
		failureReason = callbackReq.Data.Status
		if failureReason == "" {
			failureReason = "Payment failed"
		}
	default:
		log.Printf("Unknown Korapay event: %s", callbackReq.Event)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Unknown event type"})
		return
	}

	// Update transaction in database
	transaction.TransactionStatus = newStatus
	if err := db.Save(&transaction).Error; err != nil {
		log.Printf("Failed to update transaction: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update transaction"})
		return
	}

	// Update merchant balance if payment was successful
	if callbackReq.Event == "charge.success" {
		// Get merchant balance
		var balance balances.MerchantBalance
		err = db.Where("impalaMerchantId = ?", transaction.ImpalaMerchantID).First(&balance).Error
		if err != nil {
			log.Printf("Balance not found for merchant %s", transaction.ImpalaMerchantID)
		} else {
			// Add amount to balance based on currency
			switch callbackReq.Data.Currency {
			case "KES":
				balance.KESBalance += float64(callbackReq.Data.Amount)
			case "UGX":
				balance.UGXBalance += float64(callbackReq.Data.Amount)
			case "USD":
				balance.USDBalance += float64(callbackReq.Data.Amount)
			case "EUR":
				balance.EURBalance += float64(callbackReq.Data.Amount)
			case "GBP":
				balance.GBPBalance += float64(callbackReq.Data.Amount)
			case "TZS":
				balance.TZSBalance += float64(callbackReq.Data.Amount)
			case "XAF":
				balance.XAFBalance += float64(callbackReq.Data.Amount)
			default:
				log.Printf("Unsupported currency for balance update: %s", callbackReq.Data.Currency)
			}
			if err := db.Save(&balance).Error; err != nil {
				log.Printf("Failed to update balance: %v", err)
			}
		}
	}

	// Calculate net amount (amount minus fees)
	netAmount := callbackReq.Data.Amount - int(callbackReq.Data.Fee)
	if netAmount < 0 {
		netAmount = callbackReq.Data.Amount // Fallback to original amount if calculation results in negative
	}

	// Prepare callback response for merchant
	callbackResponse := CallbackResponse{
		Amount:            callbackReq.Data.Amount,
		Currency:          callbackReq.Data.Currency,
		ExternalID:        transaction.ExternalID,
		NetAmount:         netAmount,
		SecureID:          transaction.SecureID,
		TransactionReport: transactionReport,
		TransactionStatus: transactionStatus,
	}

	// Add reason only for failed transactions
	if transactionStatus == "FAILED" {
		callbackResponse.Reason = failureReason
	}

	// Send callback to merchant if callback URL is provided
	if transaction.CallbackURL != "" {
		go func() {
			// Send HTTP POST request to merchant's callback URL
			jsonData, _ := json.Marshal(callbackResponse)
			resp, err := http.Post(transaction.CallbackURL, "application/json", bytes.NewBuffer(jsonData))
			if err != nil {
				log.Printf("Failed to send callback to merchant: %v", err)
			} else {
				resp.Body.Close()
				log.Printf("Callback sent to merchant: %s", transaction.CallbackURL)
			}
		}()
	}

	log.Printf("✅ Korapay callback processed successfully for reference: %s", callbackReq.Data.Reference)
	c.JSON(http.StatusOK, gin.H{"status": "success", "message": "Callback processed"})
}

// FlutterwavePaymentHandler handles Flutterwave payment initiation
func FlutterwavePaymentHandler(c *gin.Context) {
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

	// Parse the Flutterwave payment request
	var req FlutterwavePaymentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input"})
		return
	}

	// Prevent duplicate externalId for this merchant
	if req.ExternalID != "" {
		if exists, err := transactions.ExternalIDExists(req.ImpalaMerchantId, req.ExternalID); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to validate externalId", "details": err.Error()})
			return
		} else if exists {
			c.JSON(http.StatusConflict, gin.H{
				"error":   "DUPLICATE_EXTERNAL_ID",
				"message": "A transaction with this externalId already exists for this merchant",
			})
			return
		}
	}

	// Verify the merchant ID exists
	userID, err := users.GetUserByMerchantId(req.ImpalaMerchantId)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Merchant not found"})
		return
	}

	// Get user by ID
	user, err := users.GetUserByID(uint(userID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve user"})
		return
	}

	fmt.Printf("Flutterwave payment initiated for user: %s with amount: %d\n", user.Name, req.Amount)

	// Generate secure ID for reference
	secureID := flutterwave.GenerateSecureID()

	// Initiate Flutterwave payment
	flutterwaveResponse, err := flutterwave.InitiateFlutterwavePayment(
		req.PayerPhone,
		req.CustomerEmail,
		req.Amount,
		req.Currency,
		secureID,
	)

	if err != nil {
		log.Printf("Failed to initiate Flutterwave payment: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to initiate payment"})
		return
	}

	// Create transaction record
	transaction := transactions.TransactionModel{
		SecureID:          secureID,
		ExternalID:        req.ExternalID,
		ImpalaMerchantID:  req.ImpalaMerchantId,
		Amount:            req.Amount,
		Currency:          req.Currency,
		TransactionStatus: "processing",
		TransactionReport: "collection",
		SourceOfFunds:     "flutterwave",
		CallbackURL:       req.CallbackURL,
		RedirectURL:       req.RedirectURL,
		DateAdded:         time.Now().Unix(),
	}

	// Save transaction to database
	if err := database.GetConnection().Create(&transaction).Error; err != nil {
		log.Printf("Failed to save transaction: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save transaction"})
		return
	}

	// Prepare response
	response := gin.H{
		"status":            flutterwaveResponse.Status,
		"message":           flutterwaveResponse.Message,
		"secureId":          secureID,
		"externalId":        req.ExternalID,
		"amount":            req.Amount,
		"currency":          req.Currency,
		"provider":          "flutterwave",
		"transactionStatus": "processing",
	}

	if flutterwaveResponse.Data != nil {
		response["transactionReference"] = flutterwaveResponse.Data.FlwRef
		response["paymentReference"] = flutterwaveResponse.Data.TxRef
		response["fee"] = flutterwaveResponse.Data.AppFee
		response["narration"] = flutterwaveResponse.Data.Narration
		response["authModel"] = flutterwaveResponse.Data.AuthModel
		response["processorResponse"] = flutterwaveResponse.Data.ProcessorResponse
	}

	c.JSON(http.StatusOK, response)
}

// FlutterwaveCallbackHandler handles Flutterwave payment callbacks
func FlutterwaveCallbackHandler(c *gin.Context) {
	log.Println("📞 Received Flutterwave callback")

	// Parse the callback request
	var callbackReq FlutterwaveCallbackRequest
	if err := c.ShouldBindJSON(&callbackReq); err != nil {
		log.Printf("Failed to parse Flutterwave callback request: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid callback payload", "details": err.Error()})
		return
	}

	// Log the callback details
	log.Printf("🔍 Processing Flutterwave callback for tx_ref: %s", callbackReq.Data.TxRef)

	// Process the callback
	err := flutterwave.ProcessFlutterwaveCallback(flutterwave.FlutterwaveCallbackRequest{
		Event: callbackReq.Event,
		Data: flutterwave.FlutterwaveCallbackData{
			ID:                callbackReq.Data.ID,
			TxRef:             callbackReq.Data.TxRef,
			FlwRef:            callbackReq.Data.FlwRef,
			DeviceFingerprint: callbackReq.Data.DeviceFingerprint,
			Amount:            callbackReq.Data.Amount,
			Currency:          callbackReq.Data.Currency,
			ChargedAmount:     callbackReq.Data.ChargedAmount,
			AppFee:            callbackReq.Data.AppFee,
			MerchantFee:       callbackReq.Data.MerchantFee,
			ProcessorResponse: callbackReq.Data.ProcessorResponse,
			AuthModel:         callbackReq.Data.AuthModel,
			IP:                callbackReq.Data.IP,
			Narration:         callbackReq.Data.Narration,
			Status:            callbackReq.Data.Status,
			PaymentType:       callbackReq.Data.PaymentType,
			CreatedAt:         callbackReq.Data.CreatedAt,
			AccountID:         callbackReq.Data.AccountID,
			Customer: flutterwave.FlutterwaveCustomer{
				ID:          callbackReq.Data.Customer.ID,
				PhoneNumber: callbackReq.Data.Customer.PhoneNumber,
				Name:        callbackReq.Data.Customer.Name,
				Email:       callbackReq.Data.Customer.Email,
				CreatedAt:   callbackReq.Data.Customer.CreatedAt,
			},
		},
		EventType: callbackReq.EventType,
	})
	if err != nil {
		log.Printf("Failed to process Flutterwave callback: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to process callback"})
		return
	}

	// Find the transaction by external ID or reference
	var transaction transactions.TransactionModel
	db := database.GetConnection()

	// Try to find by external ID first, then by reference
	err = db.Where("externalId = ? OR secureId LIKE ?", callbackReq.Data.TxRef, "%"+callbackReq.Data.TxRef+"%").First(&transaction).Error
	if err != nil {
		log.Printf("Transaction not found for reference: %s", callbackReq.Data.TxRef)
		c.JSON(http.StatusNotFound, gin.H{"error": "Transaction not found"})
		return
	}

	// Update transaction status based on callback event
	var newStatus string
	var transactionStatus string
	var transactionReport string
	var failureReason string

	if callbackReq.Event == "charge.completed" {
		if callbackReq.Data.Status == "successful" {
			newStatus = "success"
			transactionStatus = "COMPLETE"
			transactionReport = "COMPLETE"
		} else if callbackReq.Data.Status == "failed" {
			newStatus = "failed"
			transactionStatus = "FAILED"
			transactionReport = "FAILED"
			// Extract failure reason from processor response
			failureReason = callbackReq.Data.ProcessorResponse
			if failureReason == "" {
				failureReason = "Payment failed"
			}
		} else {
			log.Printf("Unknown Flutterwave status: %s", callbackReq.Data.Status)
			c.JSON(http.StatusBadRequest, gin.H{"error": "Unknown status"})
			return
		}
	} else {
		log.Printf("Unknown Flutterwave event: %s", callbackReq.Event)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Unknown event type"})
		return
	}

	// Update transaction in database
	transaction.TransactionStatus = newStatus
	if err := db.Save(&transaction).Error; err != nil {
		log.Printf("Failed to update transaction: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update transaction"})
		return
	}

	// Update merchant balance if payment was successful
	if callbackReq.Data.Status == "successful" {
		// Get merchant balance
		var balance balances.MerchantBalance
		err = db.Where("impalaMerchantId = ?", transaction.ImpalaMerchantID).First(&balance).Error
		if err != nil {
			log.Printf("Balance not found for merchant %s", transaction.ImpalaMerchantID)
		} else {
			// Add amount to balance based on currency
			switch callbackReq.Data.Currency {
			case "KES":
				balance.KESBalance += float64(callbackReq.Data.Amount)
			case "UGX":
				balance.UGXBalance += float64(callbackReq.Data.Amount)
			case "USD":
				balance.USDBalance += float64(callbackReq.Data.Amount)
			case "EUR":
				balance.EURBalance += float64(callbackReq.Data.Amount)
			case "GBP":
				balance.GBPBalance += float64(callbackReq.Data.Amount)
			case "TZS":
				balance.TZSBalance += float64(callbackReq.Data.Amount)
			case "XAF":
				balance.XAFBalance += float64(callbackReq.Data.Amount)
			default:
				log.Printf("Unsupported currency for balance update: %s", callbackReq.Data.Currency)
			}
			if err := db.Save(&balance).Error; err != nil {
				log.Printf("Failed to update balance: %v", err)
			}
		}
	}

	// Calculate net amount (amount minus fees)
	netAmount := callbackReq.Data.Amount - int(callbackReq.Data.AppFee)
	if netAmount < 0 {
		netAmount = callbackReq.Data.Amount // Fallback to original amount if calculation results in negative
	}

	// Prepare callback response for merchant
	callbackResponse := CallbackResponse{
		Amount:            callbackReq.Data.Amount,
		Currency:          callbackReq.Data.Currency,
		ExternalID:        transaction.ExternalID,
		NetAmount:         netAmount,
		SecureID:          transaction.SecureID,
		TransactionReport: transactionReport,
		TransactionStatus: transactionStatus,
	}

	// Add reason only for failed transactions
	if transactionStatus == "FAILED" {
		callbackResponse.Reason = failureReason
	}

	// Send callback to merchant if callback URL is provided
	if transaction.CallbackURL != "" {
		go func() {
			// Send HTTP POST request to merchant's callback URL
			jsonData, _ := json.Marshal(callbackResponse)
			resp, err := http.Post(transaction.CallbackURL, "application/json", bytes.NewBuffer(jsonData))
			if err != nil {
				log.Printf("Failed to send callback to merchant: %v", err)
			} else {
				resp.Body.Close()
				log.Printf("Callback sent to merchant: %s", transaction.CallbackURL)
			}
		}()
	}

	log.Printf("✅ Flutterwave callback processed successfully for tx_ref: %s", callbackReq.Data.TxRef)
	c.JSON(http.StatusOK, gin.H{"status": "success", "message": "Callback processed"})
}

// formatPhoneNumber converts international format (254XXXXXXXXX) to local format (0XXXXXXXXX)
func formatPhoneNumber(phoneNumber string) string {
	// Remove any + prefix if present
	phone := strings.TrimPrefix(phoneNumber, "+")

	// Remove any spaces
	phone = strings.ReplaceAll(phone, " ", "")

	// Convert 254XXXXXXXXX to 0XXXXXXXXX (Kenya)
	if strings.HasPrefix(phone, "254") && len(phone) == 12 {
		return "0" + phone[3:]
	}

	// Convert 256XXXXXXXXX to 0XXXXXXXXX (Uganda)
	if strings.HasPrefix(phone, "256") && len(phone) == 12 {
		return "0" + phone[3:]
	}

	// Convert 255XXXXXXXXX to 0XXXXXXXXX (Tanzania)
	if strings.HasPrefix(phone, "255") && len(phone) == 12 {
		return "0" + phone[3:]
	}

	// Convert 233XXXXXXXXX to 0XXXXXXXXX (Ghana)
	if strings.HasPrefix(phone, "233") && len(phone) == 12 {
		return "0" + phone[3:]
	}

	// Convert 234XXXXXXXXX to 0XXXXXXXXX (Nigeria)
	if strings.HasPrefix(phone, "234") && len(phone) == 13 {
		return "0" + phone[3:]
	}

	// Convert 225XXXXXXXXX to 0XXXXXXXXX (Ivory Coast)
	if strings.HasPrefix(phone, "225") && len(phone) == 11 {
		return "0" + phone[3:]
	}

	// Convert 237XXXXXXXXX to 0XXXXXXXXX (Cameroon)
	if strings.HasPrefix(phone, "237") && len(phone) == 12 {
		return "0" + phone[3:]
	}

	// If already in local format (starts with 0), return as is
	if strings.HasPrefix(phone, "0") {
		return phone
	}

	// If no conversion matched, return original (might already be in correct format)
	return phone
}

// getPaymentProvider determines which payment provider to use based on country
func getPaymentProvider(country string) string {
	// Flutterwave countries
	flutterwaveCountries := map[string]bool{
		"BF": true, // Burkina Faso
		"RW": true, // Rwanda
		"SN": true, // Senegal
		"UG": true, // Uganda
		"ZM": true, // Zambia
		"TZ": true, // Tanzania
	}

	// Korapay countries
	korapayCountries := map[string]bool{
		"CI": true, // Ivory Coast
		"KE": true, // Kenya
		"CM": true, // Cameroon
		"GH": true, // Ghana
		"NG": true, // Nigeria
	}

	countryUpper := strings.ToUpper(country)

	if flutterwaveCountries[countryUpper] {
		return "flutterwave"
	}

	if korapayCountries[countryUpper] {
		return "korapay"
	}

	// Default to Flutterwave if country not found
	return "flutterwave"
}

// UnifiedPaymentHandler handles payment initiation and dynamically routes to Flutterwave or Korapay
func UnifiedPaymentHandler(c *gin.Context) {
	// Get the Authorization header
	authHeader := c.GetHeader("Authorization")
	if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization header is required"})
		return
	}

	// Extract the token from the Bearer scheme
	tokenString := strings.TrimPrefix(authHeader, "Bearer ")
	if tokenString == authHeader {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid authorization token format"})
		return
	}

	// Verify the token
	err := auth.VerifyToken(tokenString)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or expired token", "details": err.Error()})
		return
	}

	// Parse the unified payment request
	var req UnifiedPaymentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input", "details": err.Error()})
		return
	}

	// Prevent duplicate externalId for this merchant
	if req.ExternalID != "" {
		if exists, err := transactions.ExternalIDExists(req.ImpalaMerchantId, req.ExternalID); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to validate externalId", "details": err.Error()})
			return
		} else if exists {
			c.JSON(http.StatusConflict, gin.H{
				"error":   "DUPLICATE_EXTERNAL_ID",
				"message": "A transaction with this externalId already exists for this merchant",
			})
			return
		}
	}

	// Verify the merchant ID exists
	userID, err := users.GetUserByMerchantId(req.ImpalaMerchantId)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Merchant not found"})
		return
	}

	// Get user by ID
	user, err := users.GetUserByID(uint(userID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve user"})
		return
	}

	// Determine which provider to use
	provider := getPaymentProvider(req.Country)
	log.Printf("Unified payment initiated for user: %s, country: %s, provider: %s, amount: %d", user.Name, req.Country, provider, req.Amount)

	// Generate secure ID for transaction reference
	secureID := flutterwave.GenerateSecureID()

	// Get transaction ID from database (will be set after transaction is created)
	var transactionID uint

	// Route to appropriate provider
	if provider == "korapay" {
		// Validate Korapay required fields
		if req.CustomerName == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "customerName is required for Korapay payments"})
			return
		}
		if req.Description == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "description is required for Korapay payments"})
			return
		}
		if req.RedirectURL == "" {
			req.RedirectURL = req.CallbackURL // Use callback URL as fallback
		}

		// Korapay uses phone number with country code (254771850050)
		// Remove + prefix if present, but keep country code
		korapayPhone := strings.TrimPrefix(req.PayerPhone, "+")
		korapayPhone = strings.ReplaceAll(korapayPhone, " ", "")
		log.Printf("Korapay phone number (with country code): %s", korapayPhone)

		// Initiate Korapay payment
		korapayResponse, err := korapay.InitiateKorapayPayment(
			korapayPhone,
			req.CustomerName,
			req.CustomerEmail,
			req.Amount,
			req.Currency,
			req.Description,
			req.CallbackURL,
			req.RedirectURL,
		)

		if err != nil {
			log.Printf("Failed to initiate Korapay payment: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to initiate payment"})
			return
		}

		// Create transaction record
		transaction := transactions.TransactionModel{
			SecureID:          secureID,
			ExternalID:        req.ExternalID,
			ImpalaMerchantID:  req.ImpalaMerchantId,
			Amount:            req.Amount,
			Currency:          req.Currency,
			TransactionStatus: "processing",
			TransactionReport: "collection",
			SourceOfFunds:     "korapay",
			CallbackURL:       req.CallbackURL,
			RedirectURL:       req.RedirectURL,
			DateAdded:         time.Now().Unix(),
		}

		if korapayResponse.Data != nil {
			transaction.MerchantRequestID = korapayResponse.Data.TransactionReference
			transaction.CheckoutRequestID = korapayResponse.Data.PaymentReference
		}

		// Save transaction to database
		if err := database.GetConnection().Create(&transaction).Error; err != nil {
			log.Printf("Failed to save transaction: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save transaction"})
			return
		}

		transactionID = transaction.ID

		// Return standardized response
		c.JSON(http.StatusOK, gin.H{
			"message":       "Payment initiation successful",
			"secureId":      secureID,
			"transactionId": fmt.Sprintf("ImpadlTdest%d", transactionID),
		})
		return
	}

	// Flutterwave payment
	// Format phone number to local format (254XXXXXXXXX -> 0XXXXXXXXX)
	formattedPhone := formatPhoneNumber(req.PayerPhone)
	log.Printf("Phone number formatted: %s -> %s", req.PayerPhone, formattedPhone)

	// Initiate Flutterwave payment
	flutterwaveResponse, err := flutterwave.InitiateFlutterwavePayment(
		formattedPhone,
		req.CustomerEmail,
		req.Amount,
		req.Currency,
		secureID,
	)

	if err != nil {
		log.Printf("Failed to initiate Flutterwave payment: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to initiate payment"})
		return
	}

	// Create transaction record
	transaction := transactions.TransactionModel{
		SecureID:          secureID,
		ExternalID:        req.ExternalID,
		ImpalaMerchantID:  req.ImpalaMerchantId,
		Amount:            req.Amount,
		Currency:          req.Currency,
		TransactionStatus: "processing",
		TransactionReport: "collection",
		SourceOfFunds:     "flutterwave",
		CallbackURL:       req.CallbackURL,
		RedirectURL:       req.RedirectURL,
		DateAdded:         time.Now().Unix(),
	}

	if flutterwaveResponse.Data != nil {
		transaction.MerchantRequestID = flutterwaveResponse.Data.TxRef
		transaction.CheckoutRequestID = flutterwaveResponse.Data.FlwRef
	}

	// Save transaction to database
	if err := database.GetConnection().Create(&transaction).Error; err != nil {
		log.Printf("Failed to save transaction: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save transaction"})
		return
	}

	transactionID = transaction.ID

	// Return standardized response
	c.JSON(http.StatusOK, gin.H{
		"message":       "Payment initiation successful",
		"secureId":      secureID,
		"transactionId": fmt.Sprintf("ImpadlTdest%d", transactionID),
	})
}

// TransferHandler handles transfer from collection balance to merchant balance
func TransferHandler(c *gin.Context) {
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

	// Parse the transfer request
	var req TransferRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input", "details": err.Error()})
		return
	}

	// Verify the merchant ID exists
	userID, err := users.GetUserByMerchantId(req.ImpalaMerchantId)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Merchant not found"})
		return
	}

	// Get user by ID
	user, err := users.GetUserByID(uint(userID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve user"})
		return
	}

	fmt.Printf("Transfer initiated for user: %s with amount: %.2f %s\n", user.Name, req.Amount, req.Currency)

	// Get database connection
	db := database.GetConnection()

	// Start transaction
	tx := db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// Get collection balance
	var collectionBalance balances.MerchantCollectionBalance
	err = tx.Where("impalaMerchantId = ?", req.ImpalaMerchantId).First(&collectionBalance).Error
	if err != nil {
		tx.Rollback()
		c.JSON(http.StatusNotFound, gin.H{"error": "Collection balance not found"})
		return
	}

	// Get merchant balance
	var merchantBalance balances.MerchantBalance
	err = tx.Where("impalaMerchantId = ?", req.ImpalaMerchantId).First(&merchantBalance).Error
	if err != nil {
		tx.Rollback()
		c.JSON(http.StatusNotFound, gin.H{"error": "Merchant balance not found"})
		return
	}

	// Check sufficient collection balance
	var availableBalance float64
	var collectionField, merchantField string

	switch req.Currency {
	case "KES":
		availableBalance = collectionBalance.KESBalance
		collectionField = "kesBalance"
		merchantField = "kesBalance"
	case "UGX":
		availableBalance = collectionBalance.UGXBalance
		collectionField = "ugxBalance"
		merchantField = "ugxBalance"
	case "USD":
		availableBalance = collectionBalance.USDBalance
		collectionField = "usdBalance"
		merchantField = "usdBalance"
	case "EUR":
		availableBalance = collectionBalance.EURBalance
		collectionField = "eurBalance"
		merchantField = "eurBalance"
	case "GBP":
		availableBalance = collectionBalance.GBPBalance
		collectionField = "gbpBalance"
		merchantField = "gbpBalance"
	case "TZS":
		availableBalance = collectionBalance.TZSBalance
		collectionField = "tzsBalance"
		merchantField = "tzsBalance"
	case "XAF":
		availableBalance = collectionBalance.XAFBalance
		collectionField = "xafBalance"
		merchantField = "xafBalance"
	default:
		tx.Rollback()
		c.JSON(http.StatusBadRequest, gin.H{"error": "Unsupported currency"})
		return
	}

	if availableBalance < req.Amount {
		tx.Rollback()
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "INSUFFICIENT_BALANCE",
			"message": fmt.Sprintf("Insufficient collection balance. Available: %.2f %s", availableBalance, req.Currency),
		})
		return
	}

	// Calculate fee (1.5%)
	fee := req.Amount * 0.015
	netAmount := req.Amount - fee

	// Deduct from collection balance
	err = tx.Model(&collectionBalance).Update(collectionField, gorm.Expr(collectionField+" - ?", req.Amount)).Error
	if err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to deduct from collection balance"})
		return
	}

	// Add to merchant balance
	err = tx.Model(&merchantBalance).Update(merchantField, gorm.Expr(merchantField+" + ?", netAmount)).Error
	if err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to add to merchant balance"})
		return
	}

	// Record platform earnings
	earning := drawings.PlatformEarningModel{
		ImpalaMerchantID:   req.ImpalaMerchantId,
		AmountTransferred:  req.Amount,
		TransactionCharges: fee,
		TransferDate:       time.Now(),
	}

	err = tx.Create(&earning).Error
	if err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to record platform earnings"})
		return
	}

	// Commit transaction
	if err := tx.Commit().Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to commit transaction"})
		return
	}

	// Success response
	c.JSON(http.StatusOK, gin.H{
		"status":       "success",
		"message":      "Transfer completed successfully",
		"amount":       req.Amount,
		"currency":     req.Currency,
		"fee":          fee,
		"netAmount":    netAmount,
		"transferDate": time.Now().Format("2006-01-02 15:04:05"),
		"merchantId":   req.ImpalaMerchantId,
	})
}

// PayazaCallbackHandler handles Payaza payment callbacks
func PayazaCallbackHandler(c *gin.Context) {
	log.Println("📞 Received Payaza callback")

	// Parse the callback request
	var callbackReq struct {
		TransactionReference string  `json:"transaction_reference"`
		TransactionStatus    string  `json:"transaction_status"`
		TransactionFee       float64 `json:"transaction_fee"`
		AmountReceived       float64 `json:"amount_received"`
		InitiatedDate        string  `json:"initiated_date"`
		CurrentStatusDate    string  `json:"current_status_date,omitempty"`
		ReceivedFrom         struct {
			AccountName   string `json:"account_name"`
			AccountNumber string `json:"account_number"`
			BankName      string `json:"bank_name"`
		} `json:"received_from"`
		Status                 string `json:"status"`
		SessionID              string `json:"session_id"`
		Channel                string `json:"channel"`
		Branch                 bool   `json:"branch"`
		CurrencyCode           string `json:"currency_code"`
		PayazaAccountReference string `json:"payaza_account_reference"`
		Narration              string `json:"narration"`
		BusinessFK             int    `json:"business_fk"`
	}

	if err := c.ShouldBindJSON(&callbackReq); err != nil {
		log.Printf("❌ Failed to parse Payaza callback request: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid callback payload", "details": err.Error()})
		return
	}

	// Log the callback details
	log.Printf("🔍 Processing Payaza callback for transaction_reference: %s, status: %s", callbackReq.TransactionReference, callbackReq.Status)

	// Get database connection
	db := database.GetConnection()
	if db == nil {
		log.Println("❌ Failed to get database connection")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database connection failed"})
		return
	}

	// Find transaction by matching transaction_reference with secureId or merchantRequestID
	var transaction transactions.TransactionModel
	err := db.Where("secureId = ? OR merchantRequestID = ?", callbackReq.TransactionReference, callbackReq.TransactionReference).
		First(&transaction).Error
	if err != nil {
		log.Printf("❌ Transaction not found for reference: %s, Error: %v", callbackReq.TransactionReference, err)
		c.JSON(http.StatusNotFound, gin.H{"error": "Transaction not found", "details": err.Error()})
		return
	}

	log.Printf("✅ Found transaction: ID=%d, Type=%s, Amount=%d, Currency=%s", transaction.ID, transaction.TransactionReport, transaction.Amount, transaction.Currency)

	// Determine transaction status based on Payaza status
	var transactionStatus, transactionReport, callbackStatus string
	var netAmount float64

	if callbackReq.Status == "Completed" {
		transactionStatus = "COMPLETE"
		transactionReport = "COMPLETE"
		callbackStatus = "SENT"
		// Net amount is amount received minus transaction fee
		netAmount = callbackReq.AmountReceived - callbackReq.TransactionFee
		if netAmount < 0 {
			netAmount = callbackReq.AmountReceived // Fallback to amount received if calculation results in negative
		}

		// Update transaction status
		updates := map[string]interface{}{
			"transactionStatus": transactionStatus,
			"callbackStatus":    callbackStatus,
			"netAmount":         netAmount,
		}

		if err := db.Model(&transactions.TransactionModel{}).
			Where("id = ?", transaction.ID).
			Updates(updates).Error; err != nil {
			log.Printf("❌ Failed to update transaction status: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update transaction status", "details": err.Error()})
			return
		}

		// Update merchant balance (XOF collection balance)
		if err := balances.AddXOFBalance(transaction.ImpalaMerchantID, netAmount); err != nil {
			log.Printf("❌ Failed to update merchant collection balance: %v", err)
			// Don't fail the callback, but log the error
		} else {
			log.Printf("✅ Successfully updated merchant collection balance for %s: %.2f XOF", transaction.ImpalaMerchantID, netAmount)
		}

	} else if callbackReq.Status == "Failed" {
		transactionStatus = "FAILED"
		transactionReport = "FAILED"
		callbackStatus = "SENT"
		netAmount = callbackReq.AmountReceived

		// Update transaction status
		updates := map[string]interface{}{
			"transactionStatus": transactionStatus,
			"callbackStatus":    callbackStatus,
		}

		if err := db.Model(&transactions.TransactionModel{}).
			Where("id = ?", transaction.ID).
			Updates(updates).Error; err != nil {
			log.Printf("❌ Failed to update transaction status: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update transaction status", "details": err.Error()})
			return
		}
	} else {
		log.Printf("⚠️ Unknown Payaza status: %s", callbackReq.Status)
		c.JSON(http.StatusOK, gin.H{"message": "Callback received but status not processed", "status": callbackReq.Status})
		return
	}

	// Prepare callback response for merchant
	callbackResponse := map[string]interface{}{
		"transactionStatus": transactionReport,
		"transactionReport": transactionReport,
		"currency":          transaction.Currency, // Use transaction currency
		"amount":            int(callbackReq.AmountReceived),
		"netAmount":         int(netAmount),
		"secureId":          transaction.SecureID,
		"externalId":        transaction.ExternalID,
	}

	// Send callback to merchant
	if err := SendCallback(transaction.ID, callbackResponse); err != nil {
		log.Printf("⚠️ Failed to send callback to merchant: %v", err)
		// Don't fail the transaction if callback fails
	}

	log.Printf("✅ Successfully processed Payaza callback for transaction: %s", callbackReq.TransactionReference)
	c.JSON(http.StatusOK, gin.H{"message": "Callback processed successfully", "status": callbackReq.Status})
}

func RegisterRoutes(router *gin.RouterGroup) {

	router.GET("/", LoginHandler)
	router.POST("mobile/initiate", MobilePaymentHandler)
	router.POST("mobile/transfer", MobileWithdrawalHandler)
	router.POST("card/initiate", CardPaymentHandler)
	router.POST("usdc/initiate", UsdcPaymentHandler)
	router.POST("mobile/callback", MobileCallbackHandler)
	router.POST("mobile/b2c/callback", B2CCallbackHandler) // Dedicated B2C withdrawal callback endpoint
	router.POST("card/callback", CardCallbackHandler)
	router.POST("usdc/callback", CryptoCallbackHandler)
	router.POST("korapay/initiate", KorapayPaymentHandler)
	router.POST("korapay/callback", KorapayCallbackHandler)
	router.POST("flutterwave/initiate", FlutterwavePaymentHandler)
	router.POST("flutterwave/callback", FlutterwaveCallbackHandler)
	router.POST("payaza/callback", PayazaCallbackHandler)
	router.POST("pay", UnifiedPaymentHandler) // Unified payment endpoint
	router.POST("transfer", TransferHandler)

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
	//balances api
	//
	protected.GET("/read/payouts/balance", GetTotalBalanceHandler)
	protected.GET("/read/payins/balance", GetTotalPayinBalanceHandler) // get payin balance
	// drawings.V1(mercury.Group("/drawings"))
	protected.POST("/wallet/transfer/toPayout", drawings.WalletTransferHandler)
	// virtualcard endpoins

	// migrate the virtual careds
	virtualcards.AutoMigrate()
	protected.POST("/vc/create/holder", CreateCardHolderHandler)
	protected.GET("/vc/list/holders", GetCardHoldersHandler)
	protected.POST("/vc/create/virtual-card", CreateCardHandler)
	router.POST("/vc/callback/card-status", GetVirtualCardByIDCardCallbackHandler)
	protected.GET("/vc/list/virtual-cards", ListVirtualCardsByMerchant) // List virtual cards by merchant
	protected.GET("/transactions", ListTransactionsHandler)
	protected.GET("/transactions/search", SearchTransactionsHandler) // Search transactions with filters
	protected.POST("/vc/card/info", RetrieveCardInfoHandler)
	protected.POST("/vc/card/balance", GetCardBalance)
	// virtual card generation
	protected.POST("/vc/card/recharge", RechargeCardHandler)
	// WEST AFRICA HANDLER
	// WEST AFRICA HANDLER
	router.POST("west-africa/callback", WestAfricaCallbackHandler)
	router.POST("/cameroon/collect/callback", CameroonXAFCallback)
	router.POST("/cameroon/disburse/callback", CameroonXAFDisburseCallback)
	// GLOBPAY URLS
	router.POST("/globpay/card/callback", GlobpayCardCallbackHandler)

	// eCitizen URLs
	router.POST("/ecitizen/validate", ECitizenValidateHandler)
	router.POST("/ecitizen/confirm", ECitizenConfirmHandler)

}
