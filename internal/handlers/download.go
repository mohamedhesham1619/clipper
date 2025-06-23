package handlers

import (
	"fmt"
	"io"
	"log/slog"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// DownloadHandler serves the requested file for download
func DownloadHandler(w http.ResponseWriter, r *http.Request) {
	// Get the file ID from the URL
	fileId := strings.TrimPrefix(r.URL.Path, "/download/")
	
	// Get the file name from the map if it exists
	fileName, exists := fileIDs[fileId]

	if !exists {
		http.Error(w, "File not found", http.StatusNotFound)
		return
	}

	filePath, _ := filepath.Abs(filepath.Join("temp", fileName))
	
	// Open the file
	file, err := os.Open(filePath)

	if err != nil {
		slog.Error("Error opening file", "filePath", filePath, "error", err)
		http.Error(w, "Error opening file", http.StatusInternalServerError)
		return
	}

	defer file.Close()

	// Set the content type
	ext := filepath.Ext(fileName)
	contentType := mime.TypeByExtension(ext)
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	// Set the headers
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s", fileName))

	// Copy the file to the response writer
	io.Copy(w, file)

}
