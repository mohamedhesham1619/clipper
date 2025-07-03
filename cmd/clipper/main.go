package main

import (
	"clipper/internal/server"
	"log/slog"
	"os"
)

func main() {

	// --- Logger Setup ---
	// Create a new handler with the minimum log level set to DEBUG.
	opts := &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}

	handler := slog.NewTextHandler(os.Stdout, opts)

	// Set this new logger as the default for the entire application.
	slog.SetDefault(slog.New(handler))

	// --- Server Initialization ---
	srv := server.New()

	if err := srv.Start(); err != nil {
		slog.Error("Server error", "error", err)
		os.Exit(1)
	}
}
