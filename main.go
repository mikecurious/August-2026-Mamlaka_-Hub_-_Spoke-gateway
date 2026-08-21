package main

import (
	"fmt"
	"os"

	"com.mam-laka/balances"
	"com.mam-laka/database"
	"com.mam-laka/forex"
	"com.mam-laka/main/merchants"
	"com.mam-laka/main/settlement"
	"com.mam-laka/transactions"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"

	"gorm.io/gorm"
)

// models migration
func Migration(database *gorm.DB) {
	balances.AutoMigrate()
	forex.AutoMigrate()
	settlement.AutoMigrate()
	transactions.AutoMigrate()
}

func main() {
	//load the env
	err := godotenv.Load()
	if err != nil {
		fmt.Println("Warning: .env file not found, using system env...")
	}

	// Initialize the database connection
	// Set Gin to release mode for production
	gin.SetMode(gin.DebugMode) // Change to gin.ReleaseMode for production
	database := database.Init()
	Migration(database)

	router := gin.Default()

	router.Use(cors.New(cors.Config{
		AllowOrigins: []string{"*"},
		AllowMethods: []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders: []string{"*"},
	}))

	sun := router.Group("/api")
	// users.Create(sun.Group("/users"))
	merchants.RegisterRoutes(sun.Group("/v1"))
	settlement.RegisterRoutes(sun.Group("/v1"))
	merchants.StartPesalinkPayoutStatusCron()
	merchants.StartAirtelReconciler()
	forex.RegisterRoutes(sun.Group("/v1"))
	// orders.Create(sun.Group("/orders"))
	// Add Swagger UI

	// LISTEN_ADDR lets a second instance (e.g. the sandbox) bind a different
	// port; prod keeps the historical default when the var is unset.
	addr := os.Getenv("LISTEN_ADDR")
	if addr == "" {
		addr = "127.0.0.1:8090"
	}
	if err := router.Run(addr); err != nil {
		panic(err)
	}
}
