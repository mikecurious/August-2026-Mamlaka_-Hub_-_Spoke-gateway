package main

import (
	"com.mam-laka/balances"
	"com.mam-laka/database"
	"com.mam-laka/main/merchants"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"

	"gorm.io/gorm"
)

// models migration
func Migration(database *gorm.DB) {
	balances.AutoMigrate()

}

func main() {
	// Initialize the database connection
	// Set Gin to release mode for production
	gin.SetMode(gin.DebugMode) // Change to gin.ReleaseMode for production
	database := database.Init()
	Migration(database)

	router := gin.Default()

	// CORS configuration :updated on august 28 17:16 pm
	config := cors.DefaultConfig()
	config.AllowAllOrigins = true
	router.Use(cors.New(config))

	sun := router.Group("/api")
	// users.Create(sun.Group("/users"))
	merchants.RegisterRoutes(sun.Group("/v1"))
	// orders.Create(sun.Group("/orders"))
	// Add Swagger UI

	if err := router.Run(":8090"); err != nil {
		panic(err)
	}
}
