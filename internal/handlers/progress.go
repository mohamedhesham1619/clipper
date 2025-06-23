package handlers

import (
	"clipper/internal/models"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

// ProgressHandler streams progress updates to the client using SSE
func ProgressHandler(w http.ResponseWriter, r *http.Request) {
	// Get the process ID from the URL
	processId := strings.TrimPrefix(r.URL.Path, "/progress/")

	// Get the progress channel
	progressChannel, exists := progressTracker[processId]

	if !exists {
		http.Error(w, "Process not found", http.StatusNotFound)
		return
	}

	// Set headers for SSE
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	// Stream progress updates
	for progress := range progressChannel {
		b, _ := json.Marshal(progress)
		fmt.Fprintf(w, "data: %s\n\n", b)
		w.(http.Flusher).Flush()
	}

	// Send final message
	final := models.ProgressResponse{
		Status:      "finished",
		Progress:    100,
		DownloadUrl: fmt.Sprintf("/download/%v", processId),
	}
	b, _ := json.Marshal(final)
	fmt.Fprintf(w, "data: %s\n\n", b)
	w.(http.Flusher).Flush()
}
