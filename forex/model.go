package forex

import (
	"time"

	"com.mam-laka/database"
	"gorm.io/gorm"
)

// ForexRate represents the forex rate data.
type ForexRate struct {
	ID             uint      `gorm:"primaryKey" json:"id"`
	CurrencyCode   string    `gorm:"column:currencyCode;type:varchar(3);unique;not null" json:"currencyCode"`
	ConversionRate float64   `gorm:"column:conversionRate;type:decimal(20,8);not null" json:"conversionRate"`
	BaseCurrency   string    `gorm:"column:baseCurrency;type:varchar(3);default:USD" json:"baseCurrency"`
	LastUpdated    time.Time `gorm:"column:lastUpdated;type:datetime;default:CURRENT_TIMESTAMP" json:"lastUpdated"`
	CreatedAt      time.Time `gorm:"column:createdAt;type:datetime;default:CURRENT_TIMESTAMP" json:"createdAt"`
}

// TableName overrides the default table name.
func (ForexRate) TableName() string {
	return "forex_rates"
}

// AutoMigrate runs database migration for forex rates.
func AutoMigrate() {
	db := database.GetConnection()
	db.AutoMigrate(&ForexRate{})
}

// GetAllForexRates retrieves all forex rates from the database.
func GetAllForexRates() ([]ForexRate, error) {
	db := database.GetConnection()
	var rates []ForexRate
	err := db.Order("currencyCode ASC").Find(&rates).Error
	if err != nil {
		return nil, err
	}
	return rates, nil
}

// GetForexRateByCurrency retrieves a forex rate by currency code.
func GetForexRateByCurrency(currencyCode string) (ForexRate, error) {
	db := database.GetConnection()
	var rate ForexRate
	err := db.Where("currencyCode = ?", currencyCode).First(&rate).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return ForexRate{}, err
		}
		return ForexRate{}, err
	}
	return rate, nil
}

// SaveForexRate creates or updates a forex rate.
func SaveForexRate(rate *ForexRate) error {
	db := database.GetConnection()
	rate.LastUpdated = time.Now()
	
	// Check if rate exists
	var existingRate ForexRate
	err := db.Where("currencyCode = ?", rate.CurrencyCode).First(&existingRate).Error
	
	if err == gorm.ErrRecordNotFound {
		// Create new rate
		rate.CreatedAt = time.Now()
		err = db.Create(rate).Error
	} else if err == nil {
		// Update existing rate
		rate.ID = existingRate.ID
		rate.CreatedAt = existingRate.CreatedAt
		err = db.Save(rate).Error
	}
	
	return err
}

// ConvertCurrency converts an amount from one currency to another using forex rates.
// The conversion rate is stored as "units per USD" (e.g., 150 KES = 1 USD means rate = 150)
// To convert: (amount / fromRate) * toRate
func ConvertCurrency(fromCurrency, toCurrency string, amount float64) (float64, error) {
	db := database.GetConnection()
	
	// If same currency, return amount as is
	if fromCurrency == toCurrency {
		return amount, nil
	}
	
	// Get rates for both currencies
	var fromRate, toRate ForexRate
	
	// Get from currency rate
	err := db.Where("currencyCode = ?", fromCurrency).First(&fromRate).Error
	if err != nil {
		return 0, err
	}
	
	// Get to currency rate
	err = db.Where("currencyCode = ?", toCurrency).First(&toRate).Error
	if err != nil {
		return 0, err
	}
	
	// Convert using the pattern from balances/model.go: (amount / rate) * baseRate
	// For direct conversion: (amount / fromRate) * toRate
	// This converts: fromCurrency -> USD -> toCurrency
	convertedAmount := (amount / fromRate.ConversionRate) * toRate.ConversionRate
	
	return convertedAmount, nil
}

