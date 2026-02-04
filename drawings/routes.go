package drawings

import (
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"com.mam-laka/database"
	"github.com/gin-gonic/gin"
)

// func V1(router *gin.RouterGroup) {
// 	protected := router.Group("/")
// 	protected.Use(auth.AuthMiddleware())
// 	protected.GET("/read/:id", ReadSingleWithdrawalRequest)
// 	protected.GET("/list", listRequests)
// 	protected.GET("/view", ViewEarnings)
// 	protected.GET("/list/withdrawal", readWithdrawalRequests)
// 	protected.GET("/list/transfer", readTransferRequests)
// 	protected.GET("/status/:status", ReadWithdrawalRequestsByStatus)
// 	protected.PUT("/update/:id", UpdateWithdrawalRequestHandler)
// 	protected.DELETE("/delete/:id", DeleteWithdrawalRequestHandler)
// 	// Route for creating a new withdrawal request
// 	protected.POST("/request/transfer", CreateWithdrawalRequest)
// 	// wallet to  wallet transfer
// 	protected.POST("/transfer/toPayout", WalletTransferHandler)
// 	// walet to collection handler
// 	protected.POST("/transfer/toCollection", WalletToCollectionTransferHandler)

// }

// get withdrawal rquests
func readAllWithdrawalRequests(c *gin.Context) {

	// Retrieve transactions based on status and date range
	requests, err := GetWithdrawalRequests()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve requests", "details": err.Error()})
		return
	}

	// Serialize and respond with the transactions
	var response []map[string]interface{} // Use the type that `serializer.Response()` returns
	for _, requests := range requests {
		serializer := NewWithdrawalRequestSerializer(c, requests)
		response = append(response, serializer.Response()) // Append the serialized response
	}

	c.JSON(http.StatusOK, gin.H{"requests": response})
}

// filter based on token
func readWithdrawalRequests(c *gin.Context) {
	// imports
	roleID, roleExists := c.Get("roleID")
	merchantID, merchantExists := c.Get("merchantID")

	var drawings []WithdrawalRequestModel
	var transactionError error

	fmt.Println(roleID, merchantID)

	// Check if roleID and merchantID exist
	if !roleExists || !merchantExists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Missing authentication details"})
		return
	}

	// Ensure roleID is an integer (handle uint cases safely)
	var roleIdInt int
	switch v := roleID.(type) {
	case int:
		roleIdInt = v
	case uint:
		roleIdInt = int(v)
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid roleID type"})
		return
	}

	// Ensure merchantID is a string
	merchantIdStr, ok := merchantID.(string)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid merchantID type"})
		return
	}

	// Fetch Drawings  based on role
	if roleIdInt == 1 { // Admin: List all transactions
		drawings, transactionError = GetAllWithdrawalRequests()
	} else { // Merchant: List transactions associated with the merchant
		drawings, transactionError = GetMerchantWithdrawalRequests(merchantIdStr)
	}

	// Handle transaction retrieval errors
	if transactionError != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": transactionError.Error()})
		return
	}
	//end imports

	// Serialize and respond with the transactions
	var response []map[string]interface{} // Use the type that `serializer.Response()` returns
	for _, requests := range drawings {
		serializer := NewWithdrawalRequestSerializer(c, requests)
		response = append(response, serializer.Response()) // Append the serialized response
	}

	c.JSON(http.StatusOK, gin.H{"requests": response})
}

// get tranfere requests
func readTransferRequests(c *gin.Context) {

	// imports
	roleID, roleExists := c.Get("roleID")
	merchantID, merchantExists := c.Get("merchantID")

	var drawings []WithdrawalRequestModel
	var transactionError error

	fmt.Println(roleID, merchantID)

	// Check if roleID and merchantID exist
	if !roleExists || !merchantExists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Missing authentication details"})
		return
	}

	// Ensure roleID is an integer (handle uint cases safely)
	var roleIdInt int
	switch v := roleID.(type) {
	case int:
		roleIdInt = v
	case uint:
		roleIdInt = int(v)
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid roleID type"})
		return
	}

	// Ensure merchantID is a string
	merchantIdStr, ok := merchantID.(string)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid merchantID type"})
		return
	}

	// Fetch Drawings  based on role
	if roleIdInt == 1 { // Admin: List all transactions
		drawings, transactionError = GetAllWalletToWalletRequests()
	} else { // Merchant: List transactions associated with the merchant
		drawings, transactionError = GetMerchantWalletToWalletRequests(merchantIdStr)
	}

	// Handle transaction retrieval errors
	if transactionError != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": transactionError.Error()})
		return
	}
	//end imports

	// Serialize and respond with the transactions
	var response []map[string]interface{} // Use the type that `serializer.Response()` returns
	for _, requests := range drawings {
		serializer := NewWithdrawalRequestSerializer(c, requests)
		response = append(response, serializer.Response()) // Append the serialized response
	}

	c.JSON(http.StatusOK, gin.H{"requests": response})
}

