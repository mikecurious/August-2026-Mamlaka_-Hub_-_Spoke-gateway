package settlement

import (
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"

	"com.mam-laka/auth"
	"com.mam-laka/database"
	"github.com/gin-gonic/gin"
)

type ManualSettlementRequest struct {
	ImpalaMerchantID string  `json:"impalaMerchantId" binding:"required"`
	Amount           float64 `json:"amount" binding:"required,gt=0"`
	Currency         string  `json:"currency" binding:"required"`
	FeePercent       float64 `json:"feePercent" binding:"gte=0"`
	Note             string  `json:"note"`
}

func RegisterRoutes(apiV1 *gin.RouterGroup) {
	merchantSettlementGroup := apiV1.Group("/settlement")
	merchantSettlementGroup.Use(auth.AuthMiddleware())
	{
		merchantSettlementGroup.GET("/report", ListMyManualSettlementsHandler)
	}

	settlementGroup := apiV1.Group("/settlement")
	settlementGroup.Use(auth.AuthMiddleware(), financeOnlyMiddleware())
	{
		settlementGroup.POST("/manual", ManualSettlementHandler)
		settlementGroup.GET("/manual/list", ListManualSettlementsHandler)
	}
}

func financeOnlyMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		usernameVal, ok := c.Get("username")
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized: missing username claim"})
			c.Abort()
			return
		}

		username, ok := usernameVal.(string)
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized: invalid username claim"})
			c.Abort()
			return
		}

		if !isFinanceUser(username) {
			c.JSON(http.StatusForbidden, gin.H{"error": "Forbidden: finance access only"})
			c.Abort()
			return
		}

		c.Next()
	}
}

func isFinanceUser(username string) bool {
	allowed := strings.TrimSpace(os.Getenv("FINANCE_USERS"))
	if allowed == "" {
		allowed = "finance"
	}

	username = strings.ToLower(strings.TrimSpace(username))
	for _, u := range strings.Split(allowed, ",") {
		if username == strings.ToLower(strings.TrimSpace(u)) {
			return true
		}
	}
	return false
}

func ManualSettlementHandler(c *gin.Context) {
	var req ManualSettlementRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload", "details": err.Error()})
		return
	}

	req.Currency = strings.ToUpper(strings.TrimSpace(req.Currency))
	column, ok := currencyColumn(req.Currency)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Unsupported currency: %s", req.Currency)})
		return
	}

	feeAmount := req.Amount * (req.FeePercent / 100)
	totalDeduction := req.Amount + feeAmount
	if totalDeduction <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid deduction amount"})
		return
	}

	usernameVal, _ := c.Get("username")
	settledBy, _ := usernameVal.(string)

	db := database.GetConnection()
	tx := db.Begin()
	if tx.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not start settlement transaction"})
		return
	}

	balanceBefore, err := getCollectionBalance(tx, req.ImpalaMerchantID, column)
	if err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not fetch merchant collection balance", "details": err.Error()})
		return
	}

	updateSQL := fmt.Sprintf("UPDATE merchant_collection_balance SET %s = %s - ? WHERE impalaMerchantId = ? AND %s >= ?", column, column, column)
	result := tx.Exec(updateSQL, totalDeduction, req.ImpalaMerchantID, totalDeduction)
	if result.Error != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to deduct collection balance", "details": result.Error.Error()})
		return
	}
	if result.RowsAffected == 0 {
		tx.Rollback()
		c.JSON(http.StatusBadRequest, gin.H{
			"error":             "Insufficient collection balance or merchant not found",
			"requiredDeduction": totalDeduction,
			"availableBalance":  balanceBefore,
		})
		return
	}

	balanceAfter, err := getCollectionBalance(tx, req.ImpalaMerchantID, column)
	if err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not read updated balance", "details": err.Error()})
		return
	}

	record := ManualSettlement{
		ImpalaMerchantID: req.ImpalaMerchantID,
		Currency:         req.Currency,
		SettledAmount:    req.Amount,
		FeePercent:       req.FeePercent,
		FeeAmount:        feeAmount,
		TotalDeducted:    totalDeduction,
		BalanceBefore:    balanceBefore,
		BalanceAfter:     balanceAfter,
		Status:           "COMPLETED",
		SettledBy:        settledBy,
		Note:             req.Note,
	}

	if err := tx.Create(&record).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save settlement audit", "details": err.Error()})
		return
	}

	if err := tx.Commit().Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to finalize settlement", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":       "Manual settlement completed successfully",
		"settlementId":  record.ID,
		"merchantId":    record.ImpalaMerchantID,
		"currency":      record.Currency,
		"settledAmount": record.SettledAmount,
		"feePercent":    record.FeePercent,
		"feeAmount":     record.FeeAmount,
		"totalDeducted": record.TotalDeducted,
		"balanceBefore": record.BalanceBefore,
		"balanceAfter":  record.BalanceAfter,
		"status":        record.Status,
		"settledBy":     record.SettledBy,
	})
}

