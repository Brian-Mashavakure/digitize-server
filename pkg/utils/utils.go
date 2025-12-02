package utils

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/signintech/gopdf"
)

func IsValidImageType(filename string) bool {
	ext := strings.ToLower(filepath.Ext(filename))
	validExtensions := map[string]bool{
		".jpg":  true,
		".jpeg": true,
		".png":  true,
		".gif":  true,
		".webp": true,
	}
	return validExtensions[ext]
}

func EncodeImageToBase64(imageData []byte) string {
	return base64.StdEncoding.EncodeToString(imageData)
}

func MarkdownToPDF(markdown string, fontPath string) ([]byte, error) {
	pdf := gopdf.GoPdf{}
	pdf.Start(gopdf.Config{PageSize: *gopdf.PageSizeA4})
	pdf.AddPage()

	err := pdf.AddTTFFont("times", fontPath)
	if err != nil {
		return nil, fmt.Errorf("failed to add font: %w", err)
	}

	err = pdf.SetFont("times", "", 14)
	if err != nil {
		return nil, fmt.Errorf("failed to set font: %w", err)
	}

	lines := strings.Split(markdown, "\n")
	yPos := 40.0
	marginX := 30.0
	lineHeight := 20.0
	pageHeight := 842.0
	bottomMargin := 50.0

	linkRegex := regexp.MustCompile(`\[([^\]]+)\]\(([^\)]+)\)`)
	headerRegex := regexp.MustCompile(`^(#{1,6})\s+(.+)$`)

	for _, line := range lines {
		line = strings.TrimSpace(line)

		if line == "" {
			yPos += lineHeight * 0.5
			continue
		}

		if yPos > pageHeight-bottomMargin {
			pdf.AddPage()
			yPos = 40.0
		}

		if headerRegex.MatchString(line) {
			matches := headerRegex.FindStringSubmatch(line)
			headerLevel := len(matches[1])
			headerText := matches[2]

			fontSize := float64(24 - headerLevel*2)
			if fontSize < 12 {
				fontSize = 12
			}

			err = pdf.SetFont("times", "B", int(fontSize))
			if err != nil {
				return nil, fmt.Errorf("failed to set font size: %w", err)
			}

			pdf.SetXY(marginX, yPos)
			pdf.Text(headerText)
			yPos += lineHeight * 1.5

			err = pdf.SetFont("times", "", 14)
			if err != nil {
				return nil, fmt.Errorf("failed to reset font: %w", err)
			}
			continue
		}

		if linkRegex.MatchString(line) {
			matches := linkRegex.FindAllStringSubmatchIndex(line, -1)
			currentX := marginX
			lastIndex := 0

			for _, match := range matches {
				linkStart := match[0]
				linkEnd := match[1]
				linkTextStart := match[2]
				linkTextEnd := match[3]
				linkURLStart := match[4]
				linkURLEnd := match[5]

				if linkStart > lastIndex {
					beforeLink := line[lastIndex:linkStart]
					if beforeLink != "" {
						pdf.SetXY(currentX, yPos)
						pdf.Text(beforeLink)
						width, err := pdf.MeasureTextWidth(beforeLink)
						if err != nil {
							return nil, fmt.Errorf("failed to measure text width: %w", err)
						}
						currentX += width
					}
				}

				linkText := line[linkTextStart:linkTextEnd]
				linkURL := line[linkURLStart:linkURLEnd]
				linkWidth, err := pdf.MeasureTextWidth(linkText)
				if err != nil {
					return nil, fmt.Errorf("failed to measure link text width: %w", err)
				}

				pdf.SetXY(currentX, yPos)
				pdf.Text(linkText)
				pdf.AddExternalLink(linkURL, currentX-2.5, yPos-12, linkWidth+5, 15)

				currentX += linkWidth
				lastIndex = linkEnd
			}

			if lastIndex < len(line) {
				remainingText := line[lastIndex:]
				pdf.SetXY(currentX, yPos)
				pdf.Text(remainingText)
			}

			yPos += lineHeight
			continue
		}

		text := line
		textWidth, err := pdf.MeasureTextWidth(text)
		if err != nil {
			return nil, fmt.Errorf("failed to measure text width: %w", err)
		}
		pageWidth := 595.0
		maxWidth := pageWidth - (marginX * 2)

		if textWidth > maxWidth {
			words := strings.Fields(text)
			currentLine := ""
			for _, word := range words {
				testLine := currentLine
				if testLine != "" {
					testLine += " "
				}
				testLine += word
				testWidth, err := pdf.MeasureTextWidth(testLine)
				if err != nil {
					return nil, fmt.Errorf("failed to measure test line width: %w", err)
				}
				if testWidth > maxWidth {
					if currentLine != "" {
						pdf.SetXY(marginX, yPos)
						pdf.Text(currentLine)
						yPos += lineHeight
						if yPos > pageHeight-bottomMargin {
							pdf.AddPage()
							yPos = 40.0
						}
					}
					currentLine = word
				} else {
					currentLine = testLine
				}
			}
			if currentLine != "" {
				pdf.SetXY(marginX, yPos)
				pdf.Text(currentLine)
				yPos += lineHeight
			}
		} else {
			pdf.SetXY(marginX, yPos)
			pdf.Text(text)
			yPos += lineHeight
		}
	}

	var buf bytes.Buffer
	err = pdf.Write(&buf)
	if err != nil {
		return nil, fmt.Errorf("failed to write PDF: %w", err)
	}

	return buf.Bytes(), nil
}
