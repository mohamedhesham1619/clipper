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
	mu.RLock()
	filePath, exists := fileIDs[fileId]
	mu.RUnlock()
	
	if !exists {
		http.Error(w, "File not found", http.StatusNotFound)
		return
	}

	// Get file info to set Content-Length and check for existence
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		slog.Error("Failed to get file info, file may have been cleaned up or never existed", "filePath", filePath, "error", err)
		http.Error(w, "File not found or unreadable", http.StatusNotFound)
		return
	}
	
	// Open the file
	file, err := os.Open(filePath)

	if err != nil {
		slog.Error("Error opening file", "filePath", filePath, "error", err)
		http.Error(w, "Error opening file", http.StatusInternalServerError)
		return
	}

	defer file.Close()

	// Set the content type
	ext := filepath.Ext(filePath)
	contentType := mime.TypeByExtension(ext)
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	// Set the headers
	// Add Content-Length for better client experience (e.g., download progress bar)
	w.Header().Set("Content-Length", fmt.Sprintf("%d", fileInfo.Size()))
	w.Header().Set("Content-Type", contentType)
	// Quote the filename to handle spaces and special characters correctly.
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", fileInfo.Name()))

	// Copy the file to the response writer
	io.Copy(w, file)

}
