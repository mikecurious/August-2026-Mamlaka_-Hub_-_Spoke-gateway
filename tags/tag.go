package tags

// package tags

import (
	"com.mam-laka/database"
	"gorm.io/gorm"
)

type TagModel struct {
	gorm.Model
	ID         uint   `gorm:"column:id;primaryKey;autoIncrement"`
	Tag        string `gorm:"column:tag;uniqueIndex;not null"`
	MerchantID string `gorm:"column:merchant_id;not null"`
}

// Migrate the schema to the database if needed
func AutoMigrate() {
	db := database.GetConnection()
	db.AutoMigrate(&TagModel{})
}

// CreateTag creates a new tag in the database
func CreateTag(tag *TagModel) error {
	db := database.GetConnection()
	err := db.Create(tag).Error
	return err
}

// GetMerchantIDByTag retrieves the merchant ID based on the supplied tag
func GetMerchantIDByTag(tag string) (string, error) {
	db := database.GetConnection()
	var tagModel TagModel
	err := db.Where("tag = ?", tag).First(&tagModel).Error
	if err != nil {
		return "", err
	}
	return tagModel.MerchantID, nil
}

// FindTagByID finds a tag by its ID
func FindTagByID(id uint) (TagModel, error) {
	db := database.GetConnection()
	var tag TagModel
	err := db.First(&tag, id).Error
	return tag, err
}

// UpdateTag updates a tag with new data
func UpdateTag(tag *TagModel, data map[string]interface{}) error {
	db := database.GetConnection()
	err := db.Model(tag).Updates(data).Error
	return err
}

// DeleteTag deletes a tag from the database
func DeleteTag(tag *TagModel) error {
	db := database.GetConnection()
	err := db.Delete(tag).Error
	return err
}

// GetAllTags retrieves all tags from the database
func GetAllTags() ([]TagModel, error) {
	db := database.GetConnection()
	var tags []TagModel
	err := db.Find(&tags).Error
	return tags, err
}
