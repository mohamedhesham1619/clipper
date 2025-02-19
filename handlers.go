package main

import (
	"clipper/models"
	"clipper/utils"
	"embed"
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

	// Read the request videoRequest from the client
	var videoRequest models.VideoRequest

	connec.ReadJSON(&videoRequest)
	slog.Info("Received request", "data", videoRequest)

	// Download the video clip
	fileName, progressChannel, err := utils.DownloadVideo(videoRequest)

	if err != nil {
		connec.WriteJSON(models.ProgressResponse{Status: "error"})
		slog.Error("Error downloading video", "error", err, "request", videoRequest)
		return
	}

	// listen to progress channel and write the received message over the web socket
	for response := range progressChannel {
		connec.WriteJSON(response)
	}

	// Generate a unique ID for the file and store it
	fileId := utils.GenerateID()
	fileIDs[fileId] = fileName

	connec.WriteJSON(models.ProgressResponse{Status: "finished", Progress: 100, DownloadUrl: fmt.Sprintf("/download/%v", fileId)})

	slog.Info("process complete", "fileId", fileId, "fileName", fileName, "downloadUrl", fmt.Sprintf("/download/%v", fileId))

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

	// delete the file after 15 minutes
	go func() {
		time.Sleep(2 * time.Minute)
		delete(fileIDs, fileId)
		os.Remove(filePath)
	}()

}
