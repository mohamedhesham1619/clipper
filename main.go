package main

import (
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/exec"
)

// set the correct permissions for linux
func init() {
	err := exec.Command("chmod", "+x", "/workspace/yt-dlp").Run()
	if err != nil {
		log.Fatalf("Failed to set executable permission for yt-dlp: %v", err)
	}

	err = exec.Command("chmod", "+x", "/workspace/ffmpeg").Run()
	if err != nil {
		log.Fatalf("Failed to set executable permission for ffmpeg: %v", err)
	}
}

func main() {

	mux := http.NewServeMux()

	mux.HandleFunc("/", homeHandler)
	
	mux.HandleFunc("/submit", submitHandler)

	mux.HandleFunc("/download/", downloadHandler)

	server := http.Server{Handler: mux, Addr: ":8080"}

	// Start the server
	slog.Info("Server started on port 8080")
	if err := server.ListenAndServe(); err != nil {
		slog.Error("Server error", "error", err)
		os.Exit(1)
	}
}