func ListManualSettlementsHandler(c *gin.Context) {
	db := database.GetConnection()

	page := 1
	if rawPage := strings.TrimSpace(c.Query("page")); rawPage != "" {
		parsed, err := strconv.Atoi(rawPage)
		if err == nil && parsed > 0 {
			page = parsed
		}
	}

	limit := 20
	if rawLimit := strings.TrimSpace(c.Query("limit")); rawLimit != "" {
		parsed, err := strconv.Atoi(rawLimit)
		if err == nil && parsed > 0 && parsed <= 200 {
			limit = parsed
		}
	}

	query := db.Model(&ManualSettlement{})
	if merchantID := strings.TrimSpace(c.Query("merchantId")); merchantID != "" {
		query = query.Where("impalaMerchantId = ?", merchantID)
	}
	if currency := strings.ToUpper(strings.TrimSpace(c.Query("currency"))); currency != "" {
		query = query.Where("currency = ?", currency)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to count manual settlements", "details": err.Error()})
		return
	}

	offset := (page - 1) * limit
	var settlements []ManualSettlement
	if err := query.Order("id DESC").Offset(offset).Limit(limit).Find(&settlements).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch manual settlements", "details": err.Error()})
		return
	}

	totalPages := 0
	if total > 0 {
		totalPages = int((total + int64(limit) - 1) / int64(limit))
	}

	c.JSON(http.StatusOK, gin.H{
		"count": len(settlements),
		"pagination": gin.H{
			"page":       page,
			"limit":      limit,
			"totalPages": totalPages,
			"totalItems": total,
		},
		"settlements": settlements,
	})
}

func ListMyManualSettlementsHandler(c *gin.Context) {
	merchantIDVal, exists := c.Get("merchantID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Missing merchantID in token"})
		return
	}

	merchantID, ok := merchantIDVal.(string)
	if !ok || strings.TrimSpace(merchantID) == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid merchantID in token"})
		return
	}

	db := database.GetConnection()

	page := 1
	if rawPage := strings.TrimSpace(c.Query("page")); rawPage != "" {
		parsed, err := strconv.Atoi(rawPage)
		if err == nil && parsed > 0 {
			page = parsed
		}
	}

	limit := 20
	if rawLimit := strings.TrimSpace(c.Query("limit")); rawLimit != "" {
		parsed, err := strconv.Atoi(rawLimit)
		if err == nil && parsed > 0 && parsed <= 200 {
			limit = parsed
		}
	}

	query := db.Model(&ManualSettlement{}).
		Where("impalaMerchantId = ?", merchantID)
	if currency := strings.ToUpper(strings.TrimSpace(c.Query("currency"))); currency != "" {
		query = query.Where("currency = ?", currency)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to count settlement report", "details": err.Error()})
		return
	}

	offset := (page - 1) * limit
	var settlements []ManualSettlement
	if err := query.Order("id DESC").Offset(offset).Limit(limit).Find(&settlements).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch settlement report", "details": err.Error()})
		return
	}

	totalPages := 0
	if total > 0 {
		totalPages = int((total + int64(limit) - 1) / int64(limit))
	}

	c.JSON(http.StatusOK, gin.H{
		"merchantId": merchantID,
		"count":      len(settlements),
		"pagination": gin.H{
			"page":       page,
			"limit":      limit,
			"totalPages": totalPages,
			"totalItems": total,
		},
		"settlements": settlements,
	})
}
