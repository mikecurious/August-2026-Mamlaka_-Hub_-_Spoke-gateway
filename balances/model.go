package balances

import (
	"fmt"
	"time"

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
	XAFBalance       float64 `gorm:"column:xafBalance;type:float(100,2)" json:"xafBalance"`
	NGNBalance       float64 `gorm:"column:ngnBalance;type:float(100,2)" json:"ngnBalance"`
	ZMWBalance       float64 `gorm:"column:zmwBalance;type:float(100,2)" json:"zmwBalance"`
	BaseCurrency     string  `gorm:"column:baseCurrency;type:varchar(3);default:USD" json:"baseCurrency"`
}

// TableName overrides the default table name.
func (MerchantBalance) TableName() string {
	return "merchant_balances"
}

// func AutoMigrate() {
// 	db := database.GetConnection()
// 	db.AutoMigrate(&ListingModel{})
// }

type ForexRate struct {
	CurrencyCode   string  `gorm:"column:currencyCode" json:"currencyCode"`
	ConversionRate float64 `gorm:"column:conversionRate" json:"conversionRate"`
}

func (ForexRate) TableName() string {
	return "forex_rates"
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

func GetTotalBalance(merchantId string, baseCurrency string) (map[string]interface{}, error) {
	db := database.GetConnection()

	// Retrieve merchant balances
	fmt.Println("Merchant ID:", merchantId)
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
		"XAF":  balance.XAFBalance,
		"NGN":  balance.NGNBalance,
		"ZMW":  balance.ZMWBalance,
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
		"totalBalance": totalBalance,
		"xafBalance":   balance.XAFBalance,
		"baseCurrency": baseCurrency,
		"merchantId":   balance.ImpalaMerchantID,
	}

	return response, nil
}

// SaveBalance creates or updates the merchant's balance record.
func SaveBalance(balance *MerchantBalance) error {
	db := database.GetConnection()
	// Update the last updated timestamp
	balance.LastUpdated = time.Now().Unix()

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
	// Add timestamp to the update data
	updatedData["lastUpdated"] = time.Now().Unix()

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
		return fmt.Errorf("insufficient KES balance: available %.2f, required %.2f", balance.KESBalance, amount)
	}

	// Deduct the amount
	balance.KESBalance -= amount
	balance.LastUpdated = time.Now().Unix()

	// Update the balance in the database
	err = db.Save(&balance).Error
	if err != nil {
		return fmt.Errorf("could not update merchant balance: %w", err)
	}

	return nil
}

// AddKESBalance adds the specified amount to the merchant's KES balance (refund function).
func AddKESBalance(impalaMerchantID string, amount float64) error {
	db := database.GetConnection()

	// Retrieve the merchant balance
	var balance MerchantCollectionBalance
	err := db.Where("impalaMerchantId = ?", impalaMerchantID).First(&balance).Error
	if err != nil {
		return fmt.Errorf("could not find merchant balance: %w", err)
	}

	// Add the amount
	balance.KESBalance += amount
	// balance.LastUpdated = time.Now().Unix()

	// Update the balance in the database
	err = db.Save(&balance).Error
	if err != nil {
		return fmt.Errorf("could not update merchant balance: %w", err)
	}

	return nil
}

// DeductUGXBalance deducts the specified amount from the merchant's UGX balance.
func DeductUGXBalance(impalaMerchantID string, amount float64) error {
	db := database.GetConnection()

	// Retrieve the merchant balance
	var balance MerchantBalance
	err := db.Where("impalaMerchantId = ?", impalaMerchantID).First(&balance).Error
	if err != nil {
		return fmt.Errorf("could not find merchant balance: %w", err)
	}

	// Check if the merchant has sufficient balance
	if balance.UGXBalance < amount {
		return fmt.Errorf("insufficient UGX balance: available %.2f, required %.2f", balance.UGXBalance, amount)
	}

	// Deduct the amount
	balance.UGXBalance -= amount
	balance.LastUpdated = time.Now().Unix()

	// Update the balance in the database
	err = db.Save(&balance).Error
	if err != nil {
		return fmt.Errorf("could not update merchant balance: %w", err)
	}

	return nil
}

