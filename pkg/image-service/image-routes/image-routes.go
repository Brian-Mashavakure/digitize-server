package image_routes

import (
	"github.com/Brian-Mashavakure/digitize-server/pkg/image-service/image-handlers"
	"github.com/gin-gonic/gin"
)

func SetupImageRoutes(router *gin.Engine) {
	api := router.Group("digitize-api/images")

	//routes
	api.POST("/process-image", image_handlers.ProcessSingleImageHandler)
	api.POST("/process-multiple-images", image_handlers.ProcessMultipleImagesHandler)
}
