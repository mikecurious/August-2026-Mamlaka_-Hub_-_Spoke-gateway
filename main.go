package main

import (
	"com.mam-laka/database"
	"com.mam-laka/main/merchants"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	ginSwagger "github.com/swaggo/gin-swagger"

	// "github.com/swaggo/gin-swagger/swaggerFiles"
	"github.com/swaggo/files"

	"gorm.io/gorm"
)

// models migration
func Migration(database *gorm.DB) {

}

func main() {
	// Initialize the database connection
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
	url := ginSwagger.URL("http://localhost:8090/swagger/doc.json") // Update port if needed
	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler, url))
	//port 8080
	if err := router.Run(":8090"); err != nil {
		panic(err)
	}
}
