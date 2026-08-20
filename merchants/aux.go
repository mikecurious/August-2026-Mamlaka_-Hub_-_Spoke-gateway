package merchants

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"com.mam-laka/transactions"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

var merchantCallbackHTTPClient = &http.Client{
	Timeout: 10 * time.Second,
	Transport: &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   3 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		MaxIdleConns:          200,
		MaxIdleConnsPerHost:   50,
		MaxConnsPerHost:       50,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   3 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	},
}

// buildMpesaMerchantCallback is the standard webhook payload for KES M-Pesa pay-in/payout.
func buildMpesaMerchantCallback(tx *transactions.TransactionModel, transactionStatus, transactionReport string, amount int, reference string) map[string]interface{} {
	payload := map[string]interface{}{
		"amount":            amount,
		"currency":          tx.Currency,
		"externalId":        tx.ExternalID,
		"secureId":          tx.SecureID,
		"transactionReport": transactionReport,
		"transactionStatus": transactionStatus,
		"environment":       tx.EnvironmentLabel(),
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

func recipientNameFromB2CMetadata(metadata map[string]interface{}) string {
	publicName := strings.TrimSpace(metadataValueString(metadata["ReceiverPartyPublicName"]))
	if publicName == "" || publicName == "<nil>" {
		return ""
	}

	parts := strings.SplitN(publicName, " - ", 2)
	if len(parts) == 2 {
		return strings.TrimSpace(parts[1])
	}

	return publicName
}

// SendCallback now accepts the callbackBody.Body type directly
// callbackSigningSecret returns the HMAC secret used to sign outbound callbacks
// to a merchant. A per-merchant secret CALLBACK_SIGNING_SECRET_<MERCHANT>
// (merchant id upper-cased, non-alphanumerics -> _) takes precedence, falling
// back to the platform-wide CALLBACK_SIGNING_SECRET. Returns "" when no secret
// is configured, in which case the callback is sent unsigned (backwards
// compatible with merchants not yet verifying).
func callbackSigningSecret(merchantID string) string {
	up := strings.Map(func(r rune) rune {
		switch {
		case r >= 'A' && r <= 'Z', r >= '0' && r <= '9':
			return r
		case r >= 'a' && r <= 'z':
			return r - 32
		default:
			return '_'
		}
	}, merchantID)
	if v := strings.TrimSpace(os.Getenv("CALLBACK_SIGNING_SECRET_" + up)); v != "" {
		return v
	}
	return strings.TrimSpace(os.Getenv("CALLBACK_SIGNING_SECRET"))
}

// stampEnvironment adds the creating environment to an outbound merchant
// callback body. Maps are updated in place; anything else (a struct payload) is
// round-tripped through JSON so the field can be added. A body that is not a
// JSON object, or one that already carries the field, is returned unchanged.
func stampEnvironment(callbackBody interface{}, env string) interface{} {
	switch b := callbackBody.(type) {
	case map[string]interface{}:
		if _, ok := b["environment"]; !ok {
			b["environment"] = env
		}
		return b
	case gin.H:
		if _, ok := b["environment"]; !ok {
			b["environment"] = env
		}
		return b
	}

	raw, err := json.Marshal(callbackBody)
	if err != nil {
		return callbackBody
	}
	var m map[string]interface{}
	if err := json.Unmarshal(raw, &m); err != nil || m == nil {
		return callbackBody
	}
	if _, ok := m["environment"]; !ok {
		m["environment"] = env
	}
	return m
}

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

	// Every rail's merchant webhook goes out through this function, so stamping
	// the environment here covers all of them rather than each payload builder.
	callbackBody = stampEnvironment(callbackBody, transaction.EnvironmentLabel())

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
	req, err := http.NewRequest(http.MethodPost, transaction.CallbackURL, bytes.NewBuffer(responseBody))
	if err != nil {
		return fmt.Errorf("failed to build callback request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	// Sign the exact bytes so the merchant can verify the callback is genuinely
	// from us: X-Mamlaka-Signature: sha256=<hex(HMAC-SHA256(body, secret))>.
	if secret := callbackSigningSecret(transaction.ImpalaMerchantID); secret != "" {
		mac := hmac.New(sha256.New, []byte(secret))
		mac.Write(responseBody)
		req.Header.Set("X-Mamlaka-Signature", "sha256="+hex.EncodeToString(mac.Sum(nil)))
	}
	resp, err := merchantCallbackHTTPClient.Do(req)
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
			"transactionStatus": transactions.NormalizeTransactionState(status),
			"callbackStatus":    callbackStatus,
		}).Error
}
