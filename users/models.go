package users

import (
	"errors"
	"fmt"

	"com.mam-laka/database"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type UserModel struct {
	gorm.Model
	ID       uint   `gorm:"primary_key"`
	Name     string `gorm:"size:2048"`
	Phone    string `gorm:"size:2048"`
	Role     string `gorm:"size:2048"`
	Status   string `gorm:"size:2048"`
	Location string `gorm:"size:2048"`
	Password string `gorm:"column:password;not null"`
	Email    string `gorm:"column:email;unique_index"`
}

func (UserModel) TableName() string {
	return "users"
}

// Migrate the schema to the database if needed
func AutoMigrate() {
	db := database.GetConnection()
	db.AutoMigrate(&UserModel{})
}

// FindSingleUser finds a single User based on the provided condition
func FindSingleUser(condition interface{}) (UserModel, error) {
	db := database.GetConnection()
	var model UserModel
	err := db.Where(condition).First(&model).Error
	return model, err
}

// SaveSingleUser saves a single User to the database
func SaveSingleUser(data interface{}) error {
	db := database.GetConnection()
	err := db.Save(data).Error
	return err
}

// UpdateSingleUser updates a User with new data
func UpdateSingleUser(model *UserModel, data interface{}) error {
	db := database.GetConnection()
	err := db.Model(model).Updates(data).Error
	return err
}

// DeleteSingleUser deletes a User from the database
func DeleteSingleUser(model *UserModel) error {
	db := database.GetConnection()
	err := db.Delete(model).Error
	return err
}

// GetAllUsers gets all Users from the database
func GetAllUsers() ([]UserModel, error) {
	db := database.GetConnection()
	var models []UserModel
	err := db.Find(&models).Error
	return models, err
}

// fix codwa
func GetUserByID(id uint) (UserModel, error) {
	db := database.GetConnection()
	var User UserModel
	err := db.First(&User, id).Error
	return User, err
}

// get user by  email
func GetUserByEmail(email string) (UserModel, error) {
	db := database.GetConnection()
	var user UserModel
	err := db.Where("Email = ?", email).First(&user).Error
	return user, err
}

func GetUserByMerchantId(merchantID string) (int, error) {
	// Get database connection
	db := database.GetConnection()

	// Begin a database transaction
	tx := db.Begin()
	if tx.Error != nil {
		return 0, fmt.Errorf("failed to start transaction: %w", tx.Error)
	}

	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// Variable to store userId
	var userId int

	// Query to check if merchant exists and retrieve userId
	// A merchant can have several dashboard users linked to it (an operator
	// added alongside the original account). Without an explicit order the row
	// returned here is arbitrary, so which user's password authenticates the
	// API would vary between calls. Pin it to the earliest-created user -- the
	// account the merchant ID was issued with.
	err := tx.Raw("SELECT userId FROM merchant_access WHERE impalaMerchantId = ? ORDER BY userId LIMIT 1", merchantID).Scan(&userId).Error
	if err != nil {
		tx.Rollback()
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return 0, errors.New("merchant not found")
		}
		return 0, fmt.Errorf("failed to fetch merchant: %w", err)
	}

	// Commit the transaction if no errors
	if err := tx.Commit().Error; err != nil {
		return 0, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return userId, nil
}

func (u *UserModel) setPassword(Password string) error {
	//check Password legnth
	if len(Password) == 0 {
		return errors.New("Password should never be empty")
	}
	bytePassword := []byte(Password)
	// Make sure the second param `bcrypt generator cost` between [4, 32)
	PasswordHash, _ := bcrypt.GenerateFromPassword(bytePassword, bcrypt.DefaultCost)
	u.Password = string(PasswordHash)
	return nil
}

func (u *UserModel) CheckPassword(Password string) error {
	bytePassword := []byte(Password)
	byteHashedPassword := []byte(u.Password)

	// Debugging inputs
	fmt.Printf("Plaintext Password: %s\n", Password)
	fmt.Printf("Hashed Password from DB: %s\n", u.Password)

	return bcrypt.CompareHashAndPassword(byteHashedPassword, bytePassword)
}
