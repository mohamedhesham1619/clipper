package utils

import (
	"fmt"
	"math/rand/v2"
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
