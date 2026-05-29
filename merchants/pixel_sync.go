package merchants

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"com.mam-laka/database"
	"com.mam-laka/transactions"
	"com.mam-laka/westafrica"
	"github.com/gin-gonic/gin"
)

const (
	pixelSyncLogFile     = "pixel_sync.log"
	pixelNotFoundLogFile = "pixel_not_found.log"
)

// PixelPendingSyncResult summarizes one cron/sync run.
type PixelPendingSyncResult struct {
	Scanned      int      `json:"scanned"`
	Completed    int      `json:"completed"`
	Failed       int      `json:"failed"`
	StillPending int      `json:"stillPending"`
	NotFound     int      `json:"notFound"`
	Skipped      int      `json:"skipped"`
	Errors       []string `json:"errors,omitempty"`
}

type pixelSyncRequest struct {
	Limit          int `json:"limit"`
	MinAgeSeconds  int `json:"minAgeSeconds"`
}

func appendPixelLog(filename, line string) {
	f, err := os.OpenFile(filename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Printf("failed to open %s: %v", filename, err)
		return
	}
	defer f.Close()
	_, _ = f.WriteString(line + "\n")
}

func logPixelSync(callerIP, payload, message string, data interface{}) {
	dataJSON := ""
	if data != nil {
		if b, err := json.Marshal(data); err == nil {
			dataJSON = string(b)
		}
	}
	line := fmt.Sprintf("%s ip=%s payload=%s message=%s data=%s",
		time.Now().Format(time.RFC3339), callerIP, payload, message, dataJSON)
	appendPixelLog(pixelSyncLogFile, line)
}

func logPixelNotFound(merchantRequestID, secureID, currency, impalaMerchantID, message string) {
	line := fmt.Sprintf("%s merchantRequestID=%s secureId=%s currency=%s impalaMerchantId=%s message=%s",
		time.Now().Format(time.RFC3339),
		merchantRequestID, secureID, currency, impalaMerchantID, message,
	)
	appendPixelLog(pixelNotFoundLogFile, line)
}

func requirePixelSyncCronSecret(c *gin.Context) bool {
	secret := strings.TrimSpace(os.Getenv("PIXEL_SYNC_CRON_SECRET"))
	if secret == "" {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error":   "SYNC_NOT_CONFIGURED",
			"message": "PIXEL_SYNC_CRON_SECRET is not set on the server",
		})
		return false
	}

	auth := strings.TrimSpace(c.GetHeader("Authorization"))
	if strings.HasPrefix(auth, "Bearer ") {
		auth = strings.TrimSpace(strings.TrimPrefix(auth, "Bearer "))
	}
	if auth == secret || strings.TrimSpace(c.GetHeader("X-Cron-Secret")) == secret {
		return true
	}

	c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
	return false
}

func pixelProviderTransactionID(txn *transactions.TransactionModel) string {
	ref := strings.TrimSpace(txn.MerchantRequestID)
	if strings.HasPrefix(strings.ToUpper(ref), "PIX_") {
		return ref
	}
	ref = strings.TrimSpace(txn.CheckoutRequestID)
	if strings.HasPrefix(strings.ToUpper(ref), "PIX_") {
		return ref
	}
	return ""
}

func airtimeCallbackFromPixelStatus(txn *transactions.TransactionModel, data *westafrica.AirtimeResponse) *AirtimeCallbackRequest {
	d := data.Data
	currency := strings.TrimSpace(d.Currency)
	if currency == "" {
		currency = strings.ToUpper(strings.TrimSpace(txn.Currency))
	}
	return &AirtimeCallbackRequest{
		TransactionID: d.TransactionID,
		Amount:        d.Amount,
		Benefice:      d.Benefice,
		Commission:    int(d.Commission),
		Destination:   d.Destination,
		Fee:           int(d.Fee),
		Response:      d.Response,
		Error:         d.Error,
		ServiceID:     d.ServiceID,
		CustomerName:  d.CustomerName,
		State:         d.State,
		CustomData:    d.CustomData,
		IPNUrl:        d.IPNUrl,
		ProviderID:    d.ProviderID,
		Currency:      currency,
	}
}

