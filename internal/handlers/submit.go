package handlers

import (
	"clipper/internal/models"
	"clipper/internal/utils"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
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

	// Log the request details
	clipDuration, _ := utils.ParseClipDuration(videoRequest.ClipStart, videoRequest.ClipEnd)
	clipDurationFormatted := utils.FormatSecondsToMMSS(clipDuration)
	slog.Info("Received download request",
		"ip", r.RemoteAddr,
		"url", videoRequest.VideoURL,
		"quality", videoRequest.Quality,
		"clip duration", clipDurationFormatted)

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

	// Store the file path, progress channel, and process in the shared maps
	data.addFileID(fileId, filePath)
	data.addProgressChannel(fileId, progressChannel)
	data.addProcess(fileId, cmd)

	select{
		// if the client disconnects or cancels the request, stop the process and clean up
		case <- r.Context().Done():
			slog.Warn("client disconnected, stopping the downloading process.", "ip", r.RemoteAddr)
			cmd.Process.Kill() // Stop the ffmpeg process
			data.cleanupAll(fileId) // Clean up all associated data
			return
		default:
			// Continue with the normal processing
	}

	// Respond with the process ID
	json.NewEncoder(w).Encode(response{Status: "started", ProcessId: fileId})

	// Start a goroutine to handle the rest of the download process
	// This will monitor the ffmpeg command and clean up resources when done.
	go func() {
		
		// If the download fails, send an error message on the channel and clean up.
		if err := cmd.Wait(); err != nil {

			slog.Error("ffmpeg process failed", "error", err, "processId", fileId)

			// Send a failure message on the channel before closing it.
			progressChannel <- models.ProgressResponse{Status: "error"}

			// close the progress channel and clean up
			close(progressChannel)
			data.cleanupAll(fileId)
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

		close(progressChannel)

		// wait for a while to ensure the client can download the file then clean up
		time.Sleep(10 * time.Second) 
		data.cleanupAll(fileId)
		slog.Info("cleanup completed for process", "processId", fileId)
	}()

	

}
