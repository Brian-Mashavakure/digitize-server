package mistral_handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/Brian-Mashavakure/digitize-server/pkg/utils"
	"github.com/joho/godotenv"
	"io"
	"net/http"
	"os"
)

func OCRImageHandler(imageData []byte) (string, error) {
	err := godotenv.Load()
	if err != nil {
		fmt.Println("Error loading .env file")
	}

	apiKey := os.Getenv("MISTRAL_API_KEY")
	if apiKey == "" {
		return "", fmt.Errorf("MISTRAL_API_KEY environment variable is not set")
	}

	encodedImage := utils.EncodeImageToBase64(imageData)

	payload := map[string]interface{}{
		"model": "mistral-ocr-2505",
		"document": map[string]string{
			"type":      "image_url",
			"image_url": fmt.Sprintf("data:image/jpeg;base64,%s", encodedImage),
		},
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request payload: %w", err)
	}

	req, err := http.NewRequest("POST", "https://api.mistral.ai/v1/ocr", bytes.NewBuffer(payloadBytes))
	if err != nil {
		return "", fmt.Errorf("failed to create HTTP request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", apiKey))

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to send request to Mistral API: %w", err)
	}
	defer resp.Body.Close()

	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("Mistral API returned error status %d: %s", resp.StatusCode, string(responseBody))
	}

	var apiResponse struct {
		Pages []struct {
			Index      int                    `json:"index"`
			Markdown   string                 `json:"markdown"`
			Images     []interface{}          `json:"images"`
			Dimensions map[string]interface{} `json:"dimensions"`
		} `json:"pages"`
		Model     string `json:"model"`
		UsageInfo struct {
			PagesProcessed int  `json:"pages_processed"`
			DocSizeBytes   *int `json:"doc_size_bytes"`
		} `json:"usage_info"`
		Error *struct {
			Message string `json:"message"`
			Type    string `json:"type"`
		} `json:"error"`
	}

	if err := json.Unmarshal(responseBody, &apiResponse); err != nil {
		return "", fmt.Errorf("failed to parse API response: %w", err)
	}

	if apiResponse.Error != nil {
		return "", fmt.Errorf("Mistral API error: %s", apiResponse.Error.Message)
	}

	if len(apiResponse.Pages) == 0 {
		return "", fmt.Errorf("no pages returned from Mistral API")
	}

	var combinedMarkdown string
	for i, page := range apiResponse.Pages {
		if i > 0 {
			combinedMarkdown += "\n\n"
		}
		combinedMarkdown += page.Markdown
	}

	if combinedMarkdown == "" {
		return "", fmt.Errorf("no markdown content found in response pages")
	}

	return combinedMarkdown, nil
}
