package drawings

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"com.mam-laka/database"
	"gorm.io/gorm"
)

type WithdrawalRequestModel struct {
	ID               uint    `gorm:"primaryKey;autoIncrement" json:"id"`
	ImpalaMerchantID string  `gorm:"column:impalaMerchantId;type:varchar(255);not null" json:"impalaMerchantId"`
	RequestedBy      int     `gorm:"column:requestedBy;type:int;not null" json:"requestedBy"`
	ApprovedBy       *int    `gorm:"column:approvedBy;type:int" json:"approvedBy"`
	Amount           float64 `gorm:"column:amount;type:float(100,2);not null" json:"amount"`
	Status           string  `gorm:"column:status;type:varchar(10);default:'PENDING';not null" json:"status"`
	DateRequested    int64   `gorm:"column:dateRequested;type:bigint;not null" json:"dateRequested"` // Changed to int64
	TransferType     string  `gorm:"column:transferType;type:enum('withdraw','transfer');default:'withdraw';not null" json:"transferType"`
	DateApproved     *int64  `gorm:"column:dateApproved;type:bigint" json:"dateApproved"` // Changed to *int64 for NULL support
	Comment          string  `gorm:"column:comment" json:"comment"`
}

// othe modedl
type PlatformEarningModel struct {
	ID                 uint      `json:"id" gorm:"column:id"`
	ImpalaMerchantID   string    `json:"impalaMerchantId" gorm:"column:impalaMerchantId"`
	AmountTransferred  float64   `json:"amountTransferred" gorm:"column:amountTransferred"`
	TransactionCharges float64   `json:"transactionCharges" gorm:"column:transactionCharges"`
	TransferDate       time.Time `json:"transferDate" gorm:"column:transferDate"`
}

// ernings function
func GetAllEarnings() ([]PlatformEarningModel, error) {
	db := database.GetConnection()
	var earnings []PlatformEarningModel
	err := db.Order("transferDate DESC").Find(&earnings).Error
	return earnings, err
}

func GetMerchantEarnings(merchantID string) ([]PlatformEarningModel, error) {
	db := database.GetConnection()
	var earnings []PlatformEarningModel
	err := db.Where("impalaMerchantId = ?", merchantID).
		Order("transferDate DESC").
		Find(&earnings).Error
	return earnings, err
}

func (WithdrawalRequestModel) TableName() string {
	return "withdrawal_requests"
}

// func (WithdrawalRequestModel) TableName() string {
// 	return "withdrawal_requests"
// }

// SaveWithdrawalRequest creates a new withdrawal request in the database.
func SaveWithdrawalRequest(data *WithdrawalRequestModel) error {
	fmt.Println(data)
	db := database.GetConnection()
	return db.Create(data).Error
}

// GetWithdrawalRequestByID retrieves a withdrawal request by its ID.
func GetWithdrawalRequestByID(id uint) (WithdrawalRequestModel, error) {
	db := database.GetConnection()
	var request WithdrawalRequestModel
	err := db.First(&request, id).Error
	return request, err
}

// GetAllWithdrawalRequests retrieves all withdrawal requests from the database.
func GetAllWithdrawalRequests() ([]WithdrawalRequestModel, error) {
	db := database.GetConnection()
	var requests []WithdrawalRequestModel
	err := db.Find(&requests).Error
	return requests, err
}

func GetAllRequests() ([]WithdrawalRequestModel, error) {
	db := database.GetConnection()
	var requests []WithdrawalRequestModel
	err := db.Find(&requests).Error
	return requests, err
}

// GetWithdrawalRequestsByStatus retrieves withdrawal requests by their status.
func GetWithdrawalRequestsByStatus(status string) ([]WithdrawalRequestModel, error) {
	db := database.GetConnection()
	var requests []WithdrawalRequestModel
	err := db.Where("status = ?", status).Find(&requests).Error
	return requests, err
}

// UpdateWithdrawalRequest updates an existing withdrawal request by its ID.
func UpdateWithdrawalRequest(id uint, updatedData map[string]interface{}) error {
	db := database.GetConnection()
	return db.Model(&WithdrawalRequestModel{}).Where("id = ?", id).Updates(updatedData).Error
}

