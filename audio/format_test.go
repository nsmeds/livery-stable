package audio_test

import (
	"testing"

	"github.com/nsmeds/livery-stable/audio"
)

func TestDetectFormat(t *testing.T) {
	cases := []struct {
		name   string
		header []byte
		want   audio.Format
	}{
		{
			name:   "wav",
			header: []byte("RIFF\x24\x00\x00\x00WAVEfmt "),
			want:   audio.WAV,
		},
		{
			name:   "aiff",
			header: []byte("FORM\x00\x00\x00\x24AIFFCOMM"),
			want:   audio.AIFF,
		},
		{
			name:   "aiff-c",
			header: []byte("FORM\x00\x00\x00\x24AIFCCOMM"),
			want:   audio.AIFF,
		},
		{
			name:   "mp3 with id3v2 tag",
			header: []byte("ID3\x04\x00\x00\x00\x00\x00\x00\x00\x00"),
			want:   audio.MP3,
		},
		{
			name:   "mp3 raw frame sync (mpeg1 layer3, mono)",
			header: []byte{0xFF, 0xFB, 0x90, 0xC0, 0, 0, 0, 0, 0, 0, 0, 0},
			want:   audio.MP3,
		},
		{
			name:   "aac adts sync",
			header: []byte{0xFF, 0xF1, 0x50, 0x40, 0x0C, 0x80, 0x00, 0, 0, 0, 0, 0},
			want:   audio.AAC,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := audio.DetectFormat(tc.header)
			if err != nil {
				t.Fatalf("DetectFormat() error = %v", err)
			}
			if got != tc.want {
				t.Errorf("DetectFormat() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestDetectFormat_Unrecognized(t *testing.T) {
	_, err := audio.DetectFormat([]byte("not an audio file!!"))
	if err == nil {
		t.Fatal("DetectFormat() error = nil, want error for garbage header")
	}
}

func TestDetectFormat_HeaderTooShort(t *testing.T) {
	_, err := audio.DetectFormat([]byte{0xFF, 0xFB})
	if err == nil {
		t.Fatal("DetectFormat() error = nil, want error for short header")
	}
}
