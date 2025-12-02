package image_handlers

import (
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/Brian-Mashavakure/digitize-server/pkg/mistral-service/mistral-handlers"
	"github.com/Brian-Mashavakure/digitize-server/pkg/utils"
	"github.com/gin-gonic/gin"
)

type ProcessImageRequest struct {
	Image *multipart.FileHeader `form:"image" binding:"required"`
}

type ProcessImageResponse struct {
	Message  string `json:"message"`
	Success  bool   `json:"success"`
	Markdown string `json:"markdown,omitempty"`
}

type ImageInfo struct {
	Filename    string `json:"filename"`
	Size        int64  `json:"size"`
	ContentType string `json:"content_type"`
}

type ProcessMultipleImagesResponse struct {
	Message         string      `json:"message"`
	Success         bool        `json:"success"`
	ImagesCount     int         `json:"images_count"`
	ProcessedImages []ImageInfo `json:"processed_images"`
}

func ProcessSingleImageHandler(c *gin.Context) {
	// Parse the multipart form with a max memory of 10 MB
	if err := c.Request.ParseMultipartForm(10 << 20); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Failed to parse form data",
			"details": err.Error(),
		})
		return
	}

	// Get the image file from the form
	file, header, err := c.Request.FormFile("image")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "No image file provided",
			"details": err.Error(),
		})
		return
	}
	defer file.Close()

	// Validate file type
	if !utils.IsValidImageType(header.Filename) {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid file type. Only JPEG, PNG, GIF, and WebP images are allowed",
		})
		return
	}

	// Validate file size (max 10 MB)
	if header.Size > 10<<20 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "File size too large. Maximum size is 10 MB",
		})
		return
	}

	// Read the image data
	imageData, err := io.ReadAll(file)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to read image data",
			"details": err.Error(),
		})
		return
	}

	// Validate that we actually have image data
	contentType := http.DetectContentType(imageData)
	if !strings.HasPrefix(contentType, "image/") {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("File is not a valid image. Detected type: %s", contentType)})
		return
	}

	markdown, err := mistral_handlers.OCRImageHandler(imageData)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to process image with OCR",
			"details": err.Error(),
		})
		return
	}

	fontPath := "pkg/fonts/times.ttf"
	pdfBytes, err := utils.MarkdownToPDF(markdown, fontPath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to generate PDF",
			"details": err.Error(),
		})
		return
	}

	c.Header("Content-Type", "application/pdf")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s.pdf\"", strings.TrimSuffix(header.Filename, filepath.Ext(header.Filename))))
	c.Data(http.StatusOK, "application/pdf", pdfBytes)
}

func ProcessMultipleImagesHandler(c *gin.Context) {
	// Parse the multipart form with a max memory of 50 MB
	if err := c.Request.ParseMultipartForm(50 << 20); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Failed to parse form data",
			"details": err.Error(),
		})
		return
	}

	// Get the multipart form
	form := c.Request.MultipartForm
	if form == nil || form.File["images"] == nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "No images provided in the form",
		})
		return
	}

	// Get all files from the "images" field
	files := form.File["images"]
	if len(files) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "No images provided",
		})
		return
	}

	// Limit the number of images (max 10)
	if len(files) > 10 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Too many images. Maximum is 10 images per request",
		})
		return
	}

	var markdowns []string

	// Process each image
	for i, fileHeader := range files {
		// Validate file type
		if !utils.IsValidImageType(fileHeader.Filename) {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": fmt.Sprintf("Invalid file type for image %d (%s). Only JPEG, PNG, GIF, and WebP images are allowed", i+1, fileHeader.Filename),
			})
			return
		}

		// Validate file size (max 10 MB per file)
		if fileHeader.Size > 10<<20 {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": fmt.Sprintf("File size too large for image %d (%s). Maximum size is 10 MB per image", i+1, fileHeader.Filename),
			})
			return
		}

		// Open the file
		file, err := fileHeader.Open()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":   fmt.Sprintf("Failed to open image %d (%s)", i+1, fileHeader.Filename),
				"details": err.Error(),
			})
			return
		}

		// Read the image data
		imageData, err := io.ReadAll(file)
		file.Close()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":   fmt.Sprintf("Failed to read image %d (%s)", i+1, fileHeader.Filename),
				"details": err.Error(),
			})
			return
		}

		// Validate that we actually have image data
		contentType := http.DetectContentType(imageData)
		if !strings.HasPrefix(contentType, "image/") {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": fmt.Sprintf("File %d (%s) is not a valid image. Detected type: %s", i+1, fileHeader.Filename, contentType),
			})
			return
		}

		// Process image with OCR
		markdown, err := mistral_handlers.OCRImageHandler(imageData)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":   fmt.Sprintf("Failed to process image %d (%s) with OCR", i+1, fileHeader.Filename),
				"details": err.Error(),
			})
			return
		}

		markdowns = append(markdowns, markdown)
	}

	// Generate PDF from all markdowns
	fontPath := "pkg/fonts/times.ttf"
	pdfBytes, err := utils.MultipleMarkdownsToPDF(markdowns, fontPath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to generate PDF",
			"details": err.Error(),
		})
		return
	}

	c.Header("Content-Type", "application/pdf")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=\"multiple_images.pdf\""))
	c.Data(http.StatusOK, "application/pdf", pdfBytes)
}
