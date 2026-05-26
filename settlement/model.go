package settlement

import (
	"fmt"
	"strings"
	"time"

	"com.mam-laka/database"
	"gorm.io/gorm"
)

type ManualSettlement struct {
	ID               uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	ImpalaMerchantID string    `gorm:"column:impalaMerchantId;type:varchar(255);index;not null" json:"impalaMerchantId"`
	Currency         string    `gorm:"column:currency;type:varchar(10);not null" json:"currency"`
	SettledAmount    float64   `gorm:"column:settledAmount;type:decimal(18,2);not null" json:"settledAmount"`
	FeePercent       float64   `gorm:"column:feePercent;type:decimal(10,4);not null" json:"feePercent"`
	FeeAmount        float64   `gorm:"column:feeAmount;type:decimal(18,2);not null" json:"feeAmount"`
	TotalDeducted    float64   `gorm:"column:totalDeducted;type:decimal(18,2);not null" json:"totalDeducted"`
	BalanceBefore    float64   `gorm:"column:balanceBefore;type:decimal(18,2);not null" json:"balanceBefore"`
	BalanceAfter     float64   `gorm:"column:balanceAfter;type:decimal(18,2);not null" json:"balanceAfter"`
	Status           string    `gorm:"column:status;type:varchar(20);default:COMPLETED;not null" json:"status"`
	SettledBy        string    `gorm:"column:settledBy;type:varchar(255);not null" json:"settledBy"`
	Note             string    `gorm:"column:note;type:text" json:"note"`
	CreatedAt        time.Time `gorm:"column:createdAt;autoCreateTime" json:"createdAt"`
}

func (ManualSettlement) TableName() string {
	return "manual_settlements"
}

func AutoMigrate() {
	db := database.GetConnection()
	db.AutoMigrate(&ManualSettlement{})
}

func currencyColumn(currency string) (string, bool) {
	switch strings.ToUpper(strings.TrimSpace(currency)) {
	case "USD":
		return "usdBalance", true
	case "USDC":
		return "usdcBalance", true
	case "IMPA":
		return "impaBalance", true
	case "XLM", "LUMEN":
		return "lumenBalance", true
	case "USDT":
		return "usdtBalance", true
	case "KES":
		return "kesBalance", true
	case "EUR":
		return "eurBalance", true
	case "GBP":
		return "gbpBalance", true
	case "TZS":
		return "tzsBalance", true
	case "UGX":
		return "ugxBalance", true
	case "XAF":
		return "xafBalance", true
	case "NGN":
		return "ngnBalance", true
	case "ZMW":
		return "zmwBalance", true
	case "GMD":
		return "gmdBalance", true
	default:
		return "", false
	}
}

func getCollectionBalance(tx *gorm.DB, merchantID, column string) (float64, error) {
	var balance float64
	query := fmt.Sprintf("SELECT COALESCE(%s, 0) FROM merchant_collection_balance WHERE impalaMerchantId = ?", column)
	if err := tx.Raw(query, merchantID).Scan(&balance).Error; err != nil {
		return 0, err
	}
	return balance, nil
}