// AddUGXBalance adds the specified amount to the merchant's UGX balance (refund function).
func AddUGXBalance(impalaMerchantID string, amount float64) error {
	db := database.GetConnection()

	// Retrieve the merchant balance
	var balance MerchantCollectionBalance
	err := db.Where("impalaMerchantId = ?", impalaMerchantID).First(&balance).Error
	if err != nil {
		return fmt.Errorf("could not find merchant balance: %w", err)
	}

	// Add the amount
	balance.UGXBalance += amount
	// balance.LastUpdated = time.Now().Unix()

	// Update the balance in the database
	err = db.Save(&balance).Error
	if err != nil {
		return fmt.Errorf("could not update merchant balance: %w", err)
	}

	return nil
}

// Generic function to add balance for any currency (refund function)
func AddBalance(impalaMerchantID string, currency string, amount float64) error {
	db := database.GetConnection()

	// Retrieve the merchant balance
	var balance MerchantBalance
	err := db.Where("impalaMerchantId = ?", impalaMerchantID).First(&balance).Error
	if err != nil {
		return fmt.Errorf("could not find merchant balance: %w", err)
	}

	// Add the amount to the appropriate currency balance
	switch currency {
	case "USD":
		balance.USDBalance += amount
	case "USDC":
		balance.USDCBalance += amount
	case "IMPA":
		balance.ImpaBalance += amount
	case "XLM", "LUMEN":
		balance.LumenBalance += amount
	case "USDT":
		balance.USDTBalance += amount
	case "KES":
		balance.KESBalance += amount
	case "EUR":
		balance.EURBalance += amount
	case "GBP":
		balance.GBPBalance += amount
	case "TZS":
		balance.TZSBalance += amount
	case "UGX":
		balance.UGXBalance += amount
	case "NGN":
		balance.NGNBalance += amount
	case "ZMW":
		balance.ZMWBalance += amount
	default:
		return fmt.Errorf("unsupported currency: %s", currency)
	}

	balance.LastUpdated = time.Now().Unix()

	// Update the balance in the database
	err = db.Save(&balance).Error
	if err != nil {
		return fmt.Errorf("could not update merchant balance: %w", err)
	}

	return nil
}

