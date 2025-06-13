package virtualcards

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"time"

	"com.mam-laka/database"
	"gorm.io/gorm"
)

// CardHolderModel represents a card holder in the database
type CardHolderModel struct {
	gorm.Model
	ID              uint   `gorm:"primary_key"`
	MerchantOrderNo string `gorm:"size:255;unique_index"`
	HolderID        string `gorm:"size:255;unique_index"`
	CardTypeID      string `gorm:"size:255"`
	AreaCode        string `gorm:"size:20"`
	Mobile          string `gorm:"size:50"`
	Email           string `gorm:"size:255"`
	FirstName       string `gorm:"size:255"`
	LastName        string `gorm:"size:255"`
	BirthDay        string `gorm:"size:20"`
	Country         string `gorm:"size:10"`
	Town            string `gorm:"size:255"`
	Address         string `gorm:"size:500"`
	PostCode        string `gorm:"size:20"`
	Status          string `gorm:"size:50"`
	StatusStr       string `gorm:"size:255"`
	Message         string `gorm:"size:500"`
	UserID          uint   `gorm:"index"`          // Foreign key to users table
	MerchantID      string `gorm:"size:255;index"` // Merchant identifier
}

func (CardHolderModel) TableName() string {
	return "card_holders"
}

// VirtualCardModel represents a virtual card in the database
type VirtualCardModel struct {
	gorm.Model
	ID               uint    `gorm:"primary_key"`
	OrderNo          string  `gorm:"size:255;unique_index"`
	MerchantOrderNo  string  `gorm:"size:255;unique_index"`
	CardTypeID       int     `gorm:"size:255"`
	HolderID         string  `gorm:"size:255;index"`
	CardNo           string  `gorm:"size:255;unique_index"`
	CardNumber       string  `gorm:"size:20"`
	CVV              string  `gorm:"size:10"`
	ValidPeriod      string  `gorm:"size:20"`
	Currency         string  `gorm:"size:10"`
	Amount           string  `gorm:"size:50"`
	Fee              string  `gorm:"size:50"`
	ReceivedAmount   string  `gorm:"size:50"`
	ReceivedCurrency string  `gorm:"size:10"`
	Type             string  `gorm:"size:50"`
	Status           string  `gorm:"size:50"`
	StatusStr        string  `gorm:"size:255"`
	TransactionTime  int64   `gorm:"index"`
	BindTime         int64   `gorm:"index"`
	Balance          float64 `gorm:"type:decimal(10,2);default:0"`
	UserID           uint    `gorm:"index"`          // Foreign key to users table
	CardHolderID     uint    `gorm:"index"`          // Foreign key to card_holders table
	MerchantID       string  `gorm:"size:255;index"` // Merchant identifier
	CallbackURL      string  `gorm:"size:500"`       // URL for callback notifications
	CallbackStatus   string  `gorm:"size:50"`        // Status of the callback (e.g., sent , not sent)
}

func (VirtualCardModel) TableName() string {
	return "virtual_cards"
}

// CardTransactionModel represents card transactions in the database
type CardTransactionModel struct {
	gorm.Model
	ID               uint    `gorm:"primary_key"`
	CardID           uint    `gorm:"index"` // Foreign key to virtual_cards table
	CardNo           string  `gorm:"size:255;index"`
	TransactionType  string  `gorm:"size:50"` // deposit, withdrawal, payment
	OrderNo          string  `gorm:"size:255;unique_index"`
	MerchantOrderNo  string  `gorm:"size:255"`
	Amount           float64 `gorm:"type:decimal(10,2)"`
	Fee              float64 `gorm:"type:decimal(10,2);default:0"`
	Currency         string  `gorm:"size:10"`
	ReceivedAmount   float64 `gorm:"type:decimal(10,2)"`
	ReceivedCurrency string  `gorm:"size:10"`
	Status           string  `gorm:"size:50"`
	StatusStr        string  `gorm:"size:255"`
	TransactionTime  int64   `gorm:"index"`
	Description      string  `gorm:"size:500"`
	BalanceAfter     float64 `gorm:"type:decimal(10,2)"`
	BalanceBefore    float64 `gorm:"type:decimal(10,2)"`
	UserID           uint    `gorm:"index"`          // Foreign key to users table
	MerchantID       string  `gorm:"size:255;index"` // Merchant identifier
}