func queryPixelStatusWithKeyFallback(ctx context.Context, currency, pixelID string) (*westafrica.AirtimeResponse, string, error) {
	keys := westafrica.PixelAPIKeysForCurrency(currency)
	if len(keys) == 0 {
		return nil, "", fmt.Errorf("no Pixel API keys configured for %s", currency)
	}

	client := westafrica.NewAirtimeClient()
	var lastErr error
	for _, key := range keys {
		resp, err := client.QueryTransactionStatus(ctx, key, pixelID)
		if err == nil {
			return resp, key, nil
		}
		if errors.Is(err, westafrica.ErrPixelTransactionNotFound) {
			lastErr = err
			continue
		}
		return nil, key, err
	}
	return nil, "", lastErr
}

// SyncPendingPixelWestAfricaTransactions polls Pixel for pending XOF/XAF/GMD transactions and finalizes them.
func SyncPendingPixelWestAfricaTransactions(callerIP, requestPayload string, limit, minAgeSeconds int) PixelPendingSyncResult {
	result := PixelPendingSyncResult{}
	logPixelSync(callerIP, requestPayload, "sync_started", nil)

	if limit <= 0 {
		limit = 100
	}
	if limit > 500 {
		limit = 500
	}
	if minAgeSeconds < 0 {
		minAgeSeconds = 0
	}

	db := database.GetConnection()
	var pending []transactions.TransactionModel
	q := db.Where("currency IN ?", []string{"XOF", "XAF", "GMD"}).
		Where("transactionStatus IN ?", []string{"PENDING", "pending"}).
		Where("merchantRequestID LIKE ?", "PIX_%").
		Order("id ASC").
		Limit(limit)

	if minAgeSeconds > 0 {
		cutoff := time.Now().Add(-time.Duration(minAgeSeconds) * time.Second).Unix()
		q = q.Where("dateAdded <= ?", cutoff)
	}

	if err := q.Find(&pending).Error; err != nil {
		msg := fmt.Sprintf("query failed: %v", err)
		result.Errors = append(result.Errors, msg)
		logPixelSync(callerIP, requestPayload, msg, nil)
		return result
	}

	result.Scanned = len(pending)
	reqCtx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	for i := range pending {
		txn := pending[i]

		var fresh transactions.TransactionModel
		if err := db.First(&fresh, txn.ID).Error; err != nil {
			result.Skipped++
			continue
		}
		if !isPixelPendingStatus(fresh.TransactionStatus) {
			result.Skipped++
			continue
		}
		txn = fresh

		pixelID := pixelProviderTransactionID(&txn)
		if pixelID == "" {
			result.Skipped++
			logPixelSync(callerIP, requestPayload, fmt.Sprintf("skip_no_pixel_id id=%d secureId=%s", txn.ID, txn.SecureID), nil)
			continue
		}

		statusResp, keyUsed, err := queryPixelStatusWithKeyFallback(reqCtx, txn.Currency, pixelID)
		if err != nil {
			if errors.Is(err, westafrica.ErrPixelTransactionNotFound) {
				result.NotFound++
				logPixelNotFound(pixelID, txn.SecureID, txn.Currency, txn.ImpalaMerchantID, "transaction not found on Pixel (tried all keys)")
				logPixelSync(callerIP, requestPayload, fmt.Sprintf("not_found id=%d pixelId=%s", txn.ID, pixelID), nil)
				continue
			}
			msg := fmt.Sprintf("status_check_failed id=%d pixelId=%s: %v", txn.ID, pixelID, err)
			result.Errors = append(result.Errors, msg)
			logPixelSync(callerIP, requestPayload, msg, nil)
			continue
		}

		state := strings.ToUpper(strings.TrimSpace(statusResp.Data.State))
		logPixelSync(callerIP, requestPayload, fmt.Sprintf("status id=%d pixelId=%s state=%s key=***", txn.ID, pixelID, state), gin.H{
			"statut_code": statusResp.StatusCode,
			"message":     statusResp.Message,
		})

		switch state {
		case "SUCCESSFUL", "SUCCESS", "COMPLETED":
			callback := airtimeCallbackFromPixelStatus(&txn, statusResp)
			if err := processSuccessfulTransaction(db, &txn, callback); err != nil {
				msg := fmt.Sprintf("finalize_success_failed id=%d: %v", txn.ID, err)
				result.Errors = append(result.Errors, msg)
				logPixelSync(callerIP, requestPayload, msg, nil)
				continue
			}
			result.Completed++
			logPixelSync(callerIP, requestPayload, fmt.Sprintf("completed id=%d pixelId=%s keySuffix=%s", txn.ID, pixelID, keySuffix(keyUsed)), nil)

		case "FAILED", "FAILURE", "CANCELLED", "CANCELED":
			callback := airtimeCallbackFromPixelStatus(&txn, statusResp)
			if err := processFailedTransaction(db, &txn, callback); err != nil {
				msg := fmt.Sprintf("finalize_failed_failed id=%d: %v", txn.ID, err)
				result.Errors = append(result.Errors, msg)
				logPixelSync(callerIP, requestPayload, msg, nil)
				continue
			}
			result.Failed++
			logPixelSync(callerIP, requestPayload, fmt.Sprintf("marked_failed id=%d pixelId=%s", txn.ID, pixelID), nil)

		default:
			result.StillPending++
			logPixelSync(callerIP, requestPayload, fmt.Sprintf("still_pending id=%d pixelId=%s state=%s", txn.ID, pixelID, state), nil)
		}
	}

	logPixelSync(callerIP, requestPayload, "sync_completed", result)
	return result
}

