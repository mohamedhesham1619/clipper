package handlers

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
)

// ProgressHandler streams progress updates to the client using SSE
func ProgressHandler(w http.ResponseWriter, r *http.Request) {
	// Get the process ID from the URL
	processId := strings.TrimPrefix(r.URL.Path, "/progress/")

	// Get the progress channel
	progressChannel, exists := data.getProgressChannel(processId)

	if !exists {
		http.Error(w, "Process not found", http.StatusNotFound)
		return
	}

	// Set headers for SSE
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	// Stop the ffmpeg process if the client disconnects
	go func() {
		<-r.Context().Done()
		if cmd, ok := data.getProcess(processId); ok && cmd.Process != nil {
			// If the process is still running, kill it and cleanup the resources
			if cmd.ProcessState == nil || !cmd.ProcessState.Exited() {
				slog.Warn("Killing ffmpeg process due to client disconnect", "ip", r.RemoteAddr, "processId", processId)
				_ = cmd.Process.Kill()
				data.cleanupAll(processId)
			} 
		}
	}()

	// Stream progress updates
	for progress := range progressChannel {
		b, _ := json.Marshal(progress)
		fmt.Fprintf(w, "data: %s\n\n", b)
		w.(http.Flusher).Flush()
	}

}
