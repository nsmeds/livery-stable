// Package audio provides hand-rolled magic-byte format detection and duration
// computation for the audio formats livery-stable accepts: MP3, AAC (ADTS),
// WAV, and AIFF.
package audio

import "fmt"

// Format identifies a supported audio container/encoding.
type Format string

const (
	MP3  Format = "mp3"
	AAC  Format = "aac"
	WAV  Format = "wav"
	AIFF Format = "aiff"
)

// DetectFormat identifies an audio format from the leading bytes of a file.
// header must contain at least the first 12 bytes of the file.
func DetectFormat(header []byte) (Format, error) {
	if len(header) < 12 {
		return "", fmt.Errorf("audio: header too short to detect format")
	}

	if string(header[0:4]) == "RIFF" && string(header[8:12]) == "WAVE" {
		return WAV, nil
	}
	if string(header[0:4]) == "FORM" && (string(header[8:12]) == "AIFF" || string(header[8:12]) == "AIFC") {
		return AIFF, nil
	}
	if string(header[0:3]) == "ID3" {
		return MP3, nil
	}
	if header[0] == 0xFF && header[1]&0xE0 == 0xE0 {
		// MP3 frame sync and AAC ADTS sync both start with 11+ set bits, so
		// they're ambiguous here. MP3 forbids a "00" layer value (reserved),
		// while ADTS always sets these same two bits to "00" - that's the
		// one field that reliably tells them apart.
		if (header[1]>>1)&0x03 == 0 {
			return AAC, nil
		}
		return MP3, nil
	}

	return "", fmt.Errorf("audio: unrecognized file format")
}
