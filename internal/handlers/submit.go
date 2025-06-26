package handlers

import (
	"clipper/internal/models"
	"clipper/internal/utils"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"time"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

func SubmitHandler(w http.ResponseWriter, r *http.Request) {
	// Upgrade the HTTP connection to a WebSocket connection
	connec, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		slog.Error("Error upgrading connection", "error", err)
		return
	}
	// ensure the connection is closed when the function returns
	defer connec.Close()

	// Read the request videoRequest from the client
	var videoRequest models.VideoRequest
	connec.ReadJSON(&videoRequest)
	
	// Download the clip
	fileName, progressChannel, err := utils.DownloadVideo(videoRequest)
	if err != nil {
		connec.WriteJSON(models.ProgressResponse{Status: "error"})
		slog.Error("Error downloading clip", "error", err, "request", videoRequest)
		return
	}

	// listen to progress channel and write the received message over the web socket
	for response := range progressChannel {
		connec.WriteJSON(response)
	}

	// Generate a unique ID for the file and store it
	fileId := utils.GenerateID()
	FileIDs[fileId] = fileName

	connec.WriteJSON(models.ProgressResponse{
		Status:      "finished",
		Progress:    100,
		DownloadUrl: fmt.Sprintf("/download/%v", fileId),
	})

	// delete the file after a certain time
	go func() {
		time.Sleep(15 * time.Minute)
		os.Remove(fileName)
		delete(FileIDs, fileId)
	}()
}