// fancy update method
func UpdateSingleWithdrawalRequest(model *WithdrawalRequestModel, data interface{}) error {
	db := database.GetConnection()

	fmt.Printf("Model Before Update: %+v\n", model)
	fmt.Printf("Data to Update: %+v\n", data)

	result := db.Model(model).Updates(data)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return errors.New("no rows affected")
	}
	return nil
}

// DeleteWithdrawalRequest deletes a withdrawal request by its ID.
func DeleteWithdrawalRequest(id uint) error {
	db := database.GetConnection()
	return db.Where("id = ?", id).Delete(&WithdrawalRequestModel{}).Error
}

// transfer
func GetAllWalletToWalletRequests() ([]WithdrawalRequestModel, error) {
	db := database.GetConnection()
	var requests []WithdrawalRequestModel

	err := db.Where("transferType = ?", "transfer").
		Find(&requests).Error

	if err != nil {
		return nil, err
	}

	return requests, nil
}

// withdraw
func GetWithdrawalRequests() ([]WithdrawalRequestModel, error) {
	db := database.GetConnection()
	var requests []WithdrawalRequestModel

	err := db.Where("transferType = ?", "withdraw").
		Find(&requests).Error

	if err != nil {
		return nil, err
	}

	return requests, nil
}

func GetMerchantWithdrawalRequests(merchantID string) ([]WithdrawalRequestModel, error) {
	db := database.GetConnection()
	var requests []WithdrawalRequestModel

	err := db.Where("impalaMerchantId = ? AND transferType = ?", merchantID, "withdraw").
		Find(&requests).Error

	if err != nil {
		return nil, err
	}

	return requests, nil
}

func GetMerchantWalletToWalletRequests(merchantID string) ([]WithdrawalRequestModel, error) {
	db := database.GetConnection()
	var requests []WithdrawalRequestModel

	err := db.Where("impalaMerchantId = ? AND transferType = ?", merchantID, "transfer").
		Find(&requests).Error

	if err != nil {
		return nil, err
	}

	return requests, nil
}

// get merchatn request
func GetMerchantRequests(merchantID string) ([]WithdrawalRequestModel, error) {
	db := database.GetConnection()
	var requests []WithdrawalRequestModel

	err := db.Where("impalaMerchantId = ?", merchantID).
		Find(&requests).Error

	if err != nil {
		return nil, err
	}

	return requests, nil
}

// wallet to wallet functionality

type TransferRequest struct {
	ImpalaMerchantId string  `json:"impalaMerchantId"`
	Amount           float64 `json:"amount" binding:"required,gt=0"`
	Currency         string  `json:"currency" binding:"required"`
}

