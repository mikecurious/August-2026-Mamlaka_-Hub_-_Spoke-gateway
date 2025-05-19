package balances

import (
	"fmt"
	"time"

	"com.mam-laka/database"
)

// MerchantCollectionBalance represents the merchant balance data.
type MerchantCollectionBalance struct {
	ID               uint      `gorm:"primaryKey" json:"id"`
	ImpalaMerchantID string    `gorm:"column:impalaMerchantId;type:varchar(255);unique" json:"impalaMerchantId"`
	USDBalance       float64   `gorm:"column:usdBalance;type:float(100,2)" json:"usdBalance"`
	USDCBalance      float64   `gorm:"column:usdcBalance;type:float(100,7)" json:"usdcBalance"`
	ImpaBalance      float64   `gorm:"column:impaBalance;type:float(100,7)" json:"impaBalance"`
	LumenBalance     float64   `gorm:"column:lumenBalance;type:float(100,7)" json:"lumenBalance"`
	LastUpdated      time.Time `gorm:"column:lastUpdated" json:"lastUpdated"`
	USDTBalance      float64   `gorm:"column:usdtBalance;type:float(100,2)" json:"usdtBalance"`
	KESBalance       float64   `gorm:"column:kesBalance;type:float(100,2)" json:"kesBalance"`
	EURBalance       float64   `gorm:"column:eurBalance;type:float(100,2)" json:"eurBalance"`
	GBPBalance       float64   `gorm:"column:gbpBalance;type:float(100,2)" json:"gbpBalance"`
	TZSBalance       float64   `gorm:"column:tzsBalance;type:float(100,2)" json:"tzsBalance"`
	UGXBalance       float64   `gorm:"column:ugxBalance;type:float(100,2)" json:"ugxBalance"`
	BaseCurrency     string    `gorm:"column:baseCurrency;type:varchar(3);default:USD" json:"baseCurrency"`
}



// TableName overrides the default table name.
func (MerchantCollectionBalance) TableName() string {
	return "merchant_collection_balance"
}

func GetTotalCollectionBalance(merchantId string, baseCurrency string) (map[string]interface{}, error) {
	db := database.GetConnection()

	// Retrieve merchant balances
	fmt.Println("Merchant ID:", merchantId)
	var balance MerchantCollectionBalance
	if err := db.Where("impalaMerchantId = ?", merchantId).First(&balance).Error; err != nil {
		return nil, err
	}

	// Fetch conversion rates from DB
	var forexRates []ForexRate
	if err := db.Find(&forexRates).Error; err != nil {
		return nil, err
	}

	// Map conversion rates (assumed: currencyCode -> units per USD)
	forexMap := make(map[string]float64)
	for _, rate := range forexRates {
		forexMap[rate.CurrencyCode] = rate.ConversionRate
	}

	// Ensure base currency conversion rate exists
	baseRate, exists := forexMap[baseCurrency]
	if !exists {
		return nil, fmt.Errorf("missing conversion rate for base currency: %s", baseCurrency)
	}

	// Map of all currency balances
	balances := map[string]float64{
		"USD":  balance.USDBalance,
		"USDC": balance.USDCBalance,
		"IMPA": balance.ImpaBalance,
		"XLM":  balance.LumenBalance,
		"USDT": balance.USDTBalance,
		"KES":  balance.KESBalance,
		"EUR":  balance.EURBalance,
		"GBP":  balance.GBPBalance,
		"TZS":  balance.TZSBalance,
		"UGX":  balance.UGXBalance,
	}

	// Calculate total balance converted to base currency
	totalBalance := 0.0
	for currency, amount := range balances {
		rate, ok := forexMap[currency]
		if !ok {
			return nil, fmt.Errorf("missing conversion rate for currency: %s", currency)
		}
		converted := (amount / rate) * baseRate
		totalBalance += converted
	}

	// Response structure
	response := map[string]interface{}{
		"usdBalance":   balance.USDBalance,
		"usdcBalance":  balance.USDCBalance,
		"impaBalance":  balance.ImpaBalance,
		"lumenBalance": balance.LumenBalance,
		"usdtBalance":  balance.USDTBalance,
		"kesBalance":   balance.KESBalance,
		"eurBalance":   balance.EURBalance,
		"gbpBalance":   balance.GBPBalance,
		"tzsBalance":   balance.TZSBalance,
		"ugxBalance":   balance.UGXBalance,
		"totalBalance": totalBalance,
		"baseCurrency": baseCurrency,
		"merchantId":   balance.ImpalaMerchantID,
	}

	return response, nil
}
