package main

import (
	"log"

	"github.com/Brian-Mashavakure/digitize-server/pkg/image-service/image-routes"
	"github.com/gin-gonic/gin"
)

func main() {
	router := gin.Default()

	image_routes.SetupImageRoutes(router)

	log.Println("Server starting on port 8080...")
	if err := router.Run(":8080"); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
