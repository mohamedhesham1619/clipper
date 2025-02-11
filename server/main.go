package main

import (
	"log/slog"
	"os"
	"net/http"
	
)


func main() {

	mux := http.NewServeMux()

	mux.HandleFunc("/submit", submitHandler)

	mux.HandleFunc("/", homeHandler)

	mux.HandleFunc("/download/", downloadHandler)

	// Create a new instance of the server
	server := http.Server{Handler: mux, Addr: ":8080"}

	// Start the server
	slog.Info("Server started on port 8080")
	if err := server.ListenAndServe(); err != nil {
		slog.Error("Server error", "error", err)
		os.Exit(1)
	}
}
