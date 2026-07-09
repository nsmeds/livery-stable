package audio

import (
	"encoding/binary"
	"fmt"
	"io"
	"math"
	"time"
)

// Duration computes the playback length of an audio file. r must be
// positioned so that reads start from the beginning of the file; size is the
// total file size in bytes.
func Duration(format Format, r io.ReadSeeker, size int64) (time.Duration, error) {
	switch format {
	case WAV:
		return durationWAV(r)
	case AIFF:
		return durationAIFF(r)
	case MP3:
		return durationMP3(r, size)
	case AAC:
		return durationAAC(r, size)
	default:
		return 0, fmt.Errorf("audio: unsupported format %q", format)
	}
}

// durationWAV walks RIFF chunks looking for "fmt " (byte rate) and "data"
// (payload size); duration is data size / byte rate.
func durationWAV(r io.ReadSeeker) (time.Duration, error) {
	if _, err := r.Seek(12, io.SeekStart); err != nil {
		return 0, err
	}

	var byteRate uint32
	haveFmt := false

	for {
		var id [4]byte
		if _, err := io.ReadFull(r, id[:]); err != nil {
			return 0, fmt.Errorf("wav: missing data chunk")
		}
		var chunkSize uint32
		if err := binary.Read(r, binary.LittleEndian, &chunkSize); err != nil {
			return 0, err
		}

		switch string(id[:]) {
		case "fmt ":
			var fc struct {
				AudioFormat   uint16
				NumChannels   uint16
				SampleRate    uint32
				ByteRate      uint32
				BlockAlign    uint16
				BitsPerSample uint16
			}
			if err := binary.Read(r, binary.LittleEndian, &fc); err != nil {
				return 0, err
			}
			byteRate = fc.ByteRate
			haveFmt = true
			if extra := int64(chunkSize) - 16; extra > 0 {
				if _, err := r.Seek(extra, io.SeekCurrent); err != nil {
					return 0, err
				}
			}
		case "data":
			if !haveFmt || byteRate == 0 {
				return 0, fmt.Errorf("wav: data chunk before fmt chunk")
			}
			return time.Duration(float64(chunkSize) / float64(byteRate) * float64(time.Second)), nil
		default:
			skip := int64(chunkSize)
			if chunkSize%2 == 1 {
				skip++
			}
			if _, err := r.Seek(skip, io.SeekCurrent); err != nil {
				return 0, err
			}
		}
	}
}

// durationAIFF walks FORM chunks looking for "COMM", which carries the
// sample rate and frame count directly - no need to inspect audio data.
func durationAIFF(r io.ReadSeeker) (time.Duration, error) {
	if _, err := r.Seek(12, io.SeekStart); err != nil {
		return 0, err
	}

	for {
		var id [4]byte
		if _, err := io.ReadFull(r, id[:]); err != nil {
			return 0, fmt.Errorf("aiff: missing COMM chunk")
		}
		var chunkSize uint32
		if err := binary.Read(r, binary.BigEndian, &chunkSize); err != nil {
			return 0, err
		}

		if string(id[:]) == "COMM" {
			var numChannels uint16
			var numSampleFrames uint32
			var sampleSize uint16
			var rateBytes [10]byte
			if err := binary.Read(r, binary.BigEndian, &numChannels); err != nil {
				return 0, err
			}
			if err := binary.Read(r, binary.BigEndian, &numSampleFrames); err != nil {
				return 0, err
			}
			if err := binary.Read(r, binary.BigEndian, &sampleSize); err != nil {
				return 0, err
			}
			if _, err := io.ReadFull(r, rateBytes[:]); err != nil {
				return 0, err
			}
			sampleRate := extendedToFloat64(rateBytes)
			if sampleRate == 0 {
				return 0, fmt.Errorf("aiff: invalid sample rate")
			}
			return time.Duration(float64(numSampleFrames) / sampleRate * float64(time.Second)), nil
		}

		skip := int64(chunkSize)
		if chunkSize%2 == 1 {
			skip++
		}
		if _, err := r.Seek(skip, io.SeekCurrent); err != nil {
			return 0, err
		}
	}
}

// extendedToFloat64 decodes the 80-bit IEEE-754 extended float (big-endian)
// AIFF uses for its sample rate field.
func extendedToFloat64(b [10]byte) float64 {
	sign := 1.0
	if b[0]&0x80 != 0 {
		sign = -1.0
	}
	exponent := (int(b[0]&0x7f) << 8) | int(b[1])
	var mantissa uint64
	for i := 2; i < 10; i++ {
		mantissa = mantissa<<8 | uint64(b[i])
	}
	if exponent == 0 && mantissa == 0 {
		return 0
	}
	return sign * float64(mantissa) * math.Pow(2, float64(exponent-16383-63))
}