func (CardTransactionModel) TableName() string {
	return "card_transactions"
}

// CardTypeModel represents different card types available
type CardTypeModel struct {
	gorm.Model
	ID          uint    `gorm:"primary_key"`
	CardTypeID  string  `gorm:"size:255;unique_index"`
	Name        string  `gorm:"size:255"`
	Description string  `gorm:"size:500"`
	Currency    string  `gorm:"size:10"`
	MinAmount   float64 `gorm:"type:decimal(10,2)"`
	MaxAmount   float64 `gorm:"type:decimal(10,2)"`
	Fee         float64 `gorm:"type:decimal(10,2)"`
	IsActive    bool    `gorm:"default:true"`
	MerchantID  string  `gorm:"size:255;index"` // Merchant identifier
}

func (CardTypeModel) TableName() string {
	return "card_types"
}

// Auto migrate all virtual card related tables
func AutoMigrate() {
	db := database.GetConnection()
	db.AutoMigrate(&CardHolderModel{})
	db.AutoMigrate(&VirtualCardModel{})
	db.AutoMigrate(&CardTransactionModel{})
	db.AutoMigrate(&CardTypeModel{})
}

// ===============================
// CardHolderModel Methods
// ===============================

func FindSingleCardHolder(condition interface{}) (CardHolderModel, error) {
	db := database.GetConnection()
	var model CardHolderModel
	err := db.Where(condition).First(&model).Error
	return model, err
}

func SaveSingleCardHolder(data interface{}) error {
	db := database.GetConnection()
	err := db.Save(data).Error
	return err
}

func UpdateSingleCardHolder(model *CardHolderModel, data interface{}) error {
	db := database.GetConnection()
	err := db.Model(model).Updates(data).Error
	return err
}

func DeleteSingleCardHolder(model *CardHolderModel) error {
	db := database.GetConnection()
	err := db.Delete(model).Error
	return err
}

func GetAllCardHolders() ([]CardHolderModel, error) {
	db := database.GetConnection()
	var models []CardHolderModel
	err := db.Find(&models).Error
	return models, err
}

func GetCardHolderByID(id uint) (CardHolderModel, error) {
	db := database.GetConnection()
	var holder CardHolderModel
	err := db.First(&holder, id).Error
	return holder, err
}

func GetCardHolderByHolderID(holderID string) (CardHolderModel, error) {
	db := database.GetConnection()
	var holder CardHolderModel
	err := db.Where("holder_id = ?", holderID).First(&holder).Error
	return holder, err
}

func GetCardHoldersByUserID(userID uint) ([]CardHolderModel, error) {
	db := database.GetConnection()
	var holders []CardHolderModel
	err := db.Where("user_id = ?", userID).Find(&holders).Error
	return holders, err
}

func GetCardHoldersByMerchantID(merchant string) ([]CardHolderModel, error) {
	db := database.GetConnection()
	var holders []CardHolderModel
	err := db.Where("merchant_id = ?", merchant).Find(&holders).Error
	return holders, err
}

// ===============================
// VirtualCardModel Methods
// ===============================
func GetVirtualCardsByMerchantID(merchantID string) ([]VirtualCardModel, error) {
	db := database.GetConnection()
	var cards []VirtualCardModel
	err := db.Where("merchant_id = ?", merchantID).Find(&cards).Error
	return cards, err
}

