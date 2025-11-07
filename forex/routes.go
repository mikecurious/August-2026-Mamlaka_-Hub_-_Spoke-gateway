package forex

import (
	"errors"
	"math"
	"net/http"
	"strings"

	"com.mam-laka/balances"
	"com.mam-laka/database"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
)

// RegisterRoutes registers forex-related routes with the router.
func RegisterRoutes(router *gin.RouterGroup) {
	router.GET("/forex/rates", ListForexRatesHandler)
	router.POST("/forex/convert", ConvertCurrencyHandler)
}

// ListForexRatesHandler handles GET requests to list all forex rates.
func ListForexRatesHandler(c *gin.Context) {
	rates, err := GetAllForexRates()
	if err != nil {
		c.JSON(http.StatusInternalServerError, database.NewError("forex", err))
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"rates":  rates,
		"count":  len(rates),
	})
}

// ConvertCurrencyRequest represents the request body for currency conversion.
type ConvertCurrencyRequest struct {
	ImpalaMerchantID string  `json:"impalaMerchantId" binding:"required"`
	AccountType      string  `json:"accountType" binding:"required,oneof=collection payout"`
	FromCurrency     string  `json:"fromCurrency" binding:"required"`
	ToCurrency       string  `json:"toCurrency" binding:"required"`
	Amount           float64 `json:"amount" binding:"required,gt=0"`
}

// ConvertCurrencyHandler handles POST requests to convert currency.
func ConvertCurrencyHandler(c *gin.Context) {
	var req ConvertCurrencyRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		var validationErr validator.ValidationErrors
		if errors.As(err, &validationErr) {
			c.JSON(http.StatusBadRequest, database.NewValidatorError(err))
			return
		}
		c.JSON(http.StatusBadRequest, database.NewError("forex", err))
		return
	}

	req.FromCurrency = strings.ToUpper(strings.TrimSpace(req.FromCurrency))
	req.ToCurrency = strings.ToUpper(strings.TrimSpace(req.ToCurrency))
	req.AccountType = strings.ToLower(strings.TrimSpace(req.AccountType))

	convertedAmount, err := ConvertCurrency(req.FromCurrency, req.ToCurrency, req.Amount)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "CONVERSION_ERROR",
			"message": err.Error(),
		})
		return
	}

	if convertedAmount < 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "CONVERSION_ERROR",
			"message": "invalid conversion result",
		})
		return
	}

	fromRate, err1 := GetForexRateByCurrency(req.FromCurrency)
	toRate, err2 := GetForexRateByCurrency(req.ToCurrency)

	if err1 != nil || err2 != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "MISSING_RATE",
			"message": "missing forex rate for one of the currencies",
		})
		return
	}

	if req.Amount == 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "INVALID_AMOUNT",
			"message": "amount cannot be zero",
		})
		return
	}

	rateMultiplier := convertedAmount / req.Amount
	if math.IsNaN(rateMultiplier) || math.IsInf(rateMultiplier, 0) {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "INVALID_RATE",
			"message": "calculated exchange rate is invalid",
		})
		return
	}

	var updateErr error
	var updatedBalance interface{}

	switch req.AccountType {
	case "payout":
		updateErr = balances.ConvertBalance(req.ImpalaMerchantID, req.FromCurrency, req.ToCurrency, req.Amount, rateMultiplier)
		if updateErr == nil {
			updatedBalance, updateErr = balances.GetMerchantBalance(req.ImpalaMerchantID)
		}
	case "collection":
		updateErr = balances.ConvertCollectionBalance(req.ImpalaMerchantID, req.FromCurrency, req.ToCurrency, req.Amount, rateMultiplier)
		if updateErr == nil {
			updatedBalance, updateErr = balances.GetMerchantCollectionBalance(req.ImpalaMerchantID)
		}
	default:
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "INVALID_ACCOUNT_TYPE",
			"message": "accountType must be either 'collection' or 'payout'",
		})
		return
	}

	if updateErr != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "BALANCE_UPDATE_FAILED",
			"message": updateErr.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":           "success",
		"accountType":      req.AccountType,
		"impalaMerchantId": req.ImpalaMerchantID,
		"fromCurrency":     req.FromCurrency,
		"toCurrency":       req.ToCurrency,
		"originalAmount":   req.Amount,
		"convertedAmount":  convertedAmount,
		"fromRate":         fromRate.ConversionRate,
		"toRate":           toRate.ConversionRate,
		"baseCurrency":     fromRate.BaseCurrency,
		"exchangeRate":     rateMultiplier,
		"balances":         updatedBalance,
	})
}