// skipID3v2 advances past a leading ID3v2 tag, if present, and returns the
// byte offset it left the reader at. If no ID3v2 tag is present, the reader
// is left at the start.
func skipID3v2(r io.ReadSeeker) (int64, error) {
	var hdr [10]byte
	if _, err := io.ReadFull(r, hdr[:]); err != nil {
		return 0, err
	}
	if string(hdr[0:3]) != "ID3" {
		if _, err := r.Seek(0, io.SeekStart); err != nil {
			return 0, err
		}
		return 0, nil
	}
	size := int64(hdr[6]&0x7f)<<21 | int64(hdr[7]&0x7f)<<14 | int64(hdr[8]&0x7f)<<7 | int64(hdr[9]&0x7f)
	offset := int64(10) + size
	if _, err := r.Seek(offset, io.SeekStart); err != nil {
		return 0, err
	}
	return offset, nil
}

type mp3Header struct {
	versionID        int // raw 2-bit value: 0 = MPEG2.5, 2 = MPEG2, 3 = MPEG1
	bitrateIdx       int
	sampleIdx        int
	channelMode      int // 3 = mono
	protectionAbsent bool
}

var mp3BitrateV1L3 = [16]int{0, 32, 40, 48, 56, 64, 80, 96, 112, 128, 160, 192, 224, 256, 320, 0}
var mp3BitrateV2L3 = [16]int{0, 8, 16, 24, 32, 40, 48, 56, 64, 80, 96, 112, 128, 144, 160, 0}

var mp3SampleRateV1 = [3]int{44100, 48000, 32000}
var mp3SampleRateV2 = [3]int{22050, 24000, 16000}
var mp3SampleRateV25 = [3]int{11025, 12000, 8000}

func mp3Bitrate(versionID, idx int) int {
	if versionID == 3 {
		return mp3BitrateV1L3[idx]
	}
	return mp3BitrateV2L3[idx]
}

func mp3SampleRate(versionID, idx int) int {
	switch versionID {
	case 3:
		return mp3SampleRateV1[idx]
	case 2:
		return mp3SampleRateV2[idx]
	default: // 0 = MPEG2.5
		return mp3SampleRateV25[idx]
	}
}

// parseMP3FrameHeader decodes a 4-byte MPEG frame header. Only Layer III is
// supported, which covers every real-world "MP3" file.
func parseMP3FrameHeader(b [4]byte) (mp3Header, bool) {
	if b[1]&0xE0 != 0xE0 {
		return mp3Header{}, false
	}
	layerBits := (b[1] >> 1) & 0x03
	if layerBits != 1 {
		return mp3Header{}, false
	}
	versionID := int((b[1] >> 3) & 0x03)
	if versionID == 1 {
		return mp3Header{}, false
	}
	protectionAbsent := b[1]&0x01 == 1

	bitrateIdx := int((b[2] >> 4) & 0x0F)
	sampleIdx := int((b[2] >> 2) & 0x03)
	if bitrateIdx == 0 || bitrateIdx == 15 || sampleIdx == 3 {
		return mp3Header{}, false
	}
	channelMode := int((b[3] >> 6) & 0x03)

	return mp3Header{
		versionID:        versionID,
		bitrateIdx:       bitrateIdx,
		sampleIdx:        sampleIdx,
		channelMode:      channelMode,
		protectionAbsent: protectionAbsent,
	}, true
}

// durationMP3 locates the first frame header, then prefers an exact frame
// count from a Xing/Info VBR header if present; otherwise it estimates
// duration from the first frame's bitrate and the remaining file size.
func durationMP3(r io.ReadSeeker, size int64) (time.Duration, error) {
	if _, err := r.Seek(0, io.SeekStart); err != nil {
		return 0, err
	}
	offset, err := skipID3v2(r)
	if err != nil {
		return 0, err
	}

	frameStart, hdr, err := findMP3Frame(r, offset)
	if err != nil {
		return 0, err
	}

	sampleRate := mp3SampleRate(hdr.versionID, hdr.sampleIdx)
	samplesPerFrame := 1152
	if hdr.versionID != 3 {
		samplesPerFrame = 576
	}

	if frames, ok := readXingFrameCount(r, frameStart, hdr); ok {
		totalSamples := int64(frames) * int64(samplesPerFrame)
		return time.Duration(float64(totalSamples) / float64(sampleRate) * float64(time.Second)), nil
	}

	bitrateBps := mp3Bitrate(hdr.versionID, hdr.bitrateIdx) * 1000
	if bitrateBps == 0 {
		return 0, fmt.Errorf("mp3: invalid bitrate")
	}
	remaining := size - frameStart
	return time.Duration(float64(remaining) * 8 / float64(bitrateBps) * float64(time.Second)), nil
}

