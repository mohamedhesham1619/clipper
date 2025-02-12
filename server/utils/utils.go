package utils

import (
	"bufio"
	"clipper/models"
	"fmt"
	"strconv"

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

	// better command: ./yt-dlp -f "best" --external-downloader ffmpeg --external-downloader-args "ffmpeg_i:-ss 0 -t 60" -o "%(title)s.%(ext)s" "https://youtu.be/zaFS-Qs1mSc?si=UPdiVrfDiCf7B2VO"

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

	slog.Debug("video title sanitization", "before:", lines[0], "after:", videoTitle)

	// Get the absolute path to the download directory
	downloadPath, _ := filepath.Abs(fmt.Sprintf("../assets/videos/%v", videoTitle))

	// Split the clip duration into start and end times
	duration := strings.Split(clipDuration, "-")
	clipStart := duration[0]
	clipEnd := duration[1]

	// Prepare the command to download the video clip
	ffmpegCmd := exec.Command(
		"./ffmpeg", "-i", videoURL,
		"-ss", clipStart, // Set the clip start and end time
		"-to", clipEnd,
		"-progress", "pipe:1",
		//"-c", "copy", // Copy without re-encoding (fast but the clip may not start at the exact time)
		downloadPath,
	)

	return ffmpegCmd, videoTitle, nil
}

// download the video clip and return the file name
func DownloadVideo(videoUrl string, clipDuration string) (string, chan models.ProgressResponse, error) {

	command, title, err := BuildClipDownloadCommand(videoUrl, clipDuration)

	if err != nil {
		return "", nil, err
	}

	stdoutPipe, err := command.StdoutPipe()

	if err != nil {
		return "", nil, fmt.Errorf("error creating stdout pipe: %v", err)
	}

	progressChan := make(chan models.ProgressResponse)
	scanner := bufio.NewScanner(stdoutPipe)

	totalTime, _ := calculateClipDuration(strings.Split(clipDuration, "-")[0], strings.Split(clipDuration, "-")[1])

	go func() {
		for scanner.Scan() {
			line := scanner.Text()

			if strings.Contains(line, "out_time_ms") {
				outTime, _ := strconv.ParseInt(strings.Split(line, "=")[1], 10, 64)

				// Convert to float64 to avoid integer division truncation and get precise percentage
				progress := (float64(outTime) / float64(totalTime)) * 100

				progressChan <- models.ProgressResponse{
					Status:   "in_progress",
					Progress: int(progress),
				}
			}
		}
		close(progressChan)
	}()

	err = command.Start()

	if err != nil {
		return "", nil, err
	}

	return title, progressChan, nil
}

// calculate the clip duration in microseconds
func calculateClipDuration(start, end string) (int64, error) {

	layout := "15:04:05"
	startTime, err := time.Parse(layout, start)

	if err != nil {
		return 0, err
	}

	endTime, err := time.Parse(layout, end)

	if err != nil {
		return 0, err
	}

	return endTime.Sub(startTime).Microseconds(), nil

}