// Generic function to deduct balance for any currency
func DeductBalance(impalaMerchantID string, currency string, amount float64) error {
	db := database.GetConnection()

	// Retrieve the merchant balance
	var balance MerchantBalance
	err := db.Where("impalaMerchantId = ?", impalaMerchantID).First(&balance).Error
	if err != nil {
		return fmt.Errorf("could not find merchant balance: %w", err)
	}

	// Check sufficient balance and deduct from the appropriate currency
	switch currency {
	case "USD":
		if balance.USDBalance < amount {
			return fmt.Errorf("insufficient USD balance: available %.2f, required %.2f", balance.USDBalance, amount)
		}
		balance.USDBalance -= amount
	case "USDC":
		if balance.USDCBalance < amount {
			return fmt.Errorf("insufficient USDC balance: available %.7f, required %.7f", balance.USDCBalance, amount)
		}
		balance.USDCBalance -= amount
	case "IMPA":
		if balance.ImpaBalance < amount {
			return fmt.Errorf("insufficient IMPA balance: available %.7f, required %.7f", balance.ImpaBalance, amount)
		}
		balance.ImpaBalance -= amount
	case "XLM", "LUMEN":
		if balance.LumenBalance < amount {
			return fmt.Errorf("insufficient XLM balance: available %.7f, required %.7f", balance.LumenBalance, amount)
		}
		balance.LumenBalance -= amount
	case "USDT":
		if balance.USDTBalance < amount {
			return fmt.Errorf("insufficient USDT balance: available %.2f, required %.2f", balance.USDTBalance, amount)
		}
		balance.USDTBalance -= amount
	case "KES":
		if balance.KESBalance < amount {
			return fmt.Errorf("insufficient KES balance: available %.2f, required %.2f", balance.KESBalance, amount)
		}
		balance.KESBalance -= amount
	case "EUR":
		if balance.EURBalance < amount {
			return fmt.Errorf("insufficient EUR balance: available %.2f, required %.2f", balance.EURBalance, amount)
		}
		balance.EURBalance -= amount
	case "GBP":
		if balance.GBPBalance < amount {
			return fmt.Errorf("insufficient GBP balance: available %.2f, required %.2f", balance.GBPBalance, amount)
		}
		balance.GBPBalance -= amount
	case "TZS":
		if balance.TZSBalance < amount {
			return fmt.Errorf("insufficient TZS balance: available %.2f, required %.2f", balance.TZSBalance, amount)
		}
		balance.TZSBalance -= amount
	case "UGX":
		if balance.UGXBalance < amount {
			return fmt.Errorf("insufficient UGX balance: available %.2f, required %.2f", balance.UGXBalance, amount)
		}
		balance.UGXBalance -= amount
	case "NGN":
		if balance.NGNBalance < amount {
			return fmt.Errorf("insufficient NGN balance: available %.2f, required %.2f", balance.NGNBalance, amount)
		}
		balance.NGNBalance -= amount
	case "ZMW":
		if balance.ZMWBalance < amount {
			return fmt.Errorf("insufficient ZMW balance: available %.2f, required %.2f", balance.ZMWBalance, amount)
		}
		balance.ZMWBalance -= amount
	default:
		return fmt.Errorf("unsupported currency: %s", currency)
	}

	balance.LastUpdated = time.Now().Unix()

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
	case "ZMW":
		originBalance = &balance.ZMWBalance
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
	case "ZMW":
		destinationBalance = &balance.ZMWBalance
	default:
		return fmt.Errorf("invalid destination currency: %s", destinationCurrency)
	}

	// Check if there is enough balance to convert
	if *originBalance < amount {
		return fmt.Errorf("insufficient balance in %s: available %.2f, required %.2f", originCurrency, *originBalance, amount)
	}

	// Perform conversion
	convertedAmount := amount * exchangeRate
	*originBalance -= amount
	*destinationBalance += convertedAmount

	// Update timestamp
	balance.LastUpdated = time.Now().Unix()

	// Save the updated balance
	err = db.Save(&balance).Error
	if err != nil {
		return fmt.Errorf("could not update merchant balance: %w", err)
	}

	return nil
}

// west africa functions
func DeductXOFBalance(impalaMerchantID string, amount float64) error {
	db := database.GetConnection()

	// Retrieve the merchant balance
	var balance MerchantBalance
	err := db.Where("impalaMerchantId = ?", impalaMerchantID).First(&balance).Error
	if err != nil {
		return fmt.Errorf("could not find merchant balance: %w", err)
	}
	// print the current balance
	fmt.Printf("Current XOF Balance: %.2f\n", balance.ImpaBalance)
	// Check if the merchant has sufficient balance
	if balance.ImpaBalance < amount {
		return fmt.Errorf("insufficient UGX balance: available %.2f, required %.2f", balance.ImpaBalance, amount)
	}

	// Deduct the amount
	balance.ImpaBalance -= amount
	// balance.LastUpdated = time.Now().Unix()

	// Update the balance in the database
	err = db.Save(&balance).Error
	if err != nil {
		return fmt.Errorf("could not update merchant balance: %w", err)
	}
	//pritn final balance
	fmt.Printf("Final XOF Balance: %.2f\n", balance.ImpaBalance)

	return nil
}

// AddUGXBalance adds the specified amount to the merchant's UGX balance (refund function).
func AddXOFBalance(impalaMerchantID string, amount float64) error {
	db := database.GetConnection()

	// Retrieve the merchant balance
	var balance MerchantCollectionBalance
	err := db.Where("impalaMerchantId = ?", impalaMerchantID).First(&balance).Error
	if err != nil {
		return fmt.Errorf("could not find merchant balance: %w", err)
	}

	// Add the amount
	balance.ImpaBalance += amount
	// balance.LastUpdated = time.Now().Unix()

	// Update the balance in the database
	err = db.Save(&balance).Error
	if err != nil {
		return fmt.Errorf("could not update merchant balance: %w", err)
	}

	return nil
}

