package merchants

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"com.mam-laka/airtel"
	"com.mam-laka/balances"
	"com.mam-laka/database"
	"com.mam-laka/transactions"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// errAirtelAlreadySettled marks an idempotent no-op inside settleAirtelTransaction.
var errAirtelAlreadySettled = errors.New("airtel transaction already settled")

// isAirtelSP reports whether a mobileMoneySP value selects the Airtel Money rail.
// MobilePaymentHandler/MobileWithdrawalHandler do NOT normalize mobileMoneySP,
// so this normalization is load-bearing.
func isAirtelSP(sp string) bool {
	s := strings.ToUpper(strings.ReplaceAll(strings.TrimSpace(sp), "-", ""))
	s = strings.ReplaceAll(s, "_", "")
	s = strings.ReplaceAll(s, " ", "")
	return s == "AIRTEL" || s == "AIRTELMONEY"
}

// generateAirtelReference returns a globally-unique, alphanumeric correlation id
// stored in merchantRequestID. Airtel echoes it back as callback.transaction.id.
// A dedicated hex value avoids base64url secureID chars (which may violate
// Airtel's reference rules) and is globally unique (externalId is only
// per-merchant unique).
func generateAirtelReference() string {
	b := make([]byte, 12)
	if _, err := rand.Read(b); err != nil {
		// crypto/rand failure is unrecoverable here; fall back to a timestamp.
		return "MLK" + fmt.Sprintf("%X", time.Now().UnixNano())
	}
	return "MLK" + strings.ToUpper(hex.EncodeToString(b))
}

// handleAirtelCollection runs the Airtel C2B (STK) collection path. It mirrors
// the M-Pesa KES collection: apply the default-paybill amount cap, persist a
// PENDING row, and settle synchronously only if Airtel already returned a
// terminal status (normally it is just an acknowledgement -> callback settles).
func handleAirtelCollection(c *gin.Context, req *MobilePaymentRequest, msisdnStored, secureID string, dateAdded int64) {
	// Airtel amounts are uncapped at the gateway — the single Airtel paybill is
	// our own, so amounts are bounded only by Airtel's per-transaction min/max.
	reference := generateAirtelReference()
	result, err := airtel.InitiateSTKPush(airtel.NormalizeMSISDN(req.PayerPhone), req.Amount, reference)
	if err != nil {
		respondPaymentFailed(c, "airtel collection", err)
		return
	}
	status := result.CollectionStatus()

	newTransaction := &transactions.TransactionModel{
		ImpalaMerchantID:    req.ImpalaMerchantId,
		MerchantRequestID:   reference,
		CheckoutRequestID:   result.ReferenceID,
		ResponseDescription: result.Message,
		ResponseCode:        result.ResponseCode,
		Currency:            "KES",
		Amount:              req.Amount,
		Msisdn:              msisdnStored,
		NetAmount:           float64(req.Amount),
		SecureID:            secureID,
		SourceOfFunds:       "AIRTEL",
		ExternalID:          req.ExternalID,
		CallbackURL:         req.CallbackURL,
		DateAdded:           dateAdded,
		TransactionReport:   "collection",
		TransactionStatus:   "PENDING",
	}
	if err := database.GetConnection().Create(newTransaction).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create transaction", "details": err.Error()})
		return
	}

	// A collection is normally only an acknowledgement (PENDING); settle here
	// only if Airtel already returned a terminal code. Idempotent with the callback.
	if status == "COMPLETE" || status == "FAILED" {
		if serr := settleAirtelTransaction(reference, status, result.AirtelMoneyID, result.Message); serr != nil {
			log.Printf("airtel collection sync-settle failed reference=%s: %v", reference, serr)
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"message":    "Payment initiation successful",
		"externalId": req.ExternalID,
		"secureId":   secureID,
	})
}

