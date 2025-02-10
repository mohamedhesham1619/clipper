package utils

import (
	"fmt"
	"log/slog"
	"math/rand/v2"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
	"unicode"
)

func GenerateID() string {

	randNum := rand.Int32N(10000)
	return fmt.Sprintf("%d%d", time.Now().UnixNano(), randNum)
}

// sanitize the filename to remove or replace characters that are problematic in filenames
func SanitizeFilename(filename string) string {

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

	// Replace problematic characters with a hyphen and remove non-printable characters
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
func BuildClipDownloadCommand(videoUrl string, clipDuration string) (*exec.Cmd, string, error) {

	// Get both the URL and the title with the extension
	cmd := exec.Command("./yt-dlp",
		"-f", "b",
		"--print", "%(title)s.%(ext)s\n%(url)s",
		"--encoding", "UTF-8",
		"--no-download",
		videoUrl,
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

	videoTitle := SanitizeFilename(lines[0])
	videoURL := lines[1]

	slog.Info("video title sanitization", "before:", lines[0], "after:", videoTitle)

	// download the clip to the current directory with the title as the file name
	downloadPath, _ := filepath.Abs(fmt.Sprintf("../assets/videos/%v", videoTitle))

	duration := strings.Split(clipDuration, "-")
	clipStart := duration[0]
	clipEnd := duration[1]

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
func DownloadVideo(videoUrl string, clipDuration string) (string, error) {

	command, title, err := BuildClipDownloadCommand(videoUrl, clipDuration)

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