// findMP3Frame scans a bounded window for the first valid MP3 frame sync.
func findMP3Frame(r io.ReadSeeker, offset int64) (int64, mp3Header, error) {
	if _, err := r.Seek(offset, io.SeekStart); err != nil {
		return 0, mp3Header{}, err
	}
	const searchWindow = 64 * 1024
	buf := make([]byte, searchWindow)
	n, err := io.ReadFull(r, buf)
	if err != nil && err != io.ErrUnexpectedEOF && err != io.EOF {
		return 0, mp3Header{}, err
	}
	buf = buf[:n]

	for i := 0; i+4 <= len(buf); i++ {
		if buf[i] != 0xFF {
			continue
		}
		hdr, ok := parseMP3FrameHeader([4]byte{buf[i], buf[i+1], buf[i+2], buf[i+3]})
		if !ok {
			continue
		}
		return offset + int64(i), hdr, nil
	}
	return 0, mp3Header{}, fmt.Errorf("mp3: no valid frame header found")
}

// readXingFrameCount looks for a Xing/Info VBR header immediately after the
// side info of the first frame, and returns its declared frame count.
func readXingFrameCount(r io.ReadSeeker, frameStart int64, hdr mp3Header) (uint32, bool) {
	sideInfo := 32
	if hdr.versionID != 3 {
		sideInfo = 17
	}
	if hdr.channelMode == 3 { // mono
		if hdr.versionID == 3 {
			sideInfo = 17
		} else {
			sideInfo = 9
		}
	}
	crc := int64(0)
	if !hdr.protectionAbsent {
		crc = 2
	}

	xingOffset := frameStart + 4 + crc + int64(sideInfo)
	if _, err := r.Seek(xingOffset, io.SeekStart); err != nil {
		return 0, false
	}
	var tag [4]byte
	if _, err := io.ReadFull(r, tag[:]); err != nil {
		return 0, false
	}
	if string(tag[:]) != "Xing" && string(tag[:]) != "Info" {
		return 0, false
	}
	var flags uint32
	if err := binary.Read(r, binary.BigEndian, &flags); err != nil {
		return 0, false
	}
	if flags&0x1 == 0 {
		return 0, false
	}
	var frames uint32
	if err := binary.Read(r, binary.BigEndian, &frames); err != nil {
		return 0, false
	}
	return frames, true
}

// durationAAC iterates ADTS frames, each carrying a fixed 1024 samples, and
// sums them against the sample rate found in the first frame.
func durationAAC(r io.ReadSeeker, size int64) (time.Duration, error) {
	if _, err := r.Seek(0, io.SeekStart); err != nil {
		return 0, err
	}
	pos, err := skipID3v2(r)
	if err != nil {
		return 0, err
	}

	const samplesPerFrame = 1024
	var totalSamples int64
	var sampleRate int
	buf := make([]byte, 7)

	for pos+7 <= size {
		if _, err := r.Seek(pos, io.SeekStart); err != nil {
			break
		}
		if _, err := io.ReadFull(r, buf); err != nil {
			break
		}
		if buf[0] != 0xFF || buf[1]&0xF0 != 0xF0 {
			break
		}
		freqIdx := (buf[2] >> 2) & 0x0F
		rate, ok := adtsSampleRate(freqIdx)
		if !ok {
			break
		}
		if sampleRate == 0 {
			sampleRate = rate
		}
		frameLen := int64(buf[3]&0x03)<<11 | int64(buf[4])<<3 | int64(buf[5]>>5)
		if frameLen < 7 {
			break
		}
		totalSamples += samplesPerFrame
		pos += frameLen
	}

	if sampleRate == 0 || totalSamples == 0 {
		return 0, fmt.Errorf("aac: no valid ADTS frames found")
	}
	return time.Duration(float64(totalSamples) / float64(sampleRate) * float64(time.Second)), nil
}

var adtsSampleRateTable = [13]int{96000, 88200, 64000, 48000, 44100, 32000, 24000, 22050, 16000, 12000, 11025, 8000, 7350}

func adtsSampleRate(idx byte) (int, bool) {
	if int(idx) >= len(adtsSampleRateTable) {
		return 0, false
	}
	return adtsSampleRateTable[idx], true
}
