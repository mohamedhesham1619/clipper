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

	// Generate a unique ID for the file
	fileId := utils.GenerateID()
	
	// Store the file path and progress channel in the shared maps
	data.addFileID(fileId, filePath)
	data.addProgressChannel(fileId, progressChannel)

	// Start a goroutine to handle the ffmpeg process without blocking the main handler.
	go func() {
		// Close the progress channel and remove it from the tracker when done.
		defer func() {
			close(progressChannel)
			data.removeProgressChannel(fileId)
		}()

		// If the download fails, send an error message on the channel and clean up.
		if err := cmd.Wait(); err != nil {

			slog.Error("ffmpeg process failed", "error", err, "processId", fileId)

			// Send a failure message on the channel before closing it.
			progressChannel <- models.ProgressResponse{Status: "error"}

			// The file ID is removed immediately on failure.
			data.removeFileID(fileId)
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

		// Schedule cleanup of the file after certain time
		time.AfterFunc(10*time.Minute, func() {
			slog.Info("Cleaning up old file", "processId", fileId, "path", filePath)
			data.removeFileID(fileId)
			os.Remove(filePath)
		})
	}()

	// Respond with the process ID
	json.NewEncoder(w).Encode(response{Status: "started", ProcessId: fileId})

}
