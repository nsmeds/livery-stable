package audio_test

import (
	"bytes"
	"encoding/binary"
	"testing"
	"time"

	"github.com/nsmeds/livery-stable/audio"
)

const durationEpsilon = time.Millisecond

func assertDuration(t *testing.T, got, want time.Duration) {
	t.Helper()
	diff := got - want
	if diff < 0 {
		diff = -diff
	}
	if diff > durationEpsilon {
		t.Errorf("duration = %v, want %v (+/- %v)", got, want, durationEpsilon)
	}
}

func buildWAV(sampleRate, byteRate uint32, dataSize uint32) []byte {
	var buf bytes.Buffer
	buf.WriteString("RIFF")
	binary.Write(&buf, binary.LittleEndian, uint32(36+dataSize))
	buf.WriteString("WAVE")

	buf.WriteString("fmt ")
	binary.Write(&buf, binary.LittleEndian, uint32(16))
	binary.Write(&buf, binary.LittleEndian, uint16(1)) // PCM
	binary.Write(&buf, binary.LittleEndian, uint16(1)) // mono
	binary.Write(&buf, binary.LittleEndian, sampleRate)
	binary.Write(&buf, binary.LittleEndian, byteRate)
	binary.Write(&buf, binary.LittleEndian, uint16(2))  // block align
	binary.Write(&buf, binary.LittleEndian, uint16(16)) // bits per sample

	buf.WriteString("data")
	binary.Write(&buf, binary.LittleEndian, dataSize)
	buf.Write(make([]byte, dataSize))

	return buf.Bytes()
}

func TestDuration_WAV(t *testing.T) {
	// 44100 Hz, 16-bit, mono -> byte rate = 88200. 1 second of data.
	data := buildWAV(44100, 88200, 88200)
	r := bytes.NewReader(data)

	got, err := audio.Duration(audio.WAV, r, int64(len(data)))
	if err != nil {
		t.Fatalf("Duration() error = %v", err)
	}
	assertDuration(t, got, time.Second)
}

func buildAIFF(rateBytes [10]byte, numSampleFrames uint32) []byte {
	var buf bytes.Buffer
	buf.WriteString("FORM")
	binary.Write(&buf, binary.BigEndian, uint32(4+8+18)) // "AIFF" + COMM chunk header + payload
	buf.WriteString("AIFF")

	buf.WriteString("COMM")
	binary.Write(&buf, binary.BigEndian, uint32(18))
	binary.Write(&buf, binary.BigEndian, uint16(1)) // mono
	binary.Write(&buf, binary.BigEndian, numSampleFrames)
	binary.Write(&buf, binary.BigEndian, uint16(16)) // sample size
	buf.Write(rateBytes[:])

	return buf.Bytes()
}

func TestDuration_AIFF(t *testing.T) {
	// 44100.0 encoded as an IEEE-754 80-bit extended float (big-endian).
	rate44100 := [10]byte{0x40, 0x0e, 0xac, 0x44, 0, 0, 0, 0, 0, 0}
	data := buildAIFF(rate44100, 44100)
	r := bytes.NewReader(data)

	got, err := audio.Duration(audio.AIFF, r, int64(len(data)))
	if err != nil {
		t.Fatalf("Duration() error = %v", err)
	}
	assertDuration(t, got, time.Second)
}

func TestDuration_MP3_WithXingHeader(t *testing.T) {
	// MPEG1, Layer III, mono, no CRC: FF FB 90 C0.
	// Bitrate index 9 = 128kbps, sample rate index 0 = 44100Hz.
	var buf bytes.Buffer
	buf.Write([]byte{0xFF, 0xFB, 0x90, 0xC0})
	buf.Write(make([]byte, 17)) // mono side info
	buf.WriteString("Xing")
	binary.Write(&buf, binary.BigEndian, uint32(1))  // flags: frame count present
	binary.Write(&buf, binary.BigEndian, uint32(50)) // 50 frames

	data := buf.Bytes()
	r := bytes.NewReader(data)

	got, err := audio.Duration(audio.MP3, r, int64(len(data)))
	if err != nil {
		t.Fatalf("Duration() error = %v", err)
	}
	// 50 frames * 1152 samples/frame / 44100 Hz.
	sampleRate := 44100.0
	want := time.Duration(float64(50*1152) / sampleRate * float64(time.Second))
	assertDuration(t, got, want)
}

func TestDuration_MP3_BitrateEstimate(t *testing.T) {
	// Same header as above but without a Xing tag, so duration falls back to
	// a bitrate-based estimate: 128kbps over a reported 16000-byte file is
	// exactly 1 second.
	var buf bytes.Buffer
	buf.Write([]byte{0xFF, 0xFB, 0x90, 0xC0})
	buf.Write(make([]byte, 21)) // side info + non-matching bytes where Xing would be

	data := buf.Bytes()
	r := bytes.NewReader(data)

	got, err := audio.Duration(audio.MP3, r, 16000)
	if err != nil {
		t.Fatalf("Duration() error = %v", err)
	}
	assertDuration(t, got, time.Second)
}

func buildADTSFrame(freqIdx byte, frameLen int) []byte {
	frame := make([]byte, frameLen)
	frame[0] = 0xFF
	frame[1] = 0xF1 // sync + MPEG-4 + layer 00 + protection absent
	frame[2] = 0x01<<6 | (freqIdx << 2)
	frame[3] = byte((frameLen >> 11) & 0x03)
	frame[4] = byte((frameLen >> 3) & 0xFF)
	frame[5] = byte((frameLen & 0x07) << 5)
	frame[6] = 0x00
	return frame
}

func TestDuration_AAC(t *testing.T) {
	frame := buildADTSFrame(4, 100) // freq index 4 = 44100Hz
	var buf bytes.Buffer
	buf.Write(frame)
	buf.Write(frame)

	data := buf.Bytes()
	r := bytes.NewReader(data)

	got, err := audio.Duration(audio.AAC, r, int64(len(data)))
	if err != nil {
		t.Fatalf("Duration() error = %v", err)
	}
	sampleRate := 44100.0
	want := time.Duration(float64(2*1024) / sampleRate * float64(time.Second))
	assertDuration(t, got, want)
}
