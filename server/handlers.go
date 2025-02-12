package main

import (
	"clipper/models"
	"clipper/utils"
	"fmt"
	"io"
	"log/slog"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gorilla/websocket"
	//"clipper/server/utils"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

// store the file IDs and their corresponding file names
var fileIDs = make(map[string]string)

func submitHandler(w http.ResponseWriter, r *http.Request) {

	// Upgrade the HTTP connection to a WebSocket connection
	connec, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		slog.Error("Error upgrading connection", "error", err)
		return
	}

	// ensure the connection is closed when the function returns
	defer connec.Close()

	// Read the request data from the client
	var data struct {
		VideoURL     string `json:"videoUrl"`
		ClipDuration string `json:"clipDuration"`
	}

	connec.ReadJSON(&data)
	slog.Info("Received request", "data", data)

	// Download the video clip
	fileName, progressChannel, err := utils.DownloadVideo(data.VideoURL, data.ClipDuration)

	if err != nil {
		connec.WriteJSON(models.ProgressResponse{Status: "error"})
		slog.Error("Error downloading video", "error", err, "request", data)
		return
	}

	for response := range progressChannel {
		connec.WriteJSON(response)
	}

	// Generate a unique ID for the file and store it
	fileId := utils.GenerateID()
	fileIDs[fileId] = fileName

	connec.WriteJSON(models.ProgressResponse{Status: "finished",Progress: 100, DownloadUrl: fmt.Sprintf("/download/%v", fileId)})

	slog.Info("process complete", "fileId", fileId, "fileName", fileName, "downloadUrl", fmt.Sprintf("/download/%v", fileId))

}

func homeHandler(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "../client/web/page.html")
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

	filePath, _ := filepath.Abs(fmt.Sprintf("../assets/videos/%v", fileName))
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

	// delete the file after 10 minutes
	go func() {
		time.Sleep(1 * time.Minute)
		delete(fileIDs, fileId)
		os.Remove(filePath)
	}()

}