// CreateWithdrawalRequest handles the creation of a new withdrawal request.
func CreateWithdrawalRequest(c *gin.Context) {
	// Create a new instance of the validator
	validator := NewWithdrawalRequestModelValidator()

	// Bind the request data to the validator (this calls the Bind method)
	if err := validator.Bind(c); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format", "details": err.Error()})
		return
	}

	// Validate required fields
	if validator.WithdrawalRequestModel.Amount <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Amount must be greater than 0"})
		return
	}

	// Check if the transfer type is valid
	// if validator.WithdrawalRequestModel.TransferType != "withdraw" {
	// 	c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid transfer type, must be 'withdraw'"})
	// 	return
	// }

	// Check if the status is valid
	validStatuses := []string{"PENDING", "CANCELED", "APPROVED", "DISBURSED"}
	isValidStatus := false
	for _, status := range validStatuses {
		if validator.WithdrawalRequestModel.Status == status {
			isValidStatus = true
			break
		}
	}
	if !isValidStatus {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid status, must be one of 'PENDING', 'CANCELED', 'APPROVED', 'DISBURSED'"})
		return
	}

	// Save the withdrawal request model to the database
	if err := SaveWithdrawalRequest(&validator.WithdrawalRequestModel); err != nil {
		log.Printf("Error saving withdrawal request: %v", err) // log the error for troubleshooting
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save withdrawal request", "details": err.Error()})
		return
	}

	// Send email and SMS notifications
	// merchantName := validator.WithdrawalRequestModel.ImpalaMerchantID // you can fetch this from the database or use the value in the request
	// amount := fmt.Sprintf("%.2f", validator.WithdrawalRequestModel.Amount)

	// Trigger the notifications after saving the request
	// if err := notifications.SendEmail(merchantName, amount); err != nil {
	// 	log.Printf("Failed to send email: %v", err)
	// }

	// if err := notifications.SendSMS(merchantName, amount); err != nil {
	// 	log.Printf("Failed to send SMS: %v", err)
	// }

	// Respond with the created withdrawal request and a success message
	c.JSON(http.StatusOK, gin.H{
		"message":   "Withdrawal request created successfully",
		"request":   validator.WithdrawalRequestModel,
		"timestamp": time.Now().Format(time.RFC3339),
	})
}

// ReadSingleWithdrawalRequest retrieves a single withdrawal request by ID.
func ReadSingleWithdrawalRequest(c *gin.Context) {
	fmt.Println("Fetching withdrawal request")
	id := c.Param("id")
	idUint, err := strconv.ParseUint(id, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid Withdrawal Request ID"})
		return
	}

	withdrawalRequest, err := GetWithdrawalRequestByID(uint(idUint))
	if err != nil {
		c.JSON(http.StatusInternalServerError, database.NewError("WithdrawalRequest", err))
		return
	}

	serializer := NewWithdrawalRequestSerializer(c, withdrawalRequest)
	c.JSON(http.StatusOK, gin.H{"WithdrawalRequest": serializer.Response()})
}

// ListWithdrawalRequests retrieves a list of all withdrawal requests.
func ListWithdrawalRequests(c *gin.Context) {
	withdrawalRequests, err := GetAllWithdrawalRequests()
	if err != nil {
		c.JSON(http.StatusInternalServerError, database.NewError("WithdrawalRequest", err))
		return
	}

	// Serialize and respond with the withdrawal requests
	var response []map[string]interface{}
	for _, request := range withdrawalRequests {
		serializer := NewWithdrawalRequestSerializer(c, request)
		response = append(response, serializer.Response())
	}

	c.JSON(http.StatusOK, gin.H{"WithdrawalRequests": response})
}

