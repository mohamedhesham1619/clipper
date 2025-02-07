package main

import (
	//"encoding/json"
	"fmt"
	"io"
	"os"

	"net/http"

	"os/exec"
	"strings"
	"unicode"
)

type RequestData struct {
	VideoURL     string `json:"videoUrl"`
	ClipDuration string `json:"clipDuration"`
}

func main() {

	mux := http.NewServeMux()
	
	
	mux.HandleFunc("/submit", func(w http.ResponseWriter, r *http.Request) {
		// var data RequestData
		// json.NewDecoder(r.Body).Decode(&data)
		//w.Write([]byte(fmt.Sprintf("received the following form data: %v - %v", data.VideoURL, data.ClipDuration)))

		data := RequestData{
			VideoURL:     r.FormValue("videoUrl"),
			ClipDuration: r.FormValue("clipDuration")}

		// todo: implemet sse to send clipping progress to the client
		videoTitle := downloadVideo(data)
		fmt.Println("Video Title:", videoTitle)

		
	})

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "page.html")
	})

	mux.HandleFunc("/download", func(w http.ResponseWriter, r *http.Request) {
		fileName := r.URL.Query().Get("file")
		file, _ := os.Open(fileName)
		defer file.Close()
		// set response headers for download
		w.Header().Set("Content-Type", "video/mp4")
		w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%v", fileName))
		io.Copy(w, file)
	})
	// Create a new instance of the server
	server := http.Server{Handler: mux, Addr: ":8080"}

	// Start the server
	fmt.Println("Server is running on port 8080")
	server.ListenAndServe()
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
func buildClipDownloadCommand(req RequestData) (*exec.Cmd, string) {

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
		fmt.Println("Error getting video URL and title:", err)
	}

	// Split output into lines
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")

	if len(lines) < 2 {

		fmt.Println("expected both URL and title but got:", lines)
	}

	videoTitle := sanitizeFilename(lines[0])
	videoURL := lines[1]

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

	return ffmpegCmd, videoTitle
}

// get the download command for the video request and run it
func downloadVideo(req RequestData) string {

	command, title := buildClipDownloadCommand(req)

	// Run the command
	fmt.Println("Downloading video:", req.VideoURL)

	output, err := command.CombinedOutput()

	fmt.Println("Output:", string(output))
	if err != nil {
		fmt.Println("Error downloading video:", err)
		return ""
	}

	fmt.Println("Download complete.")
	return title
}
