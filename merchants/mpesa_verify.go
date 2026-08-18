package merchants

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"com.mam-laka/mpesa"
	"com.mam-laka/transactions"
	"github.com/gin-gonic/gin"
)

// MpesaIdentifierVerifyHandler looks up a till or paybill via Safaricom SFC verify and records a zero-fee transaction.
func MpesaIdentifierVerifyHandler(c *gin.Context) {
	merchantIDVal, ok := c.Get("merchantID")
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Missing authentication"})
		return
	}
	merchantID, ok := merchantIDVal.(string)
	if !ok || strings.TrimSpace(merchantID) == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid merchant context"})
		return
	}

	var req MpesaIdentifierVerifyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request", "details": err.Error()})
		return
	}

	identifier := strings.TrimSpace(req.Identifier)
	if identifier == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "identifier is required"})
		return
	}

	identifierType, err := mpesa.ResolveIdentifierType(req.Type)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	externalID := strings.TrimSpace(req.ExternalID)
	if externalID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "externalId is required"})
		return
	}

	exists, err := transactions.ExternalIDExists(merchantID, externalID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to validate externalId", "details": err.Error()})
		return
	}
	if exists {
		c.JSON(http.StatusConflict, gin.H{"error": "DUPLICATE_EXTERNAL_ID", "message": "A transaction with this externalId already exists"})
		return
	}

	creds, credErr := mpesa.ResolveC2BCredentials(merchantID)
	if credErr != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "M-Pesa credentials unavailable", "message": credErr.Error()})
		return
	}
	verifyResp, err := mpesa.QueryIdentifierInfo(creds.ConsumerKey, creds.ConsumerSecret, identifierType, identifier)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "Identifier lookup failed", "message": err.Error()})
		return
	}

	secureID := transactions.GenerateSecureID()
	kind := strings.ToLower(strings.TrimSpace(req.Type))
	if kind == "2" {
		kind = "till"
	}
	if kind == "4" {
		kind = "paybill"
	}

	txnStatus := "FAILED"
	if verifyResp.IsSuccess() {
		txnStatus = "COMPLETE"
	}

	orgName := strings.TrimSpace(verifyResp.OrganizationName)
	respDesc := orgName
	if respDesc == "" {
		respDesc = strings.TrimSpace(verifyResp.ResponseMessage)
	}

	sourceOfFunds := "MPESA_TILL_VERIFY"
	if identifierType == mpesa.IdentifierTypePaybill {
		sourceOfFunds = "MPESA_PAYBILL_VERIFY"
	}

	verifyJSON, _ := json.Marshal(verifyResp)

	txn := &transactions.TransactionModel{
		ImpalaMerchantID:    merchantID,
		TransactionStatus:   txnStatus,
		TransactionReport:   "verification",
		Currency:            "KES",
		Amount:              0,
		NetAmount:           0,
		Msisdn:              identifier,
		SecureID:            secureID,
		SourceOfFunds:       sourceOfFunds,
		ExternalID:          externalID,
		CallbackURL:         strings.TrimSpace(req.CallbackURL),
		DateAdded:           time.Now().Unix(),
		MerchantRequestID:   verifyResp.ConversationID,
		CheckoutRequestID:   verifyResp.DisplayNumber(),
		ResponseCode:        verifyResp.ResponseCode,
		ResponseDescription: respDesc,
		CallbackStatus:      "NOT_SENT",
		ProviderReference:   string(verifyJSON),
	}

	if err := transactions.SaveTransaction(txn); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to record transaction", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":           txnStatus,
		"message":          fmt.Sprintf("%s lookup completed", kind),
		"secureId":         secureID,
		"externalId":       externalID,
		"transactionId":    fmt.Sprintf("ImpadlTdest%d", txn.ID),
		"type":             kind,
		"identifier":       identifier,
		"identifierType":   identifierType,
		"organizationName": orgName,
		"chargeProfileId":  verifyResp.ChargeProfileID,
		"conversationId":   verifyResp.ConversationID,
		"responseCode":     verifyResp.ResponseCode,
		"responseMessage":  verifyResp.ResponseMessage,
		"detailedMessage":  verifyResp.DetailedMessage,
		"tillNumber":       verifyResp.TillNumber,
		"paybillNumber":    verifyResp.OrganizationShortCode,
		"fee":              0,
	})
}