func listRequests(c *gin.Context) {
	// imports
	roleID, roleExists := c.Get("roleID")
	merchantID, merchantExists := c.Get("merchantID")

	var drawings []WithdrawalRequestModel
	var transactionError error

	fmt.Println(roleID, merchantID)

	// Check if roleID and merchantID exist
	if !roleExists || !merchantExists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Missing authentication details"})
		return
	}

	// Ensure roleID is an integer (handle uint cases safely)
	var roleIdInt int
	switch v := roleID.(type) {
	case int:
		roleIdInt = v
	case uint:
		roleIdInt = int(v)
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid roleID type"})
		return
	}

	// Ensure merchantID is a string
	merchantIdStr, ok := merchantID.(string)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid merchantID type"})
		return
	}

	// Fetch Drawings  based on role
	if roleIdInt == 1 { // Admin: List all transactions
		drawings, transactionError = GetAllRequests()
	} else { // Merchant: List transactions associated with the merchant
		drawings, transactionError = GetMerchantRequests(merchantIdStr)
	}

	// Handle transaction retrieval errors
	if transactionError != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": transactionError.Error()})
		return
	}
	//end imports

	// Serialize and respond with the transactions
	var response []map[string]interface{} // Use the type that `serializer.Response()` returns
	for _, requests := range drawings {
		serializer := NewWithdrawalRequestSerializer(c, requests)
		response = append(response, serializer.Response()) // Append the serialized response
	}

	c.JSON(http.StatusOK, gin.H{"WithdrawalRequests": response})
}

func ViewEarnings(c *gin.Context) {
	roleID, roleExists := c.Get("roleID")
	merchantID, merchantExists := c.Get("merchantID")

	if !roleExists || !merchantExists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Missing authentication details"})
		return
	}

	var roleIdInt int
	switch v := roleID.(type) {
	case int:
		roleIdInt = v
	case uint:
		roleIdInt = int(v)
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid roleID type"})
		return
	}

	merchantIdStr, ok := merchantID.(string)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid merchantID type"})
		return
	}

	var (
		earnings []PlatformEarningModel
		err      error
	)

	if roleIdInt == 1 {
		earnings, err = GetAllEarnings()
	} else {
		earnings, err = GetMerchantEarnings(merchantIdStr)
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	var response []map[string]interface{}
	for _, earning := range earnings {
		serializer := NewPlatformEarningSerializer(c, earning)
		response = append(response, serializer.Response())
	}

	c.JSON(http.StatusOK, gin.H{"platformEarnings": response})
}

// ReadWithdrawalRequestsByStatus retrieves withdrawal requests by their status.
func ReadWithdrawalRequestsByStatus(c *gin.Context) {
	status := c.Param("status")
	if status == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Status is required"})
		return
	}

	withdrawalRequests, err := GetWithdrawalRequestsByStatus(status)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve withdrawal requests", "details": err.Error()})
		return
	}

	// Serialize and respond with the withdrawal requests
	var response []map[string]interface{}
	for _, request := range withdrawalRequests {
		serializer := NewWithdrawalRequestSerializer(c, request)
		response = append(response, serializer.Response())
	}

	c.JSON(http.StatusOK, gin.H{"WithdrawalRequests": response})
}

// UpdateWithdrawalRequestHandler updates a withdrawal request by ID.

func UpdateWithdrawalRequestHandler(c *gin.Context) {
	id := c.Param("id")
	idUint, err := strconv.ParseUint(id, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid Withdrawal Request ID"})
		return
	}

	withdrawalRequest, err := GetWithdrawalRequestByID(uint(idUint))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch withdrawal request"})
		return
	}

	type UpdateRequest struct {
		ApprovedBy int    `json:"approvedBy"`
		Status     string `json:"status"`
		// Amount     float32 `json:"amount"`
		Comment string `json:"comment"`
	}
	var updateReq UpdateRequest

	if err := c.ShouldBindJSON(&updateReq); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid JSON data"})
		return
	}

	// Build the base update payload
	updatedData := map[string]interface{}{
		"status":  updateReq.Status,
		"comment": updateReq.Comment,
		// "amount":     updateReq.Amount,
		"approvedBy": updateReq.ApprovedBy,
	}

	// Two-step admin flow:
	// 1) First admin sets status = "CONFIRMED" (no funds move)
	// 2) Second admin sets status = "APPROVED" (funds move via WalletTransfer)
	switch updateReq.Status {
	case "CONFIRMED":
		// Only allow CONFIRMED from PENDING state
		if withdrawalRequest.Status != "PENDING" {
			c.JSON(http.StatusBadRequest, gin.H{
				"error":   "INVALID_STATUS_TRANSITION",
				"message": fmt.Sprintf("Cannot CONFIRM a request in status %s", withdrawalRequest.Status),
			})
			return
		}
	case "APPROVED":
		// Only allow APPROVED from CONFIRMED state
		if withdrawalRequest.Status != "CONFIRMED" {
			c.JSON(http.StatusBadRequest, gin.H{
				"error":   "INVALID_STATUS_TRANSITION",
				"message": fmt.Sprintf("Cannot APPROVE a request in status %s, must be CONFIRMED first", withdrawalRequest.Status),
			})
			return
		}

		// On APPROVED, perform the actual wallet transfer (collection -> payout wallet)
		transferRequest := &TransferRequest{
			ImpalaMerchantId: withdrawalRequest.ImpalaMerchantID,
			Amount:           withdrawalRequest.Amount,
			Currency:         "KES", // toPayout uses KES wallet; adjust if multi-currency is added
		}

		if err := WalletTransfer(transferRequest); err != nil {
			log.Printf("Transfer failed for Withdrawal Request ID %d: %v", idUint, err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "TRANSFER_FAILED", "details": err.Error()})
			return
		}
		log.Printf("Transfer successful for Withdrawal Request ID %d", idUint)
	default:
		// For other statuses (e.g. CANCELED, DISBURSED) we simply update the record
	}

	if err := UpdateSingleWithdrawalRequest(&withdrawalRequest, updatedData); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	serializer := NewWithdrawalRequestSerializer(c, withdrawalRequest)
	c.JSON(http.StatusOK, gin.H{"WithdrawalRequest": serializer.Response()})
}

