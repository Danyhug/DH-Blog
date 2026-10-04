package media

import (
	"encoding/binary"
	"os"
	"path/filepath"
	"testing"

	"golang.org/x/text/encoding/simplifiedchinese"
)

func latin1(raw []byte) string {
	runes := make([]rune, len(raw))
	for i, b := range raw {
		runes[i] = rune(b)
	}
	return string(runes)
}

func gbk(t *testing.T, text string) []byte {
	t.Helper()
	encoded, err := simplifiedchinese.GBK.NewEncoder().Bytes([]byte(text))
	if err != nil {
		t.Fatalf("encode gbk: %v", err)
	}
	return encoded
}

func TestRepairText(t *testing.T) {
	cases := []struct {
		name  string
		input string
		want  string
	}{
		{"plain ascii", "Hello", "Hello"},
		{"gbk mojibake", latin1(gbk(t, "周杰伦")), "周杰伦"},
		{"utf8 mojibake", latin1([]byte("晴天")), "晴天"},
		{"genuine latin1", "Mötley Crüe", "Mötley Crüe"},
		{"already unicode", "陈奕迅", "陈奕迅"},
		{"nul padded", "Title\x00\x00", "Title"},
	}
	for _, tc := range cases {
		if got := repairText(tc.input); got != tc.want {
			t.Errorf("%s: repairText = %q, want %q", tc.name, got, tc.want)
		}
	}
}

// writeID3 builds a minimal ID3v2.3 tagged file with ISO-8859-1 text frames
// holding raw (here GBK) bytes, the way legacy Windows taggers wrote them.
func writeID3(t *testing.T, path string, frames map[string][]byte) {
	t.Helper()
	var body []byte
	for id, text := range frames {
		payload := append([]byte{0}, text...)
		header := make([]byte, 10)
		copy(header, id)
		binary.BigEndian.PutUint32(header[4:8], uint32(len(payload)))
		body = append(body, header...)
		body = append(body, payload...)
	}
	size := len(body)
	tagHeader := []byte{'I', 'D', '3', 3, 0, 0,
		byte(size >> 21 & 0x7F), byte(size >> 14 & 0x7F), byte(size >> 7 & 0x7F), byte(size & 0x7F)}
	data := append(tagHeader, body...)
	// A few bytes of fake audio so the file is not only a tag.
	data = append(data, 0xFF, 0xFB, 0x90, 0x00)
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}
}

func TestReadAudioTagsRepairsGBKFrames(t *testing.T) {
	path := filepath.Join(t.TempDir(), "song.mp3")
	writeID3(t, path, map[string][]byte{
		"TIT2": gbk(t, "晴天"),
		"TPE1": gbk(t, "周杰伦"),
		"TALB": gbk(t, "叶惠美"),
		"TRCK": []byte("3/11"),
	})
	tags, err := readAudioTags(path)
	if err != nil {
		t.Fatalf("readAudioTags: %v", err)
	}
	if tags.Title != "晴天" || tags.Artist != "周杰伦" || tags.Album != "叶惠美" || tags.TrackNo != 3 {
		t.Errorf("tags = %+v", tags)
	}
}

func TestReadAudioTagsWithoutTags(t *testing.T) {
	path := filepath.Join(t.TempDir(), "raw.mp3")
	// Longer than an ID3v1 footer, which the reader probes from the end.
	if err := os.WriteFile(path, make([]byte, 512), 0o644); err != nil {
		t.Fatal(err)
	}
	tags, err := readAudioTags(path)
	if err != nil {
		t.Fatalf("readAudioTags: %v", err)
	}
	if tags != (audioTags{}) {
		t.Errorf("tags = %+v, want empty", tags)
	}
}

func TestDecodeTextHandlesGB18030Lyrics(t *testing.T) {
	encoded := gbk(t, "[00:01.00]故事的小黄花")
	if got := decodeText(encoded); got != "[00:01.00]故事的小黄花" {
		t.Errorf("decodeText = %q", got)
	}
	if got := decodeText([]byte("\xEF\xBB\xBF[00:01.00]hi")); got != "[00:01.00]hi" {
		t.Errorf("decodeText with BOM = %q", got)
	}
}
