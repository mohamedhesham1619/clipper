package main

import (
	"fmt"
	"io"
	"mime"
	"path/filepath"

	"log/slog"
	"os"
	"time"

	"net/http"

	"os/exec"
	"strings"
	"unicode"

	"math/rand/v2"

	"github.com/gorilla/websocket"
)

type RequestData struct {
	VideoURL     string `json:"videoUrl"`
	ClipDuration string `json:"clipDuration"`
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

// store the file IDs and their corresponding file names
var fileIDs = make(map[string]string)

func main() {

	mux := http.NewServeMux()

	// todo: generate a unique ID for each file
	mux.HandleFunc("/submit", func(w http.ResponseWriter, r *http.Request) {

		// Upgrade the HTTP connection to a WebSocket connection
		connec, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			slog.Error("Error upgrading connection", "error", err)
			return
		}

		// ensure the connection is closed when the function returns
		defer connec.Close()

		// Read the request data from the client
		var data RequestData
		connec.ReadJSON(&data)
		slog.Info("Received request", "data", data)

		// Download the video clip
		fileName, err := downloadVideo(data)
		if err != nil {
			connec.WriteJSON(map[string]string{"status": "error", "message": err.Error()})
			slog.Error("Error downloading video", "error", err, "request", data)
			return
		}

		// Generate a unique ID for the file and store it
		fileId := generateID()
		fileIDs[fileId] = fileName

		// Simulate processing with progress updates
		for i := 1; i <= 5; i++ {
			time.Sleep(1 * time.Second)
			progressMsg := fmt.Sprintf("Progress %d%%", i*20)
			connec.WriteMessage(websocket.TextMessage, []byte(progressMsg))
		}

		connec.WriteJSON(map[string]string{"status": "done", "downloadUrl": fmt.Sprintf("/download/%v", fileId)})
		slog.Info("process complete", "fileId", fileId, "fileName", fileName, "downloadUrl", fmt.Sprintf("/download/%v", fileId))

	})

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "page.html")
	})

	// todo: download the file with the given ID
	mux.HandleFunc("/download/", func(w http.ResponseWriter, r *http.Request) {

		// Get the file ID from the URL
		fileId := strings.TrimPrefix(r.URL.Path, "/download/")
		slog.Info("received download request", "fileId", fileId)
		
		// Get the file name from the map if it exists
		fileName, exists := fileIDs[fileId]

		if !exists {
			
			http.Error(w, "File not found", http.StatusNotFound)
			return
		}

		// Open the file
		file, err := os.Open(fileName)

		if err != nil {
			slog.Error(fmt.Sprintf("error opening %v", fileName), "error", err)
			http.Error(w, "Error opening file", http.StatusInternalServerError)
			return
		}

		defer file.Close()

		// Set the content type
		ext := filepath.Ext(fileName)
		contentType := mime.TypeByExtension(ext)
		if contentType == "" {
			contentType = "application/octet-stream"
		}

		// Set the headers
		w.Header().Set("Content-Type", contentType)
		w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s", fileName))

		// Copy the file to the response writer
		io.Copy(w, file)
		
		// delete the file after 10 minutes
		go func() {
			time.Sleep(10 * time.Minute)
			delete(fileIDs, fileId)
			os.Remove(fileName)
		}()
		

	})

	// Create a new instance of the server
	server := http.Server{Handler: mux, Addr: ":8080"}

	// Start the server
	slog.Info("Server started on port 8080")
	server.ListenAndServe()
}

func generateID() string {

	randNum := rand.Int32N(10000)
	return fmt.Sprintf("%d%d", time.Now().UnixNano(), randNum)
}

// sanitize the filename to remove or replace characters that are problematic in filenames
func sanitizeFilename(filename string) string {

	replacements := map[rune]rune{
		'/':  '-',
		'\\': '-',
		':':  '-',
		'*':  '-',
		'?':  '-',
		'"':  '-',
		'<':  '-',
		'>':  '-',
		'|':  '-',
	}

	sanitized := []rune{}
	for _, r := range filename {
		if replaced, exists := replacements[r]; exists {
			sanitized = append(sanitized, replaced)
		} else if unicode.IsPrint(r) {
			sanitized = append(sanitized, r)
		}
	}

	return string(sanitized)
}

// prepare the command to download the clip of the video
func buildClipDownloadCommand(req RequestData) (*exec.Cmd, string, error) {

	// Get both the URL and the title with the extension
	cmd := exec.Command("./yt-dlp",
		"-f", "b",
		"--print", "%(title)s.%(ext)s\n%(url)s",
		"--encoding", "UTF-8",
		"--no-download",
		req.VideoURL,
	)

	output, err := cmd.Output()

	if err != nil {

		return nil, "", fmt.Errorf("error getting video info: %v, command: %v, output: %v", err, cmd.String(), string(output))
	}

	// Split output into lines
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")

	if len(lines) < 2 {

		return nil, "", fmt.Errorf("expected both URL and title but got: %v", lines)
	}

	videoTitle := sanitizeFilename(lines[0])
	videoURL := lines[1]

	slog.Info("video title sanitization", "before:", lines[0], "after:", videoTitle)

	// download the clip to the current directory with the title as the file name
	downloadPath := videoTitle

	clipDuration := strings.Split(req.ClipDuration, "-")
	clipStart := clipDuration[0]
	clipEnd := clipDuration[1]

	ffmpegCmd := exec.Command(
		"./ffmpeg", "-i", videoURL,
		"-ss", clipStart, // Set the clip start and end time
		"-to", clipEnd,
		// "-progress",
		//"-c", "copy", // Copy without re-encoding (fast but the clip may not start at the exact time)
		downloadPath,
	)

	return ffmpegCmd, videoTitle, nil
}

// download the video clip and return the file name
func downloadVideo(req RequestData) (string, error) {

	command, title, err := buildClipDownloadCommand(req)

	if err != nil {
		return "", err
	}

	output, err := command.CombinedOutput()
	if err != nil {

		return "", fmt.Errorf("error downloading video: %v, command: %v, output: %v", err, command.String(), string(output))
	}

	slog.Info("Video downloaded", "title", title)
	return title, nil
}
