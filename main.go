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
	copyBinary("/workspace/yt-dlp", "/tmp/yt-dlp")
	copyBinary("/workspace/ffmpeg", "/tmp/ffmpeg")

	err := exec.Command("chmod", "+x", "/tmp/yt-dlp").Run()
	if err != nil {
		log.Fatalf("Failed to set executable permission for yt-dlp: %v", err)
	}

	err = exec.Command("chmod", "+x", "/tmp/ffmpeg").Run()
	if err != nil {
		log.Fatalf("Failed to set executable permission for ffmpeg: %v", err)
	}
}

// Helper function to copy binaries to /tmp/
func copyBinary(src, dest string) {
	input, err := os.ReadFile(src)
	if err != nil {
		log.Fatalf("Failed to read binary %s: %v", src, err)
	}

	err = os.WriteFile(dest, input, 0755)
	if err != nil {
		log.Fatalf("Failed to write binary %s: %v", dest, err)
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