// UpdateVirtualCardFromCallback updates the virtual card record by order number using callback data
func UpdateVirtualCardFromCallback(callbackData map[string]string) (VirtualCardModel, error) {
	db := database.GetConnection()

	orderNo := callbackData["merchantOrderNo"]

	// Step 1: Find the virtual card
	var card VirtualCardModel
	if err := db.Where("merchant_order_no = ?", orderNo).First(&card).Error; err != nil {
		return card, errors.New("virtual card with order number not found")
	}

	// Step 2: Get callback URL
	callbackURL := card.CallbackURL
	if callbackURL == "" {
		return card, errors.New("callback URL not set for the card")
	}

	// Step 3: Convert callbackData to JSON
	jsonData, err := json.Marshal(callbackData)
	if err != nil {
		return card, fmt.Errorf("failed to marshal callback data: %v", err)
	}

	// Step 4: Send POST request to callback URL
	resp, err := http.Post(callbackURL, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return card, fmt.Errorf("failed to send callback: %v", err)
	}
	defer resp.Body.Close()
	// send it to the original callback as well
	_, err2 := http.Post("https://kcb-buni.mam-laka.com/api/callback/card/", "application/json", bytes.NewBuffer(jsonData))
	if err2 != nil {
		return card, fmt.Errorf("failed to send callback: %v", err)
	}
	defer resp.Body.Close()
	// the rest 

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return card, fmt.Errorf("callback returned non-200 status: %d, body: %s", resp.StatusCode, string(body))
	}

	// Step 5: Parse float fields
	// amount, _ := strconv.ParseFloat(callbackData["amount"], 64)
	// fee, _ := strconv.ParseFloat(callbackData["fee"], 64)
	receivedAmount, _ := strconv.ParseFloat(callbackData["receivedAmount"], 64)

	// Step 6: Update card fields
	card.CardNo = callbackData["cardNo"]
	card.Currency = callbackData["currency"]
	card.Amount = callbackData["amount"]
	card.Fee = callbackData["fee"]
	card.ReceivedAmount = callbackData["receivedAmount"]
	card.ReceivedCurrency = callbackData["receivedCurrency"]
	card.Type = callbackData["type"]
	card.Status = callbackData["status"]
	card.CallbackStatus = "SENT"
	card.UpdatedAt = time.Now()

	// Optional: Update numeric balance or amount if used
	card.Balance = receivedAmount // Only if this is your logic

	// Step 7: Save updated card
	if err := db.Save(&card).Error; err != nil {
		return card, fmt.Errorf("failed to save updated card: %v", err)
	}

	log.Println("Callback processed and card updated successfully")
	return card, nil
}

func FindSingleVirtualCard(condition interface{}) (VirtualCardModel, error) {
	db := database.GetConnection()
	var model VirtualCardModel
	err := db.Where(condition).First(&model).Error
	return model, err
}

func SaveSingleVirtualCard(data interface{}) error {
	db := database.GetConnection()
	err := db.Save(data).Error
	return err
}

func UpdateSingleVirtualCard(model *VirtualCardModel, data interface{}) error {
	db := database.GetConnection()
	err := db.Model(model).Updates(data).Error
	return err
}

func DeleteSingleVirtualCard(model *VirtualCardModel) error {
	db := database.GetConnection()
	err := db.Delete(model).Error
	return err
}

func GetAllVirtualCards() ([]VirtualCardModel, error) {
	db := database.GetConnection()
	var models []VirtualCardModel
	err := db.Find(&models).Error
	return models, err
}

func GetVirtualCardByID(id uint) (VirtualCardModel, error) {
	db := database.GetConnection()
	var card VirtualCardModel
	err := db.First(&card, id).Error
	return card, err
}

func GetVirtualCardByCardNo(cardNo string) (VirtualCardModel, error) {
	db := database.GetConnection()
	var card VirtualCardModel
	err := db.Where("card_no = ?", cardNo).First(&card).Error
	return card, err
}

func GetVirtualCardsByUserID(userID uint) ([]VirtualCardModel, error) {
	db := database.GetConnection()
	var cards []VirtualCardModel
	err := db.Where("user_id = ?", userID).Find(&cards).Error
	return cards, err
}

func GetVirtualCardsByHolderID(holderID string) ([]VirtualCardModel, error) {
	db := database.GetConnection()
	var cards []VirtualCardModel
	err := db.Where("holder_id = ?", holderID).Find(&cards).Error
	return cards, err
}

// Update card balance
func UpdateCardBalance(cardID uint, newBalance float64) error {
	db := database.GetConnection()
	err := db.Model(&VirtualCardModel{}).Where("id = ?", cardID).Update("balance", newBalance).Error
	return err
}

// ===============================
// CardTransactionModel Methods
// ===============================

func FindSingleCardTransaction(condition interface{}) (CardTransactionModel, error) {
	db := database.GetConnection()
	var model CardTransactionModel
	err := db.Where(condition).First(&model).Error
	return model, err
}

func SaveSingleCardTransaction(data interface{}) error {
	db := database.GetConnection()
	err := db.Save(data).Error
	return err
}