// add XAF BALANCE
func AddXAFBalance(impalaMerchantID string, amount float64) error {
	db := database.GetConnection()

	// Retrieve the merchant balance
	var balance MerchantCollectionBalance
	err := db.Where("impalaMerchantId = ?", impalaMerchantID).First(&balance).Error
	if err != nil {
		return fmt.Errorf("could not find merchant balance: %w", err)
	}

	// Add the amount
	balance.XAFBalance += amount
	// balance.LastUpdated = time.Now().Unix()

	// Update the balance in the database
	err = db.Save(&balance).Error
	if err != nil {
		return fmt.Errorf("could not update merchant balance: %w", err)
	}

	return nil
}

// DeductNGNBalance deducts NGN from the merchant payout wallet (merchant_balances).
func DeductNGNBalance(impalaMerchantID string, amount float64) error {
	db := database.GetConnection()

	var balance MerchantBalance
	err := db.Where("impalaMerchantId = ?", impalaMerchantID).First(&balance).Error
	if err != nil {
		return fmt.Errorf("could not find merchant balance: %w", err)
	}

	if balance.NGNBalance < amount {
		return fmt.Errorf("insufficient NGN balance: available %.2f, required %.2f", balance.NGNBalance, amount)
	}

	balance.NGNBalance -= amount
	balance.LastUpdated = time.Now().Unix()

	if err := db.Save(&balance).Error; err != nil {
		return fmt.Errorf("could not update merchant balance: %w", err)
	}

	return nil
}

// AddNGNBalance adds NGN to the merchant collection wallet (merchant_collection_balance).
func AddNGNBalance(impalaMerchantID string, amount float64) error {
	db := database.GetConnection()

	var balance MerchantCollectionBalance
	err := db.Where("impalaMerchantId = ?", impalaMerchantID).First(&balance).Error
	if err != nil {
		return fmt.Errorf("could not find merchant collection balance: %w", err)
	}

	balance.NGNBalance += amount

	if err := db.Save(&balance).Error; err != nil {
		return fmt.Errorf("could not update merchant balance: %w", err)
	}

	return nil
}

// AddNGNPayoutBalance adds NGN to the merchant payout wallet (merchant_balances) - used for payout refunds.
func AddNGNPayoutBalance(impalaMerchantID string, amount float64) error {
	db := database.GetConnection()

	var balance MerchantBalance
	err := db.Where("impalaMerchantId = ?", impalaMerchantID).First(&balance).Error
	if err != nil {
		return fmt.Errorf("could not find merchant balance: %w", err)
	}

	balance.NGNBalance += amount
	balance.LastUpdated = time.Now().Unix()

	if err := db.Save(&balance).Error; err != nil {
		return fmt.Errorf("could not update merchant balance: %w", err)
	}

	return nil
}

// DEDUCT XAF BALACNE
func DeductXAFBalance(impalaMerchantID string, amount float64) error {
	db := database.GetConnection()

	// Retrieve the merchant balance
	var balance MerchantCollectionBalance
	err := db.Where("impalaMerchantId = ?", impalaMerchantID).First(&balance).Error
	if err != nil {
		return fmt.Errorf("could not find merchant balance: %w", err)
	}

	// Safety check: Prevent negative balance
	if balance.XAFBalance < amount {
		return fmt.Errorf("insufficient balance: current %.2f, required %.2f", balance.XAFBalance, amount)
	}

	// Deduct the amount
	balance.XAFBalance -= amount
	// Optionally update timestamp
	// balance.LastUpdated = time.Now()

	// Save updated balance
	err = db.Save(&balance).Error
	if err != nil {
		return fmt.Errorf("could not update merchant balance: %w", err)
	}

	return nil
}