// handleAirtelPayout runs the Airtel B2C disbursement path. Airtel disbursements
// may settle synchronously (TS/TF in the response) OR via callback, so a single
// idempotent settle handles both without double-deducting.
func handleAirtelPayout(c *gin.Context, req *MobileWithdrawalRequest, recipientPhone, secureID string, dateAdded int64, balance balances.MerchantBalance) {
	// Airtel payout amounts are uncapped at the gateway (bounded by Airtel's own
	// min/max and the merchant's Airtel balance below).

	// Sufficiency check against the SEPARATE Airtel balance (deduction happens on
	// success settlement). Airtel funds are tracked apart from the M-Pesa KES pool.
	if balance.AirtelBalance < float64(req.Amount) {
		c.JSON(http.StatusOK, gin.H{
			"status":  "FAILED",
			"error":   "INSUFFICIENT_BALANCE",
			"message": fmt.Sprintf("Insufficient Airtel balance. Available: %.2f, Required: %.2f", balance.AirtelBalance, float64(req.Amount)),
		})
		return
	}

	reference := generateAirtelReference()
	result, err := airtel.Disburse(airtel.NormalizeMSISDN(req.RecipientPhone), int(req.Amount), reference, "B2C", "")
	if err != nil {
		respondPaymentFailed(c, "airtel payout", err)
		return
	}
	status := result.DisbursementStatus()

	newTransaction := &transactions.TransactionModel{
		ImpalaMerchantID:    req.ImpalaMerchantId,
		MerchantRequestID:   reference,
		CheckoutRequestID:   result.ReferenceID,
		ResponseDescription: result.Message,
		ResponseCode:        result.ResponseCode,
		Currency:            "KES",
		Amount:              int(req.Amount),
		Msisdn:              recipientPhone,
		NetAmount:           float64(req.Amount),
		SecureID:            secureID,
		SourceOfFunds:       "AIRTEL",
		ExternalID:          req.ExternalID,
		CallbackURL:         req.CallbackURL,
		DateAdded:           dateAdded,
		TransactionReport:   "withdraw",
		TransactionStatus:   "PENDING",
	}
	if err := database.GetConnection().Create(newTransaction).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to record transaction", "details": err.Error()})
		return
	}

	// Disbursements usually settle synchronously; settle now if terminal.
	if status == "COMPLETE" || status == "FAILED" {
		if serr := settleAirtelTransaction(reference, status, result.AirtelMoneyID, result.Message); serr != nil {
			log.Printf("airtel payout sync-settle failed reference=%s: %v", reference, serr)
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"message":       "Payment initiation successful",
		"transactionId": req.ExternalID,
		"secureId":      secureID,
	})
}

// settleAirtelTransaction is the single idempotent settlement used from BOTH the
// synchronous initiate path and the async callback. status must be terminal
// (COMPLETE/FAILED); PENDING is a no-op so a late callback never downgrades a
// settled row. On COMPLETE it credits the collection balance (pay-in) or deducts
// the payout balance (pay-out), inside one DB transaction guarded so it can only
// run once, then fires the signed merchant callback.
func settleAirtelTransaction(reference, status, providerRef, desc string) error {
	if status != "COMPLETE" && status != "FAILED" {
		return nil
	}

	transaction, err := transactions.GetTransactionByMerchantRequestID(reference)
	if err != nil {
		return fmt.Errorf("airtel settle: transaction not found for reference %s: %w", reference, err)
	}

	db := database.GetConnection()
	isPayout := transaction.TransactionReport == "withdraw"

	updates := map[string]interface{}{
		"transactionStatus":   status,
		"callbackStatus":      "PROCESSING",
		"responseDescription": desc,
	}
	if providerRef != "" {
		updates["providerReference"] = providerRef
	}

	err = db.Transaction(func(tx *gorm.DB) error {
		result := tx.Model(&transactions.TransactionModel{}).
			Where("id = ? AND transactionStatus <> ? AND COALESCE(callbackStatus, '') <> ?", transaction.ID, status, "SENT").
			Updates(updates)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return errAirtelAlreadySettled
		}

		if status == "COMPLETE" {
			if isPayout {
				// Airtel Money is tracked on its own balance column (airtelBalance),
				// separate from the M-Pesa KES pool, for clean reconciliation.
				deduct := tx.Model(&balances.MerchantBalance{}).
					Where("impalaMerchantId = ?", transaction.ImpalaMerchantID).
					Updates(map[string]interface{}{
						"airtelBalance": gorm.Expr("airtelBalance - ?", float64(transaction.Amount)),
						"lastUpdated":   time.Now().Unix(),
					})
				if deduct.Error != nil {
					return deduct.Error
				}
				if deduct.RowsAffected == 0 {
					return fmt.Errorf("merchant payout balance not found for %s", transaction.ImpalaMerchantID)
				}
			} else {
				credit := tx.Model(&balances.MerchantCollectionBalance{}).
					Where("impalaMerchantId = ?", transaction.ImpalaMerchantID).
					Update("airtelBalance", gorm.Expr("airtelBalance + ?", transaction.Amount))
				if credit.Error != nil {
					return credit.Error
				}
				if credit.RowsAffected == 0 {
					// No collection-balance row: roll back rather than mark the
					// row COMPLETE with the credit silently lost. It stays PENDING
					// and the reconciler retries once the balance row exists.
					return fmt.Errorf("merchant collection balance not found for %s", transaction.ImpalaMerchantID)
				}
			}
		}
		return nil
	})
	if errors.Is(err, errAirtelAlreadySettled) {
		log.Printf("airtel settle skipped (already settled) reference=%s status=%s", reference, status)
		return nil
	}
	if err != nil {
		return err
	}

	if providerRef != "" {
		transaction.ProviderReference = providerRef
	}
	callbackResponse := buildMpesaMerchantCallback(&transaction, status, desc, transaction.Amount, providerRef)
	callbackErr := SendCallback(transaction.ID, callbackResponse)
	newCallbackStatus := "SENT"
	if callbackErr != nil {
		newCallbackStatus = "FAILED"
		log.Printf("airtel settle: merchant callback failed reference=%s: %v", reference, callbackErr)
	}
	db.Model(&transactions.TransactionModel{}).
		Where("id = ?", transaction.ID).
		Update("callbackStatus", newCallbackStatus)

	return callbackErr
}

