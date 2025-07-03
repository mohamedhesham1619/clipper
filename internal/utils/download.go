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

	// 1. Get video title to determine output filename.
	// This is a quick, separate command to avoid mixing its output with the video stream.
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
		return "", nil, nil, fmt.Errorf("yt-dlp failed to get video info: %w, output: %s", err, string(infoOutput))
	}
	if len(infoOutput) == 0 {
		return "", nil, nil, fmt.Errorf("yt-dlp returned no info for the video")
	}
	videoTitle := SanitizeFilename(strings.TrimSpace(string(infoOutput)))

	// 2. Prepare paths and directories.
	if err := os.MkdirAll("temp", os.ModePerm); err != nil {
		return "", nil, nil, fmt.Errorf("failed to create temp directory: %w", err)
	}
	downloadPath, _ := filepath.Abs(filepath.Join("temp", videoTitle))

	// 3. Prepare yt-dlp command for streaming video data to its stdout.
	ytdlpCmd := exec.Command("yt-dlp",
		"-f", fmt.Sprintf("bv*[height<=%[1]v]+ba/b[height<=%[1]v]/best", videoRequest.Quality),
		"--no-warnings",
		"-o", "-", // Critical: output to stdout
		videoRequest.VideoURL,
	)

	// 4. Prepare ffmpeg command to read from its stdin.
	clipDuration, err := ParseClipDuration(videoRequest.ClipStart, videoRequest.ClipEnd)
	if err != nil {
		return "", nil, nil, fmt.Errorf("failed to parse clip duration: %w", err)
	}
	ffmpegCmd := exec.Command("ffmpeg",
		"-hide_banner", // Quieter logs
		"-ss", videoRequest.ClipStart,
		"-i", "pipe:0", // Critical: read from stdin
		"-t", clipDuration,
		"-progress", "pipe:1", // Progress to stdout
		"-c", "copy",
		downloadPath,
	)

	// 5. Connect yt-dlp's stdout to ffmpeg's stdin.
	ffmpegCmd.Stdin, err = ytdlpCmd.StdoutPipe()
	if err != nil {
		return "", nil, nil, fmt.Errorf("failed to create pipe from yt-dlp to ffmpeg: %w", err)
	}

	// 6. Set up ffmpeg's stdout for progress and stderr for logs.
	totalTime, err := calculateClipDuration(videoRequest.ClipStart, videoRequest.ClipEnd)
	if err != nil {
		return "", nil, nil, fmt.Errorf("error calculating clip duration in microseconds: %v", err)
	}
	progressChan := make(chan models.ProgressResponse)

	ffmpegStdout, err := ffmpegCmd.StdoutPipe()
	if err != nil {
		return "", nil, nil, fmt.Errorf("error creating stdout pipe: %v", err)
	}
	go readProgress(ffmpegStdout, progressChan, totalTime)

	ffmpegStderr, err := ffmpegCmd.StderrPipe()
	if err != nil {
		return "", nil, nil, fmt.Errorf("error creating stderr pipe: %v", err)
	}
	go logPipe(ffmpegStderr, "ffmpeg")

	// Also log yt-dlp's stderr for debugging any download-specific issues.
	ytdlpStderr, err := ytdlpCmd.StderrPipe()
	if err != nil {
		return "", nil, nil, fmt.Errorf("error creating yt-dlp stderr pipe: %v", err)
	}
	go logPipe(ytdlpStderr, "yt-dlp")

	// 7. Start both commands. Order matters: start the consumer (ffmpeg) before the producer (yt-dlp) to be safe,
	// although starting yt-dlp first is generally fine as the pipe has a buffer.
	if err := ytdlpCmd.Start(); err != nil {
		return "", nil, nil, fmt.Errorf("failed to start yt-dlp: %w", err)
	}
	if err := ffmpegCmd.Start(); err != nil {
		// If ffmpeg fails to start, we must kill the lingering yt-dlp process.
		_ = ytdlpCmd.Process.Kill()
		return "", nil, nil, fmt.Errorf("failed to start ffmpeg: %w", err)
	}

	// 8. The caller will Wait() on ffmpeg. We spawn a goroutine to Wait() on yt-dlp
	// to release its resources once it's done (or killed by a broken pipe).
	go func() {
		// A "broken pipe" error is expected if ffmpeg finishes first and closes its stdin.
		// We can log this at a debug level as it's part of normal operation in this pipeline.
		if err := ytdlpCmd.Wait(); err != nil {
			slog.Debug("yt-dlp process finished", "error", err)
		}
	}()

	return downloadPath, progressChan, ffmpegCmd, nil
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
