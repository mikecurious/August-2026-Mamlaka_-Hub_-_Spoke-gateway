package westafrica

// package merc

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
)

// CallbackResponse represents the webhook callback payload
type CallbackResponse struct {
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

// CallbackHandler handles airtime transaction callbacks
type CallbackHandler struct {
	WebhookSecret string
	Logger        *log.Logger
}

// NewCallbackHandler creates a new callback handler
func NewCallbackHandler(webhookSecret string, logger *log.Logger) *CallbackHandler {
	if logger == nil {
		logger = log.New(log.Writer(), "[CALLBACK] ", log.LstdFlags)
	}
	return &CallbackHandler{
		WebhookSecret: webhookSecret,
		Logger:        logger,
	}
}

// ProcessCallback processes the incoming webhook callback
func (h *CallbackHandler) ProcessCallback(w http.ResponseWriter, r *http.Request) {
	// Only accept POST requests
	if r.Method != http.MethodPost {
		h.Logger.Printf("Invalid method: %s", r.Method)
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Read request body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		h.Logger.Printf("Failed to read request body: %v", err)
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	// Verify webhook signature if secret is provided
	if h.WebhookSecret != "" {
		signature := r.Header.Get("X-Webhook-Signature")
		if !h.verifySignature(body, signature) {
			h.Logger.Printf("Invalid webhook signature")
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
	}

	// Parse callback data
	var callback CallbackResponse
	if err := json.Unmarshal(body, &callback); err != nil {
		h.Logger.Printf("Failed to parse callback data: %v", err)
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	// Process the callback
	if err := h.handleCallback(&callback); err != nil {
		h.Logger.Printf("Failed to process callback: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Send success response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"status":  "success",
		"message": "Callback processed successfully",
	})
}

// handleCallback processes the callback based on transaction state
func (h *CallbackHandler) handleCallback(callback *CallbackResponse) error {
	h.Logger.Printf("Processing callback for transaction: %s", callback.TransactionID)
	h.Logger.Printf("State: %s, Amount: %d, Destination: %s",
		callback.State, callback.Amount, callback.Destination)

	switch strings.ToUpper(callback.State) {
	case "SUCCESSFUL":
		return h.handleSuccessfulTransaction(callback)
	case "FAILED":
		return h.handleFailedTransaction(callback)
	case "PENDING", "PENDING1":
		return h.handlePendingTransaction(callback)
	case "CANCELLED":
		return h.handleCancelledTransaction(callback)
	default:
		h.Logger.Printf("Unknown transaction state: %s", callback.State)
		return h.handleUnknownState(callback)
	}
}

// handleSuccessfulTransaction handles successful transactions
func (h *CallbackHandler) handleSuccessfulTransaction(callback *CallbackResponse) error {
	h.Logger.Printf("✅ Transaction %s completed successfully", callback.TransactionID)

	// Update database with successful transaction
	if err := h.updateTransactionStatus(callback.TransactionID, "SUCCESSFUL"); err != nil {
		return fmt.Errorf("failed to update transaction status: %w", err)
	}

	// Send success notification
	if err := h.sendNotification(callback, "Transaction completed successfully"); err != nil {
		h.Logger.Printf("Failed to send success notification: %v", err)
		// Don't return error as the main transaction succeeded
	}

	// Process any business logic for successful airtime top-up
	if err := h.processSuccessfulAirtime(callback); err != nil {
		return fmt.Errorf("failed to process successful airtime: %w", err)
	}

	return nil
}

// handleFailedTransaction handles failed transactions
func (h *CallbackHandler) handleFailedTransaction(callback *CallbackResponse) error {
	h.Logger.Printf("❌ Transaction %s failed", callback.TransactionID)

	errorMsg := "Unknown error"
	if callback.Error != nil {
		errorMsg = *callback.Error
	}

	h.Logger.Printf("Error: %s", errorMsg)

	// Update database with failed status
	if err := h.updateTransactionStatus(callback.TransactionID, "FAILED"); err != nil {
		return fmt.Errorf("failed to update transaction status: %w", err)
	}

	// Send failure notification
	if err := h.sendNotification(callback, fmt.Sprintf("Transaction failed: %s", errorMsg)); err != nil {
		h.Logger.Printf("Failed to send failure notification: %v", err)
	}

	// Process refund if applicable
	if err := h.processRefund(callback); err != nil {
		return fmt.Errorf("failed to process refund: %w", err)
	}

	return nil
}

// handlePendingTransaction handles pending transactions
func (h *CallbackHandler) handlePendingTransaction(callback *CallbackResponse) error {
	h.Logger.Printf("⏳ Transaction %s is pending", callback.TransactionID)

	// Update database with pending status
	if err := h.updateTransactionStatus(callback.TransactionID, callback.State); err != nil {
		return fmt.Errorf("failed to update transaction status: %w", err)
	}

	// Send pending notification
	if err := h.sendNotification(callback, "Transaction is being processed"); err != nil {
		h.Logger.Printf("Failed to send pending notification: %v", err)
	}

	return nil
}

// handleCancelledTransaction handles cancelled transactions
func (h *CallbackHandler) handleCancelledTransaction(callback *CallbackResponse) error {
	h.Logger.Printf("🚫 Transaction %s was cancelled", callback.TransactionID)

	// Update database with cancelled status
	if err := h.updateTransactionStatus(callback.TransactionID, "CANCELLED"); err != nil {
		return fmt.Errorf("failed to update transaction status: %w", err)
	}

	// Send cancellation notification
	if err := h.sendNotification(callback, "Transaction was cancelled"); err != nil {
		h.Logger.Printf("Failed to send cancellation notification: %v", err)
	}

	// Process refund for cancelled transaction
	if err := h.processRefund(callback); err != nil {
		return fmt.Errorf("failed to process refund: %w", err)
	}

	return nil
}

// handleUnknownState handles unknown transaction states
func (h *CallbackHandler) handleUnknownState(callback *CallbackResponse) error {
	h.Logger.Printf("⚠️  Unknown state for transaction %s: %s", callback.TransactionID, callback.State)

	// Log the unknown state but don't fail the callback
	// This allows the system to handle new states gracefully
	return nil
}

// verifySignature verifies the webhook signature
func (h *CallbackHandler) verifySignature(body []byte, signature string) bool {
	if signature == "" {
		return false
	}

	// Remove "sha256=" prefix if present
	signature = strings.TrimPrefix(signature, "sha256=")

	// Compute expected signature
	mac := hmac.New(sha256.New, []byte(h.WebhookSecret))
	mac.Write(body)
	expectedSignature := hex.EncodeToString(mac.Sum(nil))

	// Compare signatures
	return hmac.Equal([]byte(signature), []byte(expectedSignature))
}

// updateTransactionStatus updates the transaction status in database
func (h *CallbackHandler) updateTransactionStatus(transactionID, status string) error {
	// TODO: Implement database update logic
	h.Logger.Printf("Updating transaction %s status to %s", transactionID, status)

	// Example database update (replace with your actual implementation)
	// db.Exec("UPDATE transactions SET status = ?, updated_at = ? WHERE transaction_id = ?",
	//         status, time.Now(), transactionID)

	return nil
}

// sendNotification sends notification to user
func (h *CallbackHandler) sendNotification(callback *CallbackResponse, message string) error {
	// TODO: Implement notification logic (SMS, email, push notification, etc.)
	h.Logger.Printf("Sending notification for %s: %s", callback.TransactionID, message)

	// Example notification implementation
	// - Send SMS to callback.Destination
	// - Send email notification
	// - Send push notification
	// - Update user wallet balance

	return nil
}

// processSuccessfulAirtime processes successful airtime top-up
func (h *CallbackHandler) processSuccessfulAirtime(callback *CallbackResponse) error {
	h.Logger.Printf("Processing successful airtime top-up: %d %s to %s",
		callback.Amount, callback.Currency, callback.Destination)

	// TODO: Implement business logic for successful airtime
	// - Update user airtime balance
	// - Record transaction in ledger
	// - Update commission tracking
	// - Generate receipt

	return nil
}

// processRefund processes refund for failed/cancelled transactions
func (h *CallbackHandler) processRefund(callback *CallbackResponse) error {
	h.Logger.Printf("Processing refund for transaction %s: %d %s",
		callback.TransactionID, callback.Amount, callback.Currency)

	// TODO: Implement refund logic
	// - Credit user wallet
	// - Create refund transaction record
	// - Send refund notification

	return nil
}

// GetCallbackInfo returns formatted callback information
func (h *CallbackHandler) GetCallbackInfo(callback *CallbackResponse) map[string]interface{} {
	return map[string]interface{}{
		"transaction_id":      callback.TransactionID,
		"amount":              callback.Amount,
		"currency":            callback.Currency,
		"destination":         callback.Destination,
		"state":               callback.State,
		"customer_name":       callback.CustomerName,
		"provider_id":         callback.ProviderID,
		"commission":          callback.Commission,
		"fee":                 callback.Fee,
		"benefice":            callback.Benefice,
		"created_at":          callback.CreatedAt,
		"updated_at":          callback.UpdatedAt,
		"custom_data":         callback.CustomData,
		"transaction_channel": callback.TransactionChannel,
		"error":               callback.Error,
	}
}

// Example HTTP server setup
// func SetupCallbackServer(handler *CallbackHandler, port int) {
// 	http.HandleFunc("/webhook/airtime", handler.ProcessCallback)

// 	// Health check endpoint
// 	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
// 		w.WriteHeader(http.StatusOK)
// 		json.NewEncoder(w).Encode(map[string]string{"status": "healthy"})
// 	})

// 	addr := fmt.Sprintf(":%d", port)
// 	log.Printf("Starting callback server on %s", addr)
// 	log.Fatal(http.ListenAndServe(addr, nil))
// }

// Example usage
// func main() {
// 	// Create callback handler
// 	handler := NewCallbackHandler("your-webhook-secret", nil)

// 	// Setup and start server
// 	SetupCallbackServer(handler, 8080)
// }
