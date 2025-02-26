package transactions

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"

	"com.mam-laka/database"
)

type TransactionModel struct {
	ID                  uint    `gorm:"primaryKey" json:"id"`
	ImpalaMerchantID    string  `gorm:"column:impalaMerchantId;type:varchar(255)" json:"impalaMerchantId"`
	TransactionStatus   string  `gorm:"column:transactionStatus" json:"transactionStatus"`
	TransactionReport   string  `gorm:"column:transactionReport" json:"transactionReport"`
	Currency            string  `gorm:"column:currency" json:"currency"`
	Amount              int     `gorm:"column:amount" json:"amount"`
	Msisdn              string  `gorm:"column:msisdn" json:"msisdn"`
	NetAmount           float64 `gorm:"column:netAmount" json:"netAmount"`
	SecureID            string `gorm:"column:secureId" json:"secureId"`
	SourceOfFunds       string  `gorm:"column:sourceOfFunds" json:"sourceOfFunds"`
	ExternalID          string `gorm:"column:externalId" json:"externalId"`
	CallbackURL         string `gorm:"column:callbackUrl" json:"callbackUrl"`
	RedirectURL         string `gorm:"column:redirectUrl" json:"redirectUrl"`
	DateAdded           int64  `gorm:"column:dateAdded;type:string" json:"dateAdded"`
	MerchantRequestID   string `gorm:"column:merchantRequestID" json:"merchantRequestID"`
	CheckoutRequestID   string `gorm:"column:checkoutRequestID" json:"checkoutRequestID"`
	ResponseCode        string `gorm:"column:responseCode" json:"responseCode"`
	ResponseDescription string `gorm:"column:responseDescription" json:"responseDescription"`
	CallbackStatus      string `gorm:"column:callbackStatus" json:"callbackStatus"`
}

func (TransactionModel) TableName() string {
	return "merchant_transactions"
}

func SaveTransaction(data *TransactionModel) error {
	db := database.GetConnection()
	return db.Create(data).Error
}

// GetTransactionByReference retrieves a fund transfer by its reference.
func GetTransactionByReference(reference string) (TransactionModel, error) {
	db := database.GetConnection()
	var Transaction TransactionModel
	err := db.Where("transaction_reference = ?", reference).First(&Transaction).Error
	return Transaction, err
}

// GetTransactionsByStatusAndDate retrieves transactions with status "success"
// and within a given date range for a specific merchant.
func GetTransactionsByStatusAndDate(merchantId string, startDate, endDate int64, transction_status string) ([]TransactionModel, error) {
	db := database.GetConnection()
	var transactions []TransactionModel

	err := db.Where("impalaMerchantId = ? AND transactionStatus = ? AND dateAdded BETWEEN ? AND ?",
		merchantId, transction_status, startDate, endDate).
		Find(&transactions).Error

	if err != nil {
		return nil, err
	}

	return transactions, nil
}

// Get all card transaction
func GetCadTransactions() ([]TransactionModel, error) {
	db := database.GetConnection()
	var transactions []TransactionModel

	err := db.Where("sourceOfFunds", "CARD").
		Find(&transactions).Error

	if err != nil {
		return nil, err
	}

	return transactions, nil
}

// GetAllTransactions retrieves all fund transfer records from the database.
func GetAllTransactions() ([]TransactionModel, error) {
	db := database.GetConnection()
	var Transactions []TransactionModel
	err := db.Find(&Transactions).Error
	return Transactions, err
}

// UpdateTransaction updates an existing fund transfer based on its ID.
func UpdateTransaction(id uint, updatedData map[string]interface{}) error {
	db := database.GetConnection()
	return db.Model(&TransactionModel{}).Where("id = ?", id).Updates(updatedData).Error
}

// DeleteTransaction deletes a fund transfer by its reference.
func DeleteTransaction(reference string) error {
	db := database.GetConnection()
	return db.Where("transaction_reference = ?", reference).Delete(&TransactionModel{}).Error
}

// GetTransactionByID retrieves a fund transfer by its ID.
func GetTransactionByID(id uint) (TransactionModel, error) {
	db := database.GetConnection()
	var Transaction TransactionModel

	err := db.First(&Transaction, id).Error
	if err != nil {
		fmt.Printf("Error fetching transaction with ID %d: %v\n", id, err)
		return Transaction, err
	}

	fmt.Printf("Fetched transaction: %+v\n", Transaction)
	return Transaction, nil
}

// UpdateTransactionByReference updates a fund transfer by its reference.
func UpdateTransactionByReference(reference string, updatedData map[string]interface{}) error {
	db := database.GetConnection()
	return db.Model(&TransactionModel{}).Where("transaction_reference = ?", reference).Updates(updatedData).Error
}

// Retrieve mobile payins
// can get all the payions our payouts
func GetMobileTransactions(reportType string, sourceOfFunds string) ([]TransactionModel, error) {
	db := database.GetConnection()
	var transactions []TransactionModel

	err := db.Where("transactionReport = ? AND sourceOfFunds = ?", reportType, sourceOfFunds).
		Find(&transactions).Error

	if err != nil {
		return nil, err
	}

	return transactions, nil
}
func GetMobilePayoutsTransactions() ([]TransactionModel, error) {
	db := database.GetConnection()
	var transactions []TransactionModel

	err := db.Where("transactionReport = ? AND sourceOfFunds = ?", "withdraw", "MPESA").
		Find(&transactions).Error

	if err != nil {
		return nil, err
	}

	return transactions, nil
}
func GetMobilePayinsTransactions() ([]TransactionModel, error) {
	db := database.GetConnection()
	var transactions []TransactionModel

	err := db.Where("transactionReport = ? AND sourceOfFunds = ?", "collection", "Merchant").
		Find(&transactions).Error

	if err != nil {
		return nil, err
	}

	return transactions, nil
}

// get merchant payins
func GetMerchantPayins(merchantId string) ([]TransactionModel, error) {
	db := database.GetConnection()
	var transactions []TransactionModel

	err := db.Where("impalaMerchantId = ? AND transactionReport = ?",
		merchantId, "collection").
		Find(&transactions).Error

	if err != nil {
		return nil, err
	}

	return transactions, nil
}

func GetMerchantPayouts(merchantId string) ([]TransactionModel, error) {
	db := database.GetConnection()
	var transactions []TransactionModel

	err := db.Where("impalaMerchantId = ? AND transactionReport = ?",
		merchantId, "withdraw").
		Find(&transactions).Error

	if err != nil {
		return nil, err
	}

	return transactions, nil
}

// Helper function to generate a secure random string
func GenerateSecureID() string {
	b := make([]byte, 16) // Generate 16 random bytes
	_, _ = rand.Read(b)
	return base64.URLEncoding.EncodeToString(b)
}

func GetTransactionByMerchantRequestID(merchantRequestID string) (TransactionModel, error) {
	db := database.GetConnection()
	var transaction TransactionModel

	// Enable debug mode to log the SQL query
	db = db.Debug()

	// Fetch the transaction
	err := db.Where("merchantRequestID = ?", merchantRequestID).First(&transaction).Error
	if err != nil {
		return transaction, err
	}

	// Print the transaction details
	// printTransaction(transaction)

	return transaction, nil
}
