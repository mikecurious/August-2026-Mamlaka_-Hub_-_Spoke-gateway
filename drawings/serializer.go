package drawings

import (
	"github.com/gin-gonic/gin"
)

type WithdrawalRequestSerializer struct {
	c                      *gin.Context
	WithdrawalRequestModel WithdrawalRequestModel
}

func NewWithdrawalRequestSerializer(c *gin.Context, withdrawalRequest WithdrawalRequestModel) *WithdrawalRequestSerializer {
	return &WithdrawalRequestSerializer{c: c, WithdrawalRequestModel: withdrawalRequest}
}

func (s *WithdrawalRequestSerializer) Response() map[string]interface{} {
	return map[string]interface{}{
		"id":               s.WithdrawalRequestModel.ID,
		"impalaMerchantId": s.WithdrawalRequestModel.ImpalaMerchantID,
		"requestedBy":      s.WithdrawalRequestModel.RequestedBy,
		"approvedBy":       s.WithdrawalRequestModel.ApprovedBy,
		"amount":           s.WithdrawalRequestModel.Amount,
		"status":           s.WithdrawalRequestModel.Status,
		"transferType":     s.WithdrawalRequestModel.TransferType, // Added transferType
		"dateRequested":    s.WithdrawalRequestModel.DateRequested,
		"dateApproved":     s.WithdrawalRequestModel.DateApproved,
		"comment":          s.WithdrawalRequestModel.Comment,
	}
}

type WithdrawalRequestSerializerList struct {
	c                  *gin.Context
	WithdrawalRequests []WithdrawalRequestModel
}

func NewWithdrawalRequestSerializerList(c *gin.Context, withdrawalRequests []WithdrawalRequestModel) *WithdrawalRequestSerializerList {
	return &WithdrawalRequestSerializerList{c: c, WithdrawalRequests: withdrawalRequests}
}

func (s *WithdrawalRequestSerializerList) Response() []map[string]interface{} {
	var responseList []map[string]interface{}
	for _, request := range s.WithdrawalRequests {
		serializer := NewWithdrawalRequestSerializer(s.c, request)
		responseList = append(responseList, serializer.Response())
	}
	return responseList
}
