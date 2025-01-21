package users

import (
	"fmt"
	"math/rand"
	"net/http"
	"strconv"

	"com.mam-laka/database"

	"github.com/gin-gonic/gin"
)

func Create(router *gin.RouterGroup) {
	router.POST("/create", CreateUser)
	// router.POST("/login", Login)
	router.GET("/read/:id", ReadSingleUser)
	router.GET("/reset/:email", PasswordReset)
	router.PUT("/update/:id", UpdateUser)
	router.DELETE("/delete/:id", DeleteUser)
	router.GET("/list", UsersList)
}

func CreateUser(c *gin.Context) {
	modelValidator := NewUserModelValidator()
	if err := modelValidator.Bind(c); err != nil {
		c.JSON(http.StatusUnprocessableEntity, database.NewValidatorError(err))
		return
	}

	if err := SaveSingleUser(&modelValidator.userModel); err != nil {
		c.JSON(http.StatusUnprocessableEntity, database.NewError("database", err))
		return
	}

	c.Set("my_User_model", modelValidator.userModel)
	serializer := NewUserSerializer(c, modelValidator.userModel)
	c.JSON(http.StatusCreated, gin.H{"User": serializer.Response()})

	fmt.Println("User saved ...")
}

// func Login(c *gin.Context) {
// 	loginValidator := NewLoginValidator()
// 	if err := loginValidator.Bind(c); err != nil {
// 		c.JSON(http.StatusUnprocessableEntity, database.NewValidatorError(err))
// 		return
// 	}

// 	// Find user by email
// 	userModel, err := FindSingleUser(&UserModel{Email: loginValidator.User.Email})
// 	if err != nil {
// 		if err == gorm.ErrRecordNotFound {
// 			c.JSON(http.StatusForbidden, database.NewError("login", errors.New("not registered email or invalid password")))
// 		} else {
// 			c.JSON(http.StatusInternalServerError, database.NewError("login", err))
// 		}
// 		return
// 	}

// 	// Check password
// 	if err := userModel.CheckPassword(loginValidator.User.Password); err != nil {
// 		c.JSON(http.StatusForbidden, database.NewError("login", errors.New("not registered email or invalid password")))
// 		return
// 	}

// 	// Generate a token
// 	token, err := auth.CreateToken(userModel.Email) // Assuming `userModel.Email` is the username
// 	if err != nil {
// 		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
// 		return
// 	}

// 	// Return user details and token
// 	serializer := UserSerializer{c, userModel}
// 	c.JSON(http.StatusOK, gin.H{
// 		"user":  serializer.Response(),
// 		"token": token,
// 	})
// }

func ReadSingleUser(c *gin.Context) {
	UserID := c.Param("id")
	UserIDUint, err := strconv.ParseUint(UserID, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid User ID"})
		return
	}

	UserModel, err := GetUserByID(uint(UserIDUint))
	if err != nil {
		c.JSON(http.StatusInternalServerError, database.NewError("User", err))
		return
	}

	serializer := NewUserSerializer(c, UserModel)
	c.JSON(http.StatusOK, gin.H{"User": serializer.Response()})
}

func UpdateUser(c *gin.Context) {
	UserID := c.Param("id")
	UserIDUint, err := strconv.ParseUint(UserID, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid User ID"})
		return
	}
	UserModel, err := GetUserByID(uint(UserIDUint))
	if err != nil {
		c.JSON(http.StatusInternalServerError, database.NewError("User", err))
		return
	}

	// Bind and update UserModel with new data
	modelValidator := NewUserModelValidatorFillWith(UserModel)
	if err := modelValidator.Bind(c); err != nil {
		c.JSON(http.StatusUnprocessableEntity, database.NewValidatorError(err))
		return
	}

	// Call UpdateSingleUser function with the UserModel and updated data
	if err := UpdateSingleUser(&UserModel, modelValidator.userModel); err != nil {
		c.JSON(http.StatusUnprocessableEntity, database.NewError("database", err))
		return
	}

	serializer := NewUserSerializer(c, UserModel)
	c.JSON(http.StatusOK, gin.H{"User": serializer.Response()})
}

func DeleteUser(c *gin.Context) {
	UserID := c.Param("id")
	UserIDUint, err := strconv.ParseUint(UserID, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid User ID"})
		return
	}
	UserModel, err := GetUserByID(uint(UserIDUint))
	if err != nil {
		c.JSON(http.StatusInternalServerError, database.NewError("User", err))
		return
	}

	if err := DeleteSingleUser(&UserModel); err != nil {
		c.JSON(http.StatusUnprocessableEntity, database.NewError("database", err))
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "User deleted successfully"})
}

func UsersList(c *gin.Context) {
	UsersModels, err := GetAllUsers()
	if err != nil {
		c.JSON(http.StatusInternalServerError, database.NewError("Users", err))
		return
	}
	serializer := NewUsersSerializer(c, UsersModels)
	response := serializer.Response()
	c.JSON(http.StatusOK, gin.H{"Users": response})
}

// additional functions added by colls_Codes at sep 6th at 16:33 for password reset using cradlevoices
func PasswordReset(c *gin.Context) {
	email := c.Param("email")
	// UserIDUint, err := strconv.ParseString(email, 10, 64)
	// if err != nil {
	// 	c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid email address .."})
	// 	return
	// }

	UserModel, err := GetUserByEmail(email)
	if err != nil {
		c.JSON(http.StatusInternalServerError, database.NewError("User", err))
		return
	}
	//generate random password something like a otp
	randomPswd := RandomOtpGenerator()
	fmt.Println(randomPswd)

	serializer := NewUserSerializer(c, UserModel)
	c.JSON(http.StatusOK, gin.H{"User": serializer.Response()})
}

// random password generator
func RandomOtpGenerator() (otp int) {
	// Generate a random integer between 100000 and 999999 (inclusive)
	otpd := rand.Intn(900000) + 100000
	return otpd
}
