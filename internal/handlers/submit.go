package handlers

import (
	"clipper/internal/models"
	"clipper/internal/utils"
	"encoding/json"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

type response struct {
	Status    string `json:"status"`
	ProcessId string `json:"processId"`
}

// SubmitHandler handles the submission of a new video download request
func SubmitHandler(w http.ResponseWriter, r *http.Request) {
	// Read the request from the client
	var videoRequest models.VideoRequest
	json.NewDecoder(r.Body).Decode(&videoRequest)

	// Start the download process
	fileName, progressChannel, err := utils.DownloadVideo(videoRequest)

	// If there was an error during the download, return an error response
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(response{Status: "error", ProcessId: ""})
		slog.Error("Error downloading video", "error", err, "request", videoRequest)
		return
	}

	// Generate a unique ID for the file and store it
	fileId := utils.GenerateID()
	fileIDs[fileId] = fileName

	// Store the progress channel for this file ID
	progressTracker[fileId] = progressChannel

	// delete the file after a certain time
	go func() {
		time.Sleep(10 * time.Minute)
		delete(fileIDs, fileId)
		os.Remove(filepath.Join("temp", fileName))
	}()

	// Respond with the process ID
	json.NewEncoder(w).Encode(response{Status: "started", ProcessId: fileId})

}
