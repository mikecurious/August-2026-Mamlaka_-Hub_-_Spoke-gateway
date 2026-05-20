package merchants

import (
	"log"
	"strings"

	"com.mam-laka/westafrica"

	"github.com/gin-gonic/gin"
)

const (
	MsgPaymentFailed          = "Payment could not be completed. Please check your details and try again."
	MsgServiceUnavailable     = "Payment service is temporarily unavailable. Please try again later."
	MsgInternalError          = "Request could not be completed. Please try again or contact support."
)

// isSafeClientMessage returns true only for short, non-technical provider messages safe to show merchants.
func isSafeClientMessage(msg string) bool {
	msg = strings.TrimSpace(msg)
	if msg == "" || len(msg) > 256 {
		return false
	}
	lower := strings.ToLower(msg)
	unsafe := []string{
		"{", "}", "[", "]",
		"api_key", "api-key", "x-app-id", "authorization",
		"pix_", "cbapi_", "password", "secret", "token",
		"http error", "unmarshal", "tls:", "connection refused",
		"failed to ", "dial tcp", "no such host", "stack",
		"sql", "database", "gorm", "panic",
	}
	for _, p := range unsafe {
		if strings.Contains(lower, p) {
			return false
		}
	}
	return true
}

// ClientProviderMessage returns a merchant-safe message; never exposes keys or internal errors.
func ClientProviderMessage(err error) string {
	if err == nil {
		return MsgPaymentFailed
	}
	for _, candidate := range []string{westafrica.PublicError(err), err.Error()} {
		if isSafeClientMessage(candidate) {
			return candidate
		}
	}
	return MsgPaymentFailed
}

// respondPaymentFailed logs the real error server-side and returns a safe JSON body (no details field).
func respondPaymentFailed(c *gin.Context, logPrefix string, err error) {
	if err != nil {
		log.Printf("%s: %v", logPrefix, err)
	}
	c.JSON(502, gin.H{
		"error":   "Payment initiation failed",
		"message": ClientProviderMessage(err),
	})
}

// respondServiceUnavailable is used when provider credentials are missing or misconfigured.
func respondServiceUnavailable(c *gin.Context, logPrefix string, err error) {
	if err != nil {
		log.Printf("%s: %v", logPrefix, err)
	}
	c.JSON(503, gin.H{
		"error":   "Payment initiation failed",
		"message": MsgServiceUnavailable,
	})
}
