package drawings

import (
	"com.mam-laka/database"
	"github.com/gin-gonic/gin"
)

// WithdrawalRequestModelValidator represents a validator for WithdrawalRequest operations.
type WithdrawalRequestModelValidator struct {
	WithdrawalRequest struct {
		ImpalaMerchantID string  `form:"impalaMerchantId" json:"impalaMerchantId" binding:"required,min=2"`
		RequestedBy      int     `form:"requestedBy" json:"requestedBy" binding:"required"`
		ApprovedBy       *int    `form:"approvedBy" json:"approvedBy" binding:"required"`
		Amount           float64 `form:"amount" json:"amount" binding:"required,gt=0"` // Changed to float64
		TransferType     string  `form:"transferType" json:"transferType" binding:"required,oneof=withdraw transfer"`
		Status           string  `form:"status" json:"status" binding:"required"`
		DateRequested    int64   `form:"dateRequested" json:"dateRequested" `
		DateApproved     *int64  `form:"dateApproved" json:"dateApproved"`
		Comment          string  `form:"comment" json:"comment"`
	} `json:"request"`
	WithdrawalRequestModel WithdrawalRequestModel `json:"-"`
}

// Bind binds the request data to the WithdrawalRequestModelValidator.
func (v *WithdrawalRequestModelValidator) Bind(c *gin.Context) error {
	err := database.Bind(c, v)
	if err != nil {
		return err
	}

	v.WithdrawalRequestModel.ImpalaMerchantID = v.WithdrawalRequest.ImpalaMerchantID
	v.WithdrawalRequestModel.RequestedBy = v.WithdrawalRequest.RequestedBy
	v.WithdrawalRequestModel.ApprovedBy = v.WithdrawalRequest.ApprovedBy
	v.WithdrawalRequestModel.Amount = v.WithdrawalRequest.Amount
	v.WithdrawalRequestModel.TransferType = v.WithdrawalRequest.TransferType
	v.WithdrawalRequestModel.Status = v.WithdrawalRequest.Status
	v.WithdrawalRequestModel.DateRequested = v.WithdrawalRequest.DateRequested // Now correctly assigning the value
	v.WithdrawalRequestModel.DateApproved = v.WithdrawalRequest.DateApproved
	v.WithdrawalRequestModel.Comment = v.WithdrawalRequest.Comment

	// Check values after assignment

	return nil
}

// NewWithdrawalRequestModelValidator creates a new WithdrawalRequestModelValidator instance.
func NewWithdrawalRequestModelValidator() WithdrawalRequestModelValidator {
	return WithdrawalRequestModelValidator{}
}

// NewWithdrawalRequestModelValidatorFillWith creates a new WithdrawalRequestModelValidator instance and fills it with WithdrawalRequest model data.
func NewWithdrawalRequestModelValidatorFillWith(WithdrawalRequestModel WithdrawalRequestModel) WithdrawalRequestModelValidator {
	return WithdrawalRequestModelValidator{
		WithdrawalRequest: struct {
			ImpalaMerchantID string  `form:"impalaMerchantId" json:"impalaMerchantId" binding:"required,min=2"`
			RequestedBy      int     `form:"requestedBy" json:"requestedBy" binding:"required"`
			ApprovedBy       *int    `form:"approvedBy" json:"approvedBy" binding:"required"`
			Amount           float64 `form:"amount" json:"amount" binding:"required,gt=0"` // Changed to float64
			TransferType     string  `form:"transferType" json:"transferType" binding:"required,oneof=withdraw transfer"`
			Status           string  `form:"status" json:"status" binding:"required"`
			DateRequested    int64   `form:"dateRequested" json:"dateRequested" `
			DateApproved     *int64  `form:"dateApproved" json:"dateApproved"`
			Comment          string  `form:"comment" json:"comment"`
		}{
			ImpalaMerchantID: WithdrawalRequestModel.ImpalaMerchantID,
			RequestedBy:      WithdrawalRequestModel.RequestedBy,
			ApprovedBy:       WithdrawalRequestModel.ApprovedBy,
			Amount:           WithdrawalRequestModel.Amount,
			TransferType:     WithdrawalRequestModel.TransferType,
			Status:           WithdrawalRequestModel.Status,
			DateRequested:    WithdrawalRequestModel.DateRequested,
			DateApproved:     WithdrawalRequestModel.DateApproved,
			Comment:          WithdrawalRequestModel.Comment,
		},
	}
}
