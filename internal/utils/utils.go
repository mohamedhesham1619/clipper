package utils

import (
	"fmt"
	"math/rand/v2"
	"strconv"
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

// parse clip timing info.
// for ffmpeg to accurately extract the needed clip, it needs the start time and clip duration in seconds
func ParseClipDuration(startTime, endTime string) (duration string, err error) {
	layout := "15:04:05"

	t1, err := time.Parse(layout, startTime)
	if err != nil {
		return "", fmt.Errorf("invalid start time: %v", err)
	}

	t2, err := time.Parse(layout, endTime)
	if err != nil {
		return "", fmt.Errorf("invalid end time: %v", err)
	}

	// Calculate duration in seconds
	durationSeconds := int(t2.Sub(t1).Seconds())

	// Convert duration to string
	duration = strconv.Itoa(durationSeconds)

	return duration, nil
}

// FormatSecondsToMMSS converts a string representing seconds to mm:ss format.
func FormatSecondsToMMSS(secondsStr string) string {
	seconds, err := strconv.Atoi(secondsStr)
	if err != nil || seconds < 0 {
		return "00:00"
	}
	minutes := seconds / 60
	secs := seconds % 60
	return fmt.Sprintf("%02d:%02d", minutes, secs)
}
