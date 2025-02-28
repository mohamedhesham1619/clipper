package main

import (
	"clipper/models"
	"clipper/utils"
	"embed"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// store the file IDs and their corresponding file names
var fileIDs = make(map[string]string)

// store the progress channels for each download process
var progressTracker = make(map[string]chan models.ProgressResponse)

func submitHandler(w http.ResponseWriter, r *http.Request) {

	// Read the request from the client
	var videoRequest models.VideoRequest
	json.NewDecoder(r.Body).Decode(&videoRequest)

	slog.Info("Received request", "data", videoRequest)

	// Start the download process
	fileName, progressChannel, err := utils.DownloadVideo(videoRequest)

	type response struct {
		Status    string `json:"status"`
		ProcessId string `json:"processId"`
	}

	if err != nil {

		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(response{Status: "error", ProcessId: ""})
		slog.Error("Error downloading video", "error", err, "request", videoRequest)
		return
	}

	// Generate a unique ID for the file and store it
	fileId := utils.GenerateID()
	fileIDs[fileId] = fileName
	progressTracker[fileId] = progressChannel

	json.NewEncoder(w).Encode(response{Status: "started", ProcessId: fileId})

	slog.Info("process started", "fileId", fileId, "fileName", fileName)

}

func progressHandler(w http.ResponseWriter, r *http.Request) {

	// Get the process ID from the URL
	processId := strings.TrimPrefix(r.URL.Path, "/progress/")

	// Get the progress channel
	progressChannel, exists := progressTracker[processId]

	if !exists {
		http.Error(w, "Process not found", http.StatusNotFound)
		slog.Error("Process not found", "processId", processId)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	for progress := range progressChannel {
		// Format as proper SSE message
		fmt.Fprintf(w, "data: %v\n\n", string(must(json.Marshal(progress))))
		w.(http.Flusher).Flush()
	}
	// Send final message
	fmt.Fprintf(w, "data: %v\n\n", string(must(json.Marshal(models.ProgressResponse{
		Status:      "finished",
		Progress:    100,
		DownloadUrl: fmt.Sprintf("/download/%v", processId),
	}))))
}

// Helper function for json.Marshal
func must(data []byte, err error) []byte {
	if err != nil {
		return []byte("{}")
	}
	return data
}

//go:embed client/web/page.html
var content embed.FS

func homeHandler(w http.ResponseWriter, r *http.Request) {
	data, err := content.ReadFile("client/web/page.html")
	if err != nil {
		http.Error(w, "Error reading file", http.StatusInternalServerError)
		return
	}
	w.Write(data)
}

func downloadHandler(w http.ResponseWriter, r *http.Request) {

	// Get the file ID from the URL
	fileId := strings.TrimPrefix(r.URL.Path, "/download/")
	slog.Info("received download request", "fileId", fileId)

	// Get the file name from the map if it exists
	fileName, exists := fileIDs[fileId]

	if !exists {

		http.Error(w, "File not found", http.StatusNotFound)
		return
	}

	filePath, _ := filepath.Abs(fmt.Sprintf("temp/%v", fileName))
	// Open the file
	file, err := os.Open(filePath)

	if err != nil {
		slog.Error(fmt.Sprintf("error opening %v", fileName), "error", err)
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

	// delete the file after a certain time
	go func() {
		time.Sleep(10 * time.Minute)
		delete(fileIDs, fileId)
		os.Remove(filePath)
	}()

}
