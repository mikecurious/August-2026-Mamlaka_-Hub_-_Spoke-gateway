package database

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"
	"gorm.io/driver/mysql" // Use the MySQL driver
	"gorm.io/gorm"
)

var DB *gorm.DB

// Open the database and establish the connection
func Init() *gorm.DB {
	// Specify the connection properties for the MySQL database
	// DATABASE_DSN lets a second instance (e.g. the sandbox) point at another
	// database (impala_sandbox) without a code change. Falls back to the prod DSN
	// when unset so existing behavior is unchanged.
	dsn := os.Getenv("DATABASE_DSN")
	if dsn == "" {
		dsn = "colls:djsjsjsoewe88wSSDDF.Sf*@tcp(localhost:3306)/impala_gateway?charset=utf8mb4&parseTime=True&loc=Local" //prod
	}
	//
	// Open the database connection.
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatal("Failed to connect to the database:", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		log.Fatal("Failed to configure database pool:", err)
	}
	sqlDB.SetMaxOpenConns(100)
	sqlDB.SetMaxIdleConns(25)
	sqlDB.SetConnMaxLifetime(30 * time.Minute)
	sqlDB.SetConnMaxIdleTime(5 * time.Minute)

	DB = db
	return DB
}

// Function to get the database connection
func GetConnection() *gorm.DB {
	return DB
}

// Binder
func Bind(c *gin.Context, object interface{}) error {
	binder := binding.Default(c.Request.Method, c.ContentType())
	return c.ShouldBindWith(object, binder)
}

// Return customized error info
type CommonError struct {
	Errors map[string]interface{} `json:"errors"`
}

// Validators

func NewValidatorError(err error) CommonError {
	res := CommonError{}
	res.Errors = make(map[string]interface{})
	errs := err.(validator.ValidationErrors)
	for _, v := range errs {
		res1 := fmt.Sprintf("{%v: %v}", v.Tag(), v.Param())
		fmt.Println(res1)
		if v.Param() != "" {

			res.Errors[v.Field()] = fmt.Sprintf("{%v: %v}", v.Tag(), v.Param())
		} else {
			res.Errors[v.Field()] = fmt.Sprintf("{key: %v}", v.Tag())
		}
	}
	return res
}

// Wrapping error into an object
func NewError(key string, err error) CommonError {
	res := CommonError{}
	res.Errors = make(map[string]interface{})
	res.Errors[key] = err.Error()
	return res
}
