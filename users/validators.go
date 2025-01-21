package users

import (
	"com.mam-laka/database"
	"github.com/gin-gonic/gin"
)

type UserModelValidator struct {
	User struct {
		Name     string `form:"name" json:"name" binding:"required,min=4"`
		Email    string `form:"email" json:"email" binding:"required,email"`
		Phone    string `form:"phone" json:"phone" binding:"required,min=10"`
		Password string `form:"password" json:"password" binding:"required,min=4"`
		Role     string `form:"role" json:"role" binding:"required"`
		Status   string `form:"status" json:"status" binding:"required"`
		Location string `form:"location" json:"location" binding:"required"`
	} `json:"user"`
	userModel UserModel `json:"-"`
}

// Bind binds the request data to the UserModelValidator.
func (v *UserModelValidator) Bind(c *gin.Context) error {
	err := database.Bind(c, v)
	if err != nil {
		return err
	}

	v.userModel.Name = v.User.Name
	v.userModel.Email = v.User.Email
	v.userModel.Phone = v.User.Phone
	v.userModel.Role = v.User.Role
	v.userModel.Status = v.User.Status
	v.userModel.Location = v.User.Location

	if v.User.Password != "SJSKKSKSFIKDKDFLWWOWO1873300" {
		v.userModel.setPassword(v.User.Password)
	}

	return nil
}

// NewUserModelValidator creates a new UserModelValidator instance.
func NewUserModelValidator() UserModelValidator {
	return UserModelValidator{}
}

// NewUserModelValidatorFillWith creates a new UserModelValidator instance and fills it with user model data.
func NewUserModelValidatorFillWith(userModel UserModel) UserModelValidator {
	return UserModelValidator{
		User: struct {
			Name     string `form:"name" json:"name" binding:"required,min=4"`
			Email    string `form:"email" json:"email" binding:"required,email"`
			Phone    string `form:"phone" json:"phone" binding:"required,min=10"`
			Password string `form:"password" json:"password" binding:"required,min=4"`
			Role     string `form:"role" json:"role" binding:"required"`
			Status   string `form:"status" json:"status" binding:"required"`
			Location string `form:"location" json:"location" binding:"required"`
		}{
			Name:     userModel.Name,
			Email:    userModel.Email,
			Phone:    userModel.Phone,
			Password: userModel.Password,
			Role:     userModel.Role,
			Status:   userModel.Status,
			Location: userModel.Location,
		},
	}
}

// LoginValidator represents a validator for user login.
func phone() bool {
	return true
}

type LoginValidator struct {
	User struct {
		Email    string `form:"email" json:"email" binding:"required,min=4"`
		Password string `form:"password" json:"password" binding:"required,min=2"`
	} `json:"user"`
	userModel UserModel `json:"-"`
}

func (self *LoginValidator) Bind(c *gin.Context) error {
	err := database.Bind(c, self)
	// err := database.Bind
	if err != nil {
		return err
	}

	self.userModel.Email = self.User.Email
	return nil
}

// You can put the default value of a Validator here
func NewLoginValidator() LoginValidator {
	loginValidator := LoginValidator{}
	return loginValidator
}