func keySuffix(apiKey string) string {
	apiKey = strings.TrimSpace(apiKey)
	if len(apiKey) <= 8 {
		return apiKey
	}
	return apiKey[len(apiKey)-8:]
}

// SyncPendingPixelTransactionsHandler is the authenticated cron endpoint for Pixel status polling.
func SyncPendingPixelTransactionsHandler(c *gin.Context) {
	if !requirePixelSyncCronSecret(c) {
		return
	}

	raw, _ := c.GetRawData()
	payload := strings.TrimSpace(string(raw))
	if payload == "" {
		payload = "{}"
	}

	limit := 100
	minAgeSeconds := 60
	if payload != "{}" {
		var req pixelSyncRequest
		if err := json.Unmarshal([]byte(payload), &req); err == nil {
			if req.Limit > 0 {
				limit = req.Limit
			}
			if req.MinAgeSeconds > 0 {
				minAgeSeconds = req.MinAgeSeconds
			}
		}
	}
	if q := strings.TrimSpace(c.Query("limit")); q != "" {
		if n, err := parsePositiveInt(q); err == nil {
			limit = n
		}
	}
	if q := strings.TrimSpace(c.Query("minAgeSeconds")); q != "" {
		if n, err := parsePositiveInt(q); err == nil {
			minAgeSeconds = n
		}
	}

	result := SyncPendingPixelWestAfricaTransactions(c.ClientIP(), payload, limit, minAgeSeconds)
	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "Pixel pending transaction sync completed",
		"result":  result,
	})
}

func isPixelPendingStatus(status string) bool {
	switch strings.ToUpper(strings.TrimSpace(status)) {
	case "PENDING", "PENDING1", "PROCESSING":
		return true
	default:
		return false
	}
}

func parsePositiveInt(s string) (int, error) {
	var n int
	_, err := fmt.Sscanf(s, "%d", &n)
	if err != nil || n <= 0 {
		return 0, fmt.Errorf("invalid int")
	}
	return n, nil
}
