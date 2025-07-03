package utils

import (
	"bufio"
	"clipper/internal/models"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

// get the title of the video and prepare the command to download the clip
func buildClipDownloadCommand(videoRequest models.VideoRequest) (*exec.Cmd, string, error) {

	// Get the video title, video url, and audio url with the desired quality using yt-dlp
	cmd := exec.Command("yt-dlp",
		"-f", fmt.Sprintf("bv*[height<=%[1]v]+ba/b[height<=%[1]v]/best", videoRequest.Quality),
		"--print", "%(title)s-%(height)sp.%(ext)s\n%(urls)s",
		"--encoding", "utf-8",
		"--no-playlist",
		"--no-download",
		"--no-warnings",
		videoRequest.VideoURL,
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, "", fmt.Errorf("error getting video info: %v, command: %v, output: %v", err, cmd.String(), string(output))
	}

	// Split output into lines
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	if len(lines) < 2 {
		return nil, "", fmt.Errorf("expected both URL and title but got: %v", lines)
	}

	videoTitle := SanitizeFilename(lines[0])
	clipDuration, err := ParseClipDuration(videoRequest.ClipStart, videoRequest.ClipEnd)
	if err != nil {
		return nil, "", fmt.Errorf("error parsing clip duration: %v", err)
	}

	// Create the temp directory if it doesn't exist
	err = os.MkdirAll("temp", os.ModePerm)
	if err != nil {
		return nil, "", fmt.Errorf("error creating temp directory: %v", err)
	}

	// Get the absolute path to the temp directory
	downloadPath, _ := filepath.Abs(filepath.Join("temp", videoTitle))

	var ffmpegCmd *exec.Cmd

	// Construct a single header string to mimic a browser request.
	// This is more robust than multiple -headers flags and helps avoid 403 errors.
	// The \r\n is crucial for separating headers.
	headerString := "User-Agent: Mozilla/5.0\r\n" +
		"Accept: text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,*/*;q=0.8\r\n" +
		"Accept-Language: en-US,en;q=0.5\r\n" +
		"Referer: https://www.google.com/\r\n" // A generic Referer can help with services like Google Drive.

	// If yt-dlp returns separate URLs for audio and video
	if len(lines) > 2 { // Separate video and audio streams
		videoURL := lines[1] // URL of the video stream
		audioURL := lines[2] // URL of the audio stream

		ffmpegCmd = exec.Command(
			"ffmpeg",
			"-headers", headerString,
			"-ss", videoRequest.ClipStart,
			"-i", videoURL,
			"-headers", headerString,
			"-ss", videoRequest.ClipStart,
			"-i", audioURL,
			"-t", clipDuration,
			"-progress", "pipe:1",
			"-c", "copy", // Copy without re-encoding (fast but the clip may not start at the exact time)
			downloadPath,
		)
	} else { // If yt-dlp returns a single URL (video + audio combined)
		url := lines[1] // Combined video and audio URL
		ffmpegCmd = exec.Command(
			"ffmpeg",
			"-headers", headerString,
			"-ss", videoRequest.ClipStart,
			"-i", url,
			"-t", clipDuration,
			"-progress", "pipe:1",
			"-c", "copy",
			downloadPath,
		)
	}

	return ffmpegCmd, downloadPath, nil
}

// download the clip and return the file name and a channel to share the progress
func DownloadVideo(videoRequest models.VideoRequest) (string, chan models.ProgressResponse, *exec.Cmd, error) {

	command, filePath, err := buildClipDownloadCommand(videoRequest)

	if err != nil {
		return "", nil, nil, err
	}

	// total time in microseconds
	// it is required to calculate the progress because ffmpeg returns the output time in microseconds
	totalTime, err := calculateClipDuration(videoRequest.ClipStart, videoRequest.ClipEnd)

	if err != nil {
		return "", nil, nil, fmt.Errorf("error calculating clip duration in microseconds: %v", err)
	}

	// Create a pipe to read the command's stdout
	// This is necessary to capture the progress output from ffmpeg
	stdoutPipe, err := command.StdoutPipe()

	if err != nil {
		return "", nil, nil, fmt.Errorf("error creating stdout pipe: %v", err)
	}

	// Also create a pipe to read stderr for logging purposes. This is crucial for debugging.
	stderrPipe, err := command.StderrPipe()
	if err != nil {
		return "", nil, nil, fmt.Errorf("error creating stderr pipe: %v", err)
	}

	progressChan := make(chan models.ProgressResponse)
	scanner := bufio.NewScanner(stdoutPipe)

	// Start a goroutine to log stderr for debugging.
	// This will show us exactly what ffmpeg is doing or why it's failing.
	go func() {
		stderrScanner := bufio.NewScanner(stderrPipe)
		for stderrScanner.Scan() {
			// Log ffmpeg's output for debugging. Use Debug level to avoid cluttering logs in production.
			slog.Debug("ffmpeg", "output", stderrScanner.Text())
		}
	}()

	// start listening to stdout pipe
	// since this is I/O blocking process, it needs to start in a separate goroutine
	go func() {
		for scanner.Scan() {
			line := scanner.Text()

			if strings.Contains(line, "out_time_ms") {
				outTime, err := strconv.ParseInt(strings.Split(line, "=")[1], 10, 64)

				if err != nil {
					slog.Error("error parsing out_time_ms from ffmpeg", "error", err)
				}

				// Convert to float64 to avoid integer division truncation and get precise percentage
				progress := (float64(outTime) / float64(totalTime)) * 100

				progressChan <- models.ProgressResponse{
					Status:   "in_progress",
					Progress: int(progress),
				}
			}
		}

	}()

	// run the download command
	err = command.Start()

	if err != nil {
		return "", nil, nil, err
	}

	return filePath, progressChan, command, nil
}
