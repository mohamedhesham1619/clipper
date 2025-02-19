package utils

import (
	"bufio"
	"clipper/models"
	"fmt"
	"log/slog"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

// ==============================================================|
// TODO: MAKE THE OPTION TO DOWNLOAD 360, 460, 720, 1080 QULAITY |
// ==============================================================|

// prepare the command to download the clip of the video
func BuildClipDownloadCommand(videoRequest models.VideoRequest) (*exec.Cmd, string, error) {

	// Get both the URL and the title with the extension
	cmd := exec.Command("./yt-dlp",
		"-f", "bestvideo[height<=1080]+bestaudio",
		"--get-title",       // Get the video title
		"--get-url",         // Get video and audio URLs
		"--encoding", "UTF-8",
		"--no-download",
		videoRequest.VideoURL,
	)

	output, err := cmd.Output()

	if err != nil {

		return nil, "", fmt.Errorf("error getting video info: %v, command: %v, output: %v", err, cmd.String(), string(output))
	}

	// Split output into lines
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")

	if len(lines) < 3 {

		return nil, "", fmt.Errorf("expected both URL and title but got: %v", lines)
	}

	videoTitle := SanitizeFilename(lines[0]) + "-1080.mp4"
	videoURL := lines[1]
	audioURL := lines[2]

	slog.Debug("video title sanitization", "before:", lines[0], "after:", videoTitle)

	// Get the absolute path to the download directory
	downloadPath, _ := filepath.Abs(fmt.Sprintf("temp/%v", videoTitle))

	// Prepare the command to download the video clip
	ffmpegCmd := exec.Command(
		"./ffmpeg",
		"-ss", videoRequest.ClipStart,
		"-i", videoURL,
		"-i", audioURL,
		"-to", videoRequest.ClipEnd,
		"-progress", "pipe:1",
		"-c", "copy", // Copy without re-encoding (fast and decrease the cpu usage but the clip may not start at the exact time).
		downloadPath,
	)

	return ffmpegCmd, videoTitle, nil
}

// download the clip and return the file name and a channel to share the progress
func DownloadVideo(videoRequest models.VideoRequest) (string, chan models.ProgressResponse, error) {

	command, title, err := BuildClipDownloadCommand(videoRequest)

	if err != nil {
		return "", nil, err
	}

	stdoutPipe, err := command.StdoutPipe()

	if err != nil {
		return "", nil, fmt.Errorf("error creating stdout pipe: %v", err)
	}

	progressChan := make(chan models.ProgressResponse)
	scanner := bufio.NewScanner(stdoutPipe)

	// total time in microseconds
	// it is required to calculate the progress because ffmpeg returns the output time in microseconds
	totalTime, err := calculateClipDuration(videoRequest.ClipStart, videoRequest.ClipEnd)

	if err != nil {
		return "", nil, fmt.Errorf("error calculating clip duration in microseconds: %v", err)
	}

	// start listening to stdout pipe
	// since this is I/O blocking process, it needs to start in a separate goroutine
	go func() {
		for scanner.Scan() {
			line := scanner.Text()

			if strings.Contains(line, "out_time_ms") {
				outTime, err := strconv.ParseInt(strings.Split(line, "=")[1], 10, 64)

				if err != nil {
					slog.Error(fmt.Sprintf("error reading out_time_ms value from downloading command output: %v", err))
				}

				// Convert to float64 to avoid integer division truncation and get precise percentage
				progress := (float64(outTime) / float64(totalTime)) * 100

				progressChan <- models.ProgressResponse{
					Status:   "in_progress",
					Progress: int(progress),
				}
			}
		}

		// close the channel after the download finish
		close(progressChan)
	}()

	// run the download command
	err = command.Start()

	if err != nil {
		return "", nil, err
	}

	return title, progressChan, nil
}
