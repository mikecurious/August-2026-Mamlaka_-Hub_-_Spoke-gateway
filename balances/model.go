package balances

import (
	"fmt"

	"com.mam-laka/database"
)

// MerchantBalance represents the merchant balance data.
type MerchantBalance struct {
	ID               uint    `gorm:"primaryKey" json:"id"`
	ImpalaMerchantID string  `gorm:"column:impalaMerchantId;type:varchar(255);unique" json:"impalaMerchantId"`
	USDBalance       float64 `gorm:"column:usdBalance;type:float(100,2)" json:"usdBalance"`
	USDCBalance      float64 `gorm:"column:usdcBalance;type:float(100,7)" json:"usdcBalance"`
	ImpaBalance      float64 `gorm:"column:impaBalance;type:float(100,7)" json:"impaBalance"`
	LumenBalance     float64 `gorm:"column:lumenBalance;type:float(100,7)" json:"lumenBalance"`
	LastUpdated      int64   `gorm:"column:lastUpdated;type:int" json:"lastUpdated"`
	USDTBalance      float64 `gorm:"column:usdtBalance;type:float(100,2)" json:"usdtBalance"`
	KESBalance       float64 `gorm:"column:kesBalance;type:float(100,2)" json:"kesBalance"`
	EURBalance       float64 `gorm:"column:eurBalance;type:float(100,2)" json:"eurBalance"`
	GBPBalance       float64 `gorm:"column:gbpBalance;type:float(100,2)" json:"gbpBalance"`
	TZSBalance       float64 `gorm:"column:tzsBalance;type:float(100,2)" json:"tzsBalance"`
	UGXBalance       float64 `gorm:"column:ugxBalance;type:float(100,2)" json:"ugxBalance"`
	BaseCurrency     string  `gorm:"column:baseCurrency;type:varchar(3);default:USD" json:"baseCurrency"`
}

// TableName overrides the default table name.
func (MerchantBalance) TableName() string {
	return "merchant_balances"
}

// GetMerchantBalance retrieves a merchant's balance by their ImpalaMerchantID.
func GetMerchantBalance(impalaMerchantID string) (MerchantBalance, error) {
	db := database.GetConnection()
	var balance MerchantBalance
	err := db.Where("impalaMerchantId = ?", impalaMerchantID).First(&balance).Error
	if err != nil {
		return MerchantBalance{}, fmt.Errorf("could not find merchant balance: %w", err)
	}
	return balance, nil
}

func (ForexRate) TableName() string {
	return "forex_rates"
}

type ForexRate struct {
	CurrencyCode   string  `gorm:"column:currencyCode" json:"currencyCode"`
	ConversionRate float64 `gorm:"column:conversionRate" json:"conversionRate"`
}

func GetTotalBalance(merchantId string, baseCurrency string) (map[string]interface{}, error) {
	db := database.GetConnection()

	// Retrieve merchant balances
	fmt.Println("Merchant ID:", merchantId)
	// var balance MerchantCollectionBalance
	var balance MerchantBalance
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

// SaveBalance creates or updates the merchant's balance record.
func SaveBalance(balance *MerchantBalance) error {
	db := database.GetConnection()
	// Checking if the balance already exists and updating or inserting accordingly
	err := db.Save(balance).Error
	if err != nil {
		return fmt.Errorf("could not save merchant balance: %w", err)
	}
	return nil
}

// GetAllMerchantBalances retrieves all merchant balances.
func GetAllMerchantBalances() ([]MerchantBalance, error) {
	db := database.GetConnection()
	var balances []MerchantBalance
	err := db.Find(&balances).Error
	if err != nil {
		return nil, fmt.Errorf("could not fetch merchant balances: %w", err)
	}
	return balances, nil
}

// UpdateBalance updates a merchant's balance by their ID.
func UpdateBalance(id uint, updatedData map[string]interface{}) error {
	db := database.GetConnection()
	err := db.Model(&MerchantBalance{}).Where("id = ?", id).Updates(updatedData).Error
	if err != nil {
		return fmt.Errorf("could not update merchant balance: %w", err)
	}
	return nil
}

// DeleteMerchantBalance deletes a merchant balance by ImpalaMerchantID.
func DeleteMerchantBalance(impalaMerchantID string) error {
	db := database.GetConnection()
	err := db.Where("impalaMerchantId = ?", impalaMerchantID).Delete(&MerchantBalance{}).Error
	if err != nil {
		return fmt.Errorf("could not delete merchant balance: %w", err)
	}
	return nil
}

// DeductKESBalance deducts the specified amount from the merchant's KES balance.
func DeductKESBalance(impalaMerchantID string, amount float64) error {
	db := database.GetConnection()

	// Retrieve the merchant balance
	var balance MerchantBalance
	err := db.Where("impalaMerchantId = ?", impalaMerchantID).First(&balance).Error
	if err != nil {
		return fmt.Errorf("could not find merchant balance: %w", err)
	}

	// Check if the merchant has sufficient balance
	if balance.KESBalance < amount {
		return fmt.Errorf("insufficient KES balance")
	}

	// Deduct the amount
	balance.KESBalance -= amount

	// Update the balance in the database
	err = db.Save(&balance).Error
	if err != nil {
		return fmt.Errorf("could not update merchant balance: %w", err)
	}

	return nil
}

// ConvertBalance converts a given amount from one currency to another and updates the balances.
func ConvertBalance(impalaMerchantID, originCurrency, destinationCurrency string, amount, exchangeRate float64) error {
	db := database.GetConnection()

	// Retrieve the merchant balance
	var balance MerchantBalance
	err := db.Where("impalaMerchantId = ?", impalaMerchantID).First(&balance).Error
	if err != nil {
		return fmt.Errorf("could not find merchant balance: %w", err)
	}

	// Get the origin currency balance
	var originBalance *float64
	var destinationBalance *float64

	switch originCurrency {
	case "USD":
		originBalance = &balance.USDBalance
	case "USDC":
		originBalance = &balance.USDCBalance
	case "IMPA":
		originBalance = &balance.ImpaBalance
	case "LUMEN":
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
	default:
		return fmt.Errorf("invalid origin currency: %s", originCurrency)
	}

	// Get the destination currency balance
	switch destinationCurrency {
	case "USD":
		destinationBalance = &balance.USDBalance
	case "USDC":
		destinationBalance = &balance.USDCBalance
	case "IMPA":
		destinationBalance = &balance.ImpaBalance
	case "LUMEN":
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
	default:
		return fmt.Errorf("invalid destination currency: %s", destinationCurrency)
	}

	// Check if there is enough balance to convert
	if *originBalance < amount {
		return fmt.Errorf("insufficient balance in %s", originCurrency)
	}

	// Perform conversion
	convertedAmount := amount * exchangeRate
	*originBalance -= amount
	*destinationBalance += convertedAmount

	// Save the updated balance
	err = db.Save(&balance).Error
	if err != nil {
		return fmt.Errorf("could not update merchant balance: %w", err)
	}

	return nil
}
