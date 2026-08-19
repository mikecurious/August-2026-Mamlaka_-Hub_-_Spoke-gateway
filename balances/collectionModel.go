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
	XAFBalance       float64   `gorm:"column:xafBalance;type:float(100,2)" json:"xafBalance"`
	NGNBalance       float64   `gorm:"column:ngnBalance;type:float(100,2)" json:"ngnBalance"`
	ZMWBalance       float64   `gorm:"column:zmwBalance;type:float(100,2)" json:"zmwBalance"`
	GMDBalance       float64   `gorm:"column:gmdBalance;type:float(100,2)" json:"gmdBalance"`
	RWFBalance       float64   `gorm:"column:rwfBalance;type:float(100,2)" json:"rwfBalance"`
	AirtelBalance    float64   `gorm:"column:airtelBalance;type:float(100,2)" json:"airtelBalance"`
	BaseCurrency     string    `gorm:"column:baseCurrency;type:varchar(3);default:USD" json:"baseCurrency"`
}

// TableName overrides the default table name.
func (MerchantCollectionBalance) TableName() string {
	return "merchant_collection_balance"
}
// GetMerchantCollectionBalance retrieves a merchant's collection balance by their ImpalaMerchantID.
func GetMerchantCollectionBalance(impalaMerchantID string) (MerchantCollectionBalance, error) {
    db := database.GetConnection()
    var balance MerchantCollectionBalance
    if err := db.Where("impalaMerchantId = ?", impalaMerchantID).First(&balance).Error; err != nil {
        return MerchantCollectionBalance{}, fmt.Errorf("could not find merchant collection balance: %w", err)
    }
    return balance, nil
}

// ConvertCollectionBalance converts a given amount from one currency to another within the collection balance table.
func ConvertCollectionBalance(impalaMerchantID, originCurrency, destinationCurrency string, amount, exchangeRate float64) error {
    db := database.GetConnection()

    var balance MerchantCollectionBalance
    if err := db.Where("impalaMerchantId = ?", impalaMerchantID).First(&balance).Error; err != nil {
        return fmt.Errorf("could not find merchant collection balance: %w", err)
    }

    var originBalance *float64
    var destinationBalance *float64

    switch originCurrency {
    case "USD":
        originBalance = &balance.USDBalance
    case "USDC":
        originBalance = &balance.USDCBalance
    case "IMPA":
        originBalance = &balance.ImpaBalance
    case "LUMEN", "XLM":
        originBalance = &balance.LumenBalance
    case "USDT":
        originBalance = &balance.USDTBalance
    case "KES":
        originBalance = &balance.KESBalance
    case "EUR":
        originBalance = &balance.EURBalance
    case "GBP":
        originBalance = &balance.GBPBalance
    case "TZS":
        originBalance = &balance.TZSBalance
    case "UGX":
        originBalance = &balance.UGXBalance
    case "XAF":
        originBalance = &balance.XAFBalance
    case "ZMW":
        originBalance = &balance.ZMWBalance
    case "GMD":
        originBalance = &balance.GMDBalance
    case "RWF":
        originBalance = &balance.RWFBalance
    default:
        return fmt.Errorf("invalid origin currency: %s", originCurrency)
    }

    switch destinationCurrency {
    case "USD":
        destinationBalance = &balance.USDBalance
    case "USDC":
        destinationBalance = &balance.USDCBalance
    case "IMPA":
        destinationBalance = &balance.ImpaBalance
    case "LUMEN", "XLM":
        destinationBalance = &balance.LumenBalance
    case "USDT":
        destinationBalance = &balance.USDTBalance
    case "KES":
        destinationBalance = &balance.KESBalance
    case "EUR":
        destinationBalance = &balance.EURBalance
    case "GBP":
        destinationBalance = &balance.GBPBalance
    case "TZS":
        destinationBalance = &balance.TZSBalance
    case "UGX":
        destinationBalance = &balance.UGXBalance
    case "XAF":
        destinationBalance = &balance.XAFBalance
    case "ZMW":
        destinationBalance = &balance.ZMWBalance
    case "GMD":
        destinationBalance = &balance.GMDBalance
    case "RWF":
        destinationBalance = &balance.RWFBalance
    default:
        return fmt.Errorf("invalid destination currency: %s", destinationCurrency)
    }

    if originBalance == nil || destinationBalance == nil {
        return fmt.Errorf("invalid currency conversion from %s to %s", originCurrency, destinationCurrency)
    }

    if *originBalance < amount {
        return fmt.Errorf("insufficient balance in %s: available %.2f, required %.2f", originCurrency, *originBalance, amount)
    }

    convertedAmount := amount * exchangeRate
    *originBalance -= amount
    *destinationBalance += convertedAmount

    balance.LastUpdated = time.Now()

    if err := db.Save(&balance).Error; err != nil {
        return fmt.Errorf("could not update merchant collection balance: %w", err)
    }

    return nil
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
		"XAF":  balance.XAFBalance,
		"NGN":  balance.NGNBalance,
		"ZMW":  balance.ZMWBalance,
		"GMD":  balance.GMDBalance,
		"RWF":  balance.RWFBalance,
	}

	// Calculate total balance converted to base currency
	totalBalance := 0.0
	for currency, amount := range balances {
		rate, ok := forexMap[currency]
		if !ok || rate == 0 {
			// Skip currencies without a configured conversion rate.
			continue
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
		"ngnBalance":   balance.NGNBalance,
		"zmwBalance":   balance.ZMWBalance,
		"gmdBalance":   balance.GMDBalance,
		"rwfBalance":   balance.RWFBalance,
		"totalBalance": totalBalance,
		"baseCurrency": baseCurrency,
		"xafBalance":   balance.XAFBalance,
		"merchantId":   balance.ImpalaMerchantID,
	}

	return response, nil
}
