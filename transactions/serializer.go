package transactions

import (
	"github.com/gin-gonic/gin"
	"strings"
)

type TransactionSerializer struct {
	c           *gin.Context
	Transaction TransactionModel
}

func NewTransactionSerializer(c *gin.Context, transaction TransactionModel) *TransactionSerializer {
	return &TransactionSerializer{c: c, Transaction: transaction}
}

func (s *TransactionSerializer) Response() map[string]interface{} {
	status := s.Transaction.TransactionStatus
	normalizedStatus := strings.ToUpper(strings.TrimSpace(status))
	if normalizedStatus == "SUCCESS" || normalizedStatus == "COMPLETE" {
		status = "COMPLETED"
	}

	return map[string]interface{}{
		// "id":               s.Transaction.ID,
		"impalaMerchantId": s.Transaction.ImpalaMerchantID,
		"transaction_status": status,
		"transaction_report": s.Transaction.TransactionReport,
		"currency":           s.Transaction.Currency,
		"amount":             s.Transaction.Amount,
		// "msisdn":               s.Transaction.Msisdn,
		// "net_amount":           s.Transaction.NetAmount,
		"secure_id":       s.Transaction.SecureID,
		"external_id":     s.Transaction.ExternalID,
		"callback_url":    s.Transaction.CallbackURL,
		// "redirect_url":         s.Transaction.RedirectURL,
		"date_added": s.Transaction.DateAdded,
		// "merchant_request_id": s.Transaction.MerchantRequestID,
		// "checkout_request_id": s.Transaction.CheckoutRequestID,
		// "response_code":        s.Transaction.ResponseCode,
		// "response_description": s.Transaction.ResponseDescription,
		// "callback_status": s.Transaction.CallbackStatus,
	}
}

type TransactionSerializerList struct {
	c            *gin.Context
	Transactions []TransactionModel
}

func NewTransactionSerializerList(c *gin.Context, transactions []TransactionModel) *TransactionSerializerList {
	return &TransactionSerializerList{c: c, Transactions: transactions}
}

func (s *TransactionSerializerList) Response() []map[string]interface{} {
	var responseList []map[string]interface{}
	for _, transaction := range s.Transactions {
		serializer := NewTransactionSerializer(s.c, transaction)
		responseList = append(responseList, serializer.Response())
	}
	return responseList
}