func UpdateSingleCardTransaction(model *CardTransactionModel, data interface{}) error {
	db := database.GetConnection()
	err := db.Model(model).Updates(data).Error
	return err
}

func DeleteSingleCardTransaction(model *CardTransactionModel) error {
	db := database.GetConnection()
	err := db.Delete(model).Error
	return err
}

func GetAllCardTransactions() ([]CardTransactionModel, error) {
	db := database.GetConnection()
	var models []CardTransactionModel
	err := db.Find(&models).Error
	return models, err
}

func GetCardTransactionByID(id uint) (CardTransactionModel, error) {
	db := database.GetConnection()
	var transaction CardTransactionModel
	err := db.First(&transaction, id).Error
	return transaction, err
}

func GetTransactionsByCardID(cardID uint) ([]CardTransactionModel, error) {
	db := database.GetConnection()
	var transactions []CardTransactionModel
	err := db.Where("card_id = ?", cardID).Order("created_at DESC").Find(&transactions).Error
	return transactions, err
}

func GetTransactionsByUserID(userID uint) ([]CardTransactionModel, error) {
	db := database.GetConnection()
	var transactions []CardTransactionModel
	err := db.Where("user_id = ?", userID).Order("created_at DESC").Find(&transactions).Error
	return transactions, err
}

func GetTransactionsByDateRange(cardID uint, startDate, endDate time.Time) ([]CardTransactionModel, error) {
	db := database.GetConnection()
	var transactions []CardTransactionModel
	err := db.Where("card_id = ? AND created_at BETWEEN ? AND ?", cardID, startDate, endDate).
		Order("created_at DESC").Find(&transactions).Error
	return transactions, err
}

// ===============================
// CardTypeModel Methods
// ===============================

func FindSingleCardType(condition interface{}) (CardTypeModel, error) {
	db := database.GetConnection()
	var model CardTypeModel
	err := db.Where(condition).First(&model).Error
	return model, err
}

func SaveSingleCardType(data interface{}) error {
	db := database.GetConnection()
	err := db.Save(data).Error
	return err
}

func UpdateSingleCardType(model *CardTypeModel, data interface{}) error {
	db := database.GetConnection()
	err := db.Model(model).Updates(data).Error
	return err
}

func DeleteSingleCardType(model *CardTypeModel) error {
	db := database.GetConnection()
	err := db.Delete(model).Error
	return err
}

func GetAllCardTypes() ([]CardTypeModel, error) {
	db := database.GetConnection()
	var models []CardTypeModel
	err := db.Find(&models).Error
	return models, err
}

func GetActiveCardTypes() ([]CardTypeModel, error) {
	db := database.GetConnection()
	var models []CardTypeModel
	err := db.Where("is_active = ?", true).Find(&models).Error
	return models, err
}

func GetCardTypeByID(id uint) (CardTypeModel, error) {
	db := database.GetConnection()
	var cardType CardTypeModel
	err := db.First(&cardType, id).Error
	return cardType, err
}

func GetCardTypeByCardTypeID(cardTypeID string) (CardTypeModel, error) {
	db := database.GetConnection()
	var cardType CardTypeModel
	err := db.Where("card_type_id = ?", cardTypeID).First(&cardType).Error
	return cardType, err
}

// ===============================
// Business Logic Methods
// ===============================

