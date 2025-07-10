package utils

import (
	"bufio"
	"clipper/internal/models"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

// DownloadVideo downloads the video and returns the output file path, a progress channel, and the ffmpeg command.
func DownloadVideo(videoRequest models.VideoRequest) (string, chan models.ProgressResponse, *exec.Cmd, error) {

	// Start a goroutine to get the video title and construct the download path.
	// This is a separate command to avoid mixing its output with the video stream.
	downloadPathChan := make(chan string)
	defer close(downloadPathChan)

	go func() {
		
		videoTitle, err := getVideoTitle(videoRequest)
		if err != nil {
			slog.Error("failed to get video title", "error", err)
			downloadPathChan <- ""
			return
		}

		// Construct the download path using the sanitized title.
		// This will be used to save the video file.
		downloadPath := filepath.Join("temp", videoTitle)

		downloadPathChan <- downloadPath
	}()

	// Create a temp directory if it doesn't exist.
	// This is where the video will be saved.
	if err := os.MkdirAll("temp", os.ModePerm); err != nil {
		return "", nil, nil, fmt.Errorf("failed to create temp directory: %w", err)
	}

	// Create the yt-dlp and ffmpeg commands.
	ytdlpCmd := prepareYtDlpCommand(videoRequest)
	ffmpegCmd := prepareFfmpegCommand()

	err := preparePipes(ytdlpCmd, ffmpegCmd)

	if err != nil {
		return "", nil, nil, fmt.Errorf("error preparing pipes: %v", err)
	}

	// Calculate the total clip duration.
	// This is used to calculate the progress percentage.
	totalTime, err := calculateClipDuration(videoRequest.ClipStart, videoRequest.ClipEnd)
	if err != nil {
		return "", nil, nil, fmt.Errorf("error calculating clip duration in microseconds: %v", err)
	}

	// Create a channel to send progress updates from ffmpeg.
	// This will be used to communicate the download progress back to the caller.
	progressChan := make(chan models.ProgressResponse)

	// Create pipes for ffmpeg's output.
	// This pipe will be used to read progress updates.
	ffmpegStdout, err := ffmpegCmd.StdoutPipe()
	if err != nil {
		return "", nil, nil, fmt.Errorf("error creating ffmpeg stdout pipe: %v", err)
	}
	go readProgress(ffmpegStdout, progressChan, totalTime)

	// Get the download path from the channel when it's ready and add it to the ffmpeg command.
	downloadPath := <-downloadPathChan
	if downloadPath == "" {
		return "", nil, nil, fmt.Errorf("failed to get video title for download path")
	}
	ffmpegCmd.Args = append(ffmpegCmd.Args, downloadPath)

	// Start both commands.
	if err := ytdlpCmd.Start(); err != nil {
		return "", nil, nil, fmt.Errorf("failed to start yt-dlp: %w", err)
	}
	if err := ffmpegCmd.Start(); err != nil {
		// If ffmpeg fails to start, we must kill the lingering yt-dlp process.
		_ = ytdlpCmd.Process.Kill()
		return "", nil, nil, fmt.Errorf("failed to start ffmpeg: %w", err)
	}

	// This goroutine will release yt-dlp's resources once it's done and will log any errors.
	// The caller of this function will handle the ffmpeg process so we don't need to wait for it here.
	go func() {
		if err := ytdlpCmd.Wait(); err != nil {
			slog.Debug("yt-dlp process finished", "error", err)
		}
	}()

	return downloadPath, progressChan, ffmpegCmd, nil
}

func prepareYtDlpCommand(videoRequest models.VideoRequest) *exec.Cmd {
	return exec.Command("yt-dlp",
		"-f", fmt.Sprintf("bv*[height<=%[1]v]+ba/b[height<=%[1]v]/best", videoRequest.Quality),
		"--download-sections", fmt.Sprintf("*%s-%s", videoRequest.ClipStart, videoRequest.ClipEnd),
		"--no-warnings",
		"-o", "-", // Output to stdout
		videoRequest.VideoURL,
	)
}

func prepareFfmpegCommand() *exec.Cmd {
	return exec.Command("ffmpeg",
		"-hide_banner", // Quieter logs
		"-i", "pipe:0", // Read from stdin
		"-progress", "pipe:1", // Progress to stdout
		"-c", "copy", // Just copy the stream yt-dlp provides.
	)
}

// preparePipes sets up the necessary pipes for yt-dlp and ffmpeg commands.
func preparePipes(ytdlpCmd *exec.Cmd, ffmpegCmd *exec.Cmd) error {

	// This pipe will be used to log ffmpeg's stderr output.
	// It's useful for debugging any issues with the ffmpeg command.
	ffmpegStderr, err := ffmpegCmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("error creating stderr pipe: %v", err)
	}
	go logPipe(ffmpegStderr, "ffmpeg")

	// Also log yt-dlp's stderr for debugging any download-specific issues.
	ytdlpStderr, err := ytdlpCmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("error creating yt-dlp stderr pipe: %v", err)
	}
	go logPipe(ytdlpStderr, "yt-dlp")

	// Set the stdin of ffmpeg to read from yt-dlp's stdout.
	// This allows ffmpeg to process the video data streamed by yt-dlp.
	ffmpegCmd.Stdin, err = ytdlpCmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("error connecting yt-dlp stdout to ffmpeg stdin: %v", err)
	}

	return nil
}