// AirtelCallbackHandler authenticates and settles an inbound Airtel callback.
// It is registered for BOTH the collection and disbursement callback URLs;
// collection-vs-payout is decided from the looked-up transaction, so it works
// regardless of which URL Airtel posts to. Fail-closed: a missing/invalid hash
// is rejected with 401 (a collections callback credits money).
func AirtelCallbackHandler(c *gin.Context) {
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "could not read callback body"})
		return
	}

	verified, verr := airtel.VerifyCallbackHash(body)
	if verr != nil || !verified {
		log.Printf("airtel callback rejected: verified=%v err=%v", verified, verr)
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid callback signature"})
		return
	}

	cb, perr := airtel.ParseCallback(body)
	if perr != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid callback body"})
		return
	}
	reference := strings.TrimSpace(cb.Transaction.ID)
	if reference == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "callback missing transaction id"})
		return
	}

	status := airtel.MapStatus(cb.Transaction.StatusCode)
	if status == "PENDING" {
		// Non-terminal callback: acknowledge, never downgrade a settled row.
		c.JSON(http.StatusOK, gin.H{"status": "SUCCESS", "message": "callback received"})
		return
	}

	if serr := settleAirtelTransaction(reference, status, cb.Transaction.AirtelMoneyID, cb.Transaction.Message); serr != nil {
		// Reference not in THIS instance's DB? If a peer is configured (e.g. prod
		// forwarding to the sandbox, since the live Airtel app posts all callbacks
		// to the one registered prod URL), forward the raw callback there. The peer
		// re-verifies the hash from the body and settles its own row. Env-gated and
		// only for not-found, so prod without the var is unchanged and there is no
		// loop (the peer leaves AIRTEL_CALLBACK_FORWARD_URL unset).
		if fwd := strings.TrimSpace(os.Getenv("AIRTEL_CALLBACK_FORWARD_URL")); fwd != "" && errors.Is(serr, gorm.ErrRecordNotFound) {
			if forwardAirtelCallback(fwd, c.Request.URL.Path, body) {
				c.JSON(http.StatusOK, gin.H{"status": "SUCCESS", "message": "callback forwarded"})
				return
			}
		}
		// Unknown reference or DB error — 404 so Airtel retries (logged loudly).
		log.Printf("airtel callback settle failed reference=%s: %v", reference, serr)
		c.JSON(http.StatusNotFound, gin.H{"error": "transaction not found or not settled"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "SUCCESS", "message": "callback processed"})
}

// forwardAirtelCallback re-POSTs a raw Airtel callback body to a peer instance
// (baseURL, no trailing slash) preserving the request path. Best-effort. The peer
// re-verifies the hash from the body itself, so no headers need to be copied.
// Returns true on a 2xx from the peer.
func forwardAirtelCallback(baseURL, path string, body []byte) bool {
	url := strings.TrimRight(baseURL, "/") + path
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		log.Printf("airtel callback forward: build request failed url=%s: %v", url, err)
		return false
	}
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("airtel callback forward failed url=%s: %v", url, err)
		return false
	}
	defer resp.Body.Close()
	ok := resp.StatusCode >= http.StatusOK && resp.StatusCode < http.StatusMultipleChoices
	log.Printf("airtel callback forwarded url=%s http_status=%d ok=%v", url, resp.StatusCode, ok)
	return ok
}