// CreateCardHolderWithUser creates a card holder and associates it with a user
func CreateCardHolderWithUser(userID uint, holderData CardHolderModel) (CardHolderModel, error) {
	db := database.GetConnection()

	// Begin transaction
	tx := db.Begin()
	if tx.Error != nil {
		return CardHolderModel{}, fmt.Errorf("failed to start transaction: %w", tx.Error)
	}

	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// Set user ID
	holderData.UserID = userID

	// Save card holder
	if err := tx.Save(&holderData).Error; err != nil {
		tx.Rollback()
		return CardHolderModel{}, fmt.Errorf("failed to create card holder: %w", err)
	}

	// Commit transaction
	if err := tx.Commit().Error; err != nil {
		return CardHolderModel{}, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return holderData, nil
}

// CreateVirtualCardWithTransaction creates a virtual card and logs the initial transaction
func CreateVirtualCardWithTransaction(userID uint, cardData VirtualCardModel) (VirtualCardModel, error) {
	db := database.GetConnection()

	// Begin transaction
	tx := db.Begin()
	if tx.Error != nil {
		return VirtualCardModel{}, fmt.Errorf("failed to start transaction: %w", tx.Error)
	}

	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// Set user ID
	cardData.UserID = userID

	// Save virtual card
	if err := tx.Save(&cardData).Error; err != nil {
		tx.Rollback()
		return VirtualCardModel{}, fmt.Errorf("failed to create virtual card: %w", err)
	}

	// Create initial transaction record
	transaction := CardTransactionModel{
		CardID:          cardData.ID,
		CardNo:          cardData.CardNo,
		TransactionType: "create",
		OrderNo:         cardData.OrderNo,
		MerchantOrderNo: cardData.MerchantOrderNo,
		Amount:          0,
		Currency:        cardData.Currency,
		Status:          cardData.Status,
		StatusStr:       cardData.StatusStr,
		TransactionTime: cardData.TransactionTime,
		Description:     "Card creation",
		BalanceAfter:    cardData.Balance,
		BalanceBefore:   0,
		UserID:          userID,
	}

	if err := tx.Save(&transaction).Error; err != nil {
		tx.Rollback()
		return VirtualCardModel{}, fmt.Errorf("failed to create transaction record: %w", err)
	}

	// Commit transaction
	if err := tx.Commit().Error; err != nil {
		return VirtualCardModel{}, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return cardData, nil
}

// ProcessCardDeposit processes a card deposit with balance update and transaction logging
func ProcessCardDeposit(cardID uint, userID uint, amount float64, orderNo, merchantOrderNo string) error {
	if amount < 10.0 {
		return errors.New("minimum deposit amount is $10 USD")
	}

	db := database.GetConnection()

	// Begin transaction
	tx := db.Begin()
	if tx.Error != nil {
		return fmt.Errorf("failed to start transaction: %w", tx.Error)
	}

	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// Get current card
	var card VirtualCardModel
	if err := tx.First(&card, cardID).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to find card: %w", err)
	}

	// Calculate new balance
	oldBalance := card.Balance
	newBalance := oldBalance + amount

	// Update card balance
	if err := tx.Model(&card).Update("balance", newBalance).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to update card balance: %w", err)
	}

	// Create transaction record
	transaction := CardTransactionModel{
		CardID:           cardID,
		CardNo:           card.CardNo,
		TransactionType:  "deposit",
		OrderNo:          orderNo,
		MerchantOrderNo:  merchantOrderNo,
		Amount:           amount,
		Currency:         "USD",
		ReceivedAmount:   amount,
		ReceivedCurrency: "USD",
		Status:           "completed",
		StatusStr:        "Deposit Successful",
		TransactionTime:  time.Now().Unix(),
		Description:      fmt.Sprintf("Card deposit of $%.2f", amount),
		BalanceAfter:     newBalance,
		BalanceBefore:    oldBalance,
		UserID:           userID,
	}

	if err := tx.Save(&transaction).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to create transaction record: %w", err)
	}

	// Commit transaction
	if err := tx.Commit().Error; err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// GetCardBalance gets the current balance of a card
func GetCardBalance(cardID uint) (float64, error) {
	db := database.GetConnection()
	var card VirtualCardModel
	err := db.Select("balance").First(&card, cardID).Error
	if err != nil {
		return 0, err
	}
	return card.Balance, nil
}

// GetUserCardsSummary gets a summary of all cards for a user
func GetUserCardsSummary(userID uint) (map[string]interface{}, error) {
	db := database.GetConnection()

	var totalCards int64
	var activeCards int64
	var totalBalance float64

	// Count total cards
	db.Model(&VirtualCardModel{}).Where("user_id = ?", userID).Count(&totalCards)

	// Count active cards
	db.Model(&VirtualCardModel{}).Where("user_id = ? AND status = ?", userID, "active").Count(&activeCards)

	// Sum total balance
	db.Model(&VirtualCardModel{}).Where("user_id = ?", userID).Select("COALESCE(SUM(balance), 0)").Scan(&totalBalance)

	summary := map[string]interface{}{
		"total_cards":   totalCards,
		"active_cards":  activeCards,
		"total_balance": totalBalance,
	}

	return summary, nil
}
