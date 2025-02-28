package main

import (
	"log/slog"
	"net/http"
	"os"
)

func main() {

	mux := http.NewServeMux()

	mux.HandleFunc("/", homeHandler)
	
	mux.HandleFunc("/submit", submitHandler)

	mux.HandleFunc("/download/", downloadHandler)

	mux.HandleFunc("/progress/", progressHandler)

	server := http.Server{Handler: mux, Addr: ":8080"}

	// Start the server
	slog.Info("Server started on port 8080 in docker container")
	if err := server.ListenAndServe(); err != nil {
		slog.Error("Server error", "error", err)
		os.Exit(1)
	}
}
