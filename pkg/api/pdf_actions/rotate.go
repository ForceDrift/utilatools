package api

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/pdfcpu/pdfcpu/pkg/api"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
)

// Simple logging function
func logWritingTo(path string) {
	fmt.Printf("Writing PDF to: %s\n", path)
}

func RotationRoutes(router *gin.Engine) {
	router.POST("/api/rotate", handlePDFRotation)
}

func savePDF(c *gin.Context) (string, error) {
	file, header, err := c.Request.FormFile("pdf")
	if err != nil {
		return "", err
	}
	defer file.Close()

	tempDir := "C:\\Users\\Roshan\\UtilaTools\\output"

	filePath := filepath.Join(tempDir, "uploaded_"+header.Filename)
	dst, err := os.Create(filePath)
	if err != nil {
		return "", err
	}
	defer dst.Close()

	_, err = io.Copy(dst, file)
	if err != nil {
		return "", err
	}

	return filePath, nil
}

func rotatePDF(inputPath string, outputPath string, rotationAngle int) error {
	conf := model.NewDefaultConfiguration()

	selectedPages := []string{}

	return api.RotateFile(inputPath, outputPath, rotationAngle, selectedPages, conf)
}

func handlePDFRotation(c *gin.Context) {

	rotationStr := c.PostForm("rotation")
	rotationAngle, err := strconv.Atoi(rotationStr)
	if err != nil || (rotationAngle != 90 && rotationAngle != 180 && rotationAngle != 270) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid rotation angle"})
		return
	}

	filePath, err := savePDF(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to upload file: " + err.Error()})
		return
	}

	outputPath := strings.Replace(filePath, "uploaded_", "rotated_", 1)

	err = rotatePDF(filePath, outputPath, rotationAngle)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to rotate PDF: " + err.Error()})
		return
	}

	// return the rotated PDF as a download
	c.FileAttachment(outputPath, "rotated_"+filepath.Base(filePath)[9:])
}