func WalletTransfer(transferRequest *TransferRequest) error {
	db := database.GetConnection()

	currency := strings.ToLower(transferRequest.Currency)
	balanceField := fmt.Sprintf("%sBalance", currency)

	tx := db.Begin()
	if tx.Error != nil {
		return fmt.Errorf("failed to start transaction: %w", tx.Error)
	}

	var collectionBalance sql.NullFloat64
	query := fmt.Sprintf("SELECT %s FROM merchant_collection_balance WHERE impalaMerchantId = ?", balanceField)
	err := tx.Raw(query, transferRequest.ImpalaMerchantId).Scan(&collectionBalance).Error
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to fetch collection balance: %w", err)
	}
	if !collectionBalance.Valid {
		tx.Rollback()
		return fmt.Errorf("no %s balance found for merchant: %s", currency, transferRequest.ImpalaMerchantId)
	}

	log.Printf("Collection balance before transfer: %.2f", collectionBalance.Float64)
	log.Printf("Transfer amount: %.2f", transferRequest.Amount)

	transactionCharges := transferRequest.Amount * 0.015
	totalDeduction := transferRequest.Amount + transactionCharges

	if collectionBalance.Float64 < totalDeduction {
		tx.Rollback()
		return fmt.Errorf("insufficient balance: %.2f < %.2f", collectionBalance.Float64, totalDeduction)
	}

	// Deduct from collection balance
	updateCollection := fmt.Sprintf("UPDATE merchant_collection_balance SET %s = %s - ? WHERE impalaMerchantId = ?", balanceField, balanceField)
	if err := tx.Exec(updateCollection, totalDeduction, transferRequest.ImpalaMerchantId).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to deduct from collection balance: %w", err)
	}

	// Add to merchant wallet balance
	updateWallet := fmt.Sprintf("UPDATE merchant_balances SET %s = COALESCE(%s, 0) + ? WHERE impalaMerchantId = ?", balanceField, balanceField)
	if err := tx.Exec(updateWallet, transferRequest.Amount, transferRequest.ImpalaMerchantId).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to add to merchant wallet: %w", err)
	}

	// Record platform earnings
	if err := tx.Exec(
		`INSERT INTO platform_earning_models (impalaMerchantId, amountTransferred, transactionCharges, currency) 
		 VALUES (?, ?, ?, ?)`,
		transferRequest.ImpalaMerchantId, transferRequest.Amount, transactionCharges, transferRequest.Currency,
	).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to record platform earnings: %w", err)
	}

	// Fetch updated balances
	var updatedCollectionBalance, updatedMerchantBalance sql.NullFloat64

	queryCollection := fmt.Sprintf("SELECT %s FROM merchant_collection_balance WHERE impalaMerchantId = ?", balanceField)
	if err := tx.Raw(queryCollection, transferRequest.ImpalaMerchantId).Scan(&updatedCollectionBalance).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to fetch updated collection balance: %w", err)
	}
	if !updatedCollectionBalance.Valid {
		tx.Rollback()
		return fmt.Errorf("collection balance is NULL after update")
	}

	queryMerchant := fmt.Sprintf("SELECT %s FROM merchant_balances WHERE impalaMerchantId = ?", balanceField)
	if err := tx.Raw(queryMerchant, transferRequest.ImpalaMerchantId).Scan(&updatedMerchantBalance).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to fetch updated merchant balance: %w", err)
	}
	if !updatedMerchantBalance.Valid {
		tx.Rollback()
		return fmt.Errorf("merchant balance is NULL after update")
	}

	log.Printf("Collection balance after transfer: %.2f", updatedCollectionBalance.Float64)
	log.Printf("Merchant balance after transfer: %.2f", updatedMerchantBalance.Float64)

	if err := tx.Commit().Error; err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// WalletTransferBackToCollection handles transferring money from the merchant's wallet back to the collection table.
func WalletTransferBackToCollection(transferRequest *TransferRequest) error {
	db := database.GetConnection()

	// Begin a database transaction
	tx := db.Begin()
	if tx.Error != nil {
		return fmt.Errorf("failed to start transaction: %w", tx.Error)
	}

	// Fetch merchant wallet balance
	var walletBalance float64
	err := tx.Raw("SELECT kesBalance FROM merchant_wallet_balance WHERE impalaMerchantId = ?", transferRequest.ImpalaMerchantId).Scan(&walletBalance).Error
	if err != nil {
		tx.Rollback()
		if err == gorm.ErrRecordNotFound {
			return fmt.Errorf("merchant not found in wallet balance")
		}
		return fmt.Errorf("failed to fetch wallet balance: %w", err)
	}

	// Check if the merchant has sufficient balance
	if walletBalance < transferRequest.Amount {
		tx.Rollback()
		return fmt.Errorf("insufficient balance in wallet")
	}

	// Deduct from merchant wallet balance
	err = tx.Exec("UPDATE merchant_wallet_balance SET kesBalance = kesBalance - ? WHERE impalaMerchantId = ?", transferRequest.Amount, transferRequest.ImpalaMerchantId).Error
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to deduct from wallet balance: %w", err)
	}

	// Add to merchant collection balance
	err = tx.Exec("UPDATE merchant_collection_balance SET kesBalance = kesBalance + ? WHERE impalaMerchantId = ?", transferRequest.Amount, transferRequest.ImpalaMerchantId).Error
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to add to collection balance: %w", err)
	}

	// Commit the transaction if everything is successful
	if err := tx.Commit().Error; err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	// Send success response
	return nil
}
