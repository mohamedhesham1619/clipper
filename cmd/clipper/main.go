package main

import (
	"clipper/internal/server"
	"log/slog"
	"os"
)

func main() {
	srv := server.New()

	if err := srv.Start(); err != nil {
		slog.Error("Server error", "error", err)
		os.Exit(1)
	}
}
