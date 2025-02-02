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