// getVideoTitle retrieves the video title using yt-dlp and sanitizes it for use as a filename.
func getVideoTitle(videoRequest models.VideoRequest) (string, error) {
	infoCmd := exec.Command("yt-dlp",
		"-f", fmt.Sprintf("bv*[height<=%[1]v]+ba/b[height<=%[1]v]/best", videoRequest.Quality),
		"--print", "%(title)s-%(height)sp.%(ext)s",
		"--no-playlist",
		"--no-download",
		"--no-warnings",
		videoRequest.VideoURL,
	)

	infoOutput, err := infoCmd.CombinedOutput()

	if err != nil {
		slog.Error("yt-dlp failed to get video info", "error", err, "output", string(infoOutput))
		return "", fmt.Errorf("failed to get video info: %w", err)
	}
	if len(infoOutput) == 0 {
		slog.Error("yt-dlp returned empty output for video info", "videoURL", videoRequest.VideoURL)
		return "", fmt.Errorf("yt-dlp returned empty output for video info: %w", err)
	}
	slog.Debug("yt-dlp video title", "title", string(infoOutput))

	// Sanitize the video title to create a valid filename.
	videoTitle := SanitizeFilename(strings.TrimSpace(string(infoOutput)))
	return videoTitle, nil
}

// logPipe reads from a process's output pipe and logs each line for debugging.
func logPipe(pipe io.ReadCloser, prefix string) {
	scanner := bufio.NewScanner(pipe)
	for scanner.Scan() {
		slog.Debug(prefix, "output", scanner.Text())
	}
}

// readProgress reads from ffmpeg's progress pipe, parses the progress, and sends it to a channel.
func readProgress(pipe io.ReadCloser, progressChan chan models.ProgressResponse, totalTime int64) {
	scanner := bufio.NewScanner(pipe)
	for scanner.Scan() {
		line := scanner.Text()

		if strings.Contains(line, "out_time_ms") {
			outTime, err := strconv.ParseInt(strings.Split(line, "=")[1], 10, 64)

			if err != nil {
				slog.Error("error parsing out_time_ms from ffmpeg", "error", err)
				continue
			}

			// Convert to float64 to avoid integer division truncation and get precise percentage
			progress := (float64(outTime) / float64(totalTime)) * 100

			progressChan <- models.ProgressResponse{
				Status:   "in_progress",
				Progress: int(progress),
			}
		}
	}
}