// DeleteWithdrawalRequestHandler deletes a withdrawal request by ID.
func DeleteWithdrawalRequestHandler(c *gin.Context) {
	id := c.Param("id")
	idUint, err := strconv.ParseUint(id, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid Withdrawal Request ID"})
		return
	}

	withdrawalRequest, err := GetWithdrawalRequestByID(uint(idUint))
	if err != nil {
		c.JSON(http.StatusInternalServerError, database.NewError("WithdrawalRequest", err))
		return
	}

	if err := DeleteWithdrawalRequest(withdrawalRequest.ID); err != nil {
		c.JSON(http.StatusUnprocessableEntity, database.NewError("database", err))
		return
	}

	c.JSON(http.StatusNoContent, gin.H{"message": "Withdrawal Request Deleted"})
}

// WalletTransferHandler now creates a pending withdrawal/transfer request
// that must be confirmed and then approved by admins before funds move.
func WalletTransferHandler(c *gin.Context) {
	// Merchant identity comes from the auth middleware
	merchantIDVal, merchantExists := c.Get("merchantID")
	if !merchantExists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Missing authentication details"})
		return
	}
	merchantID, ok := merchantIDVal.(string)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid merchantID type"})
		return
	}

	// Parse transfer request payload (amount, currency)
	var transferRequest TransferRequest
	if err := c.ShouldBindJSON(&transferRequest); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format", "details": err.Error()})
		return
	}
	// Ensure the merchantId in the body (if any) cannot override the token merchant
	transferRequest.ImpalaMerchantId = merchantID

	if transferRequest.Amount <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Amount must be greater than 0"})
		return
	}

	// Create a pending withdrawal/transfer request.
	// This will later be:
	//  - CONFIRMED by the first admin
	//  - APPROVED by the second admin (which then moves the funds)
	req := WithdrawalRequestModel{
		ImpalaMerchantID: merchantID,
		RequestedBy:      0, // can be mapped to a real user ID in future
		ApprovedBy:       nil,
		Amount:           transferRequest.Amount,
		Status:           "PENDING",
		DateRequested:    time.Now().Unix(),
		TransferType:     "transfer", // wallet (collection) -> payout wallet
		Comment:          fmt.Sprintf("toPayout %s %.2f", transferRequest.Currency, transferRequest.Amount),
	}

	if err := SaveWithdrawalRequest(&req); err != nil {
		log.Printf("Error saving wallet transfer request: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create transfer request", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":   "Transfer request created successfully and is pending approval",
		"requestId": req.ID,
		"status":    req.Status,
	})
}

// WalletTransferHandler handles wallet-to-wallet transfer requests.
func WalletToCollectionTransferHandler(c *gin.Context) {
	// Bind the incoming JSON to the TransferRequest struct
	var transferRequest TransferRequest
	if err := c.ShouldBindJSON(&transferRequest); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format"})
		return
	}

	// Call the WalletTransferBackToCollection function to handle the transaction
	err := WalletTransferBackToCollection(&transferRequest)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Transaction failed", "details": err.Error()})
		return
	}

	// Respond with success message
	c.JSON(http.StatusOK, gin.H{"message": "Transfer successful"})
}
