package handlers

import (
	"clipper/internal/models"
	"clipper/internal/utils"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
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
	filePath, progressChannel, cmd, err := utils.DownloadVideo(videoRequest)

	// If there was an error during the download, return an error response
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(response{Status: "error", ProcessId: ""})
		slog.Error("Error downloading video", "error", err, "request", videoRequest)
		return
	}

	// Generate a unique ID for the file and store it
	fileId := utils.GenerateID()
	
	mu.Lock()
	fileIDs[fileId] = filePath
	progressTracker[fileId] = progressChannel
	jobStatus[fileId] = "in_progress"
	mu.Unlock()

	// In a goroutine, wait for the command to finish and update the job status.
	go func() {
		// Ensure the progress channel is always closed and removed from the tracker when this goroutine exits.
		defer func() {
			close(progressChannel)
			mu.Lock()
			delete(progressTracker, fileId)
			mu.Unlock()
		}()

		// cmd.Wait() blocks until the ffmpeg process is finished.
		if err := cmd.Wait(); err != nil {
			// This block runs if ffmpeg fails.
			slog.Error("ffmpeg process failed", "error", err, "processId", fileId)

			// Send a failure message on the channel before closing it.
			progressChannel <- models.ProgressResponse{Status: "error"}

			mu.Lock()
			jobStatus[fileId] = "failed"
			delete(fileIDs, fileId) // Remove so it can't be downloaded
			mu.Unlock()
			os.Remove(filePath) // remove potentially partial file
			return
		}

		// This block runs only if ffmpeg succeeds.
		slog.Info("ffmpeg process finished successfully", "processId", fileId)

		// Send the final success message on the channel before closing it.
		progressChannel <- models.ProgressResponse{
			Status:      "finished",
			Progress:    100,
			DownloadUrl: fmt.Sprintf("/download/%v", fileId),
		}

		mu.Lock()
		jobStatus[fileId] = "completed"
		mu.Unlock()

		time.AfterFunc(10*time.Minute, func() {
			slog.Info("Cleaning up old file and status", "processId", fileId, "path", filePath)
			mu.Lock()
			delete(fileIDs, fileId)
			delete(jobStatus, fileId)
			mu.Unlock()
			os.Remove(filePath)
		})
	}()

	// Respond with the process ID
	json.NewEncoder(w).Encode(response{Status: "started", ProcessId: fileId})

}
