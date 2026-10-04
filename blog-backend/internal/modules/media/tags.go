package media

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"strings"
	"unicode/utf8"

	"github.com/dhowden/tag"
	"golang.org/x/text/encoding/simplifiedchinese"
)

// audioTags is the subset of embedded tags the library shows.
type audioTags struct {
	Title       string
	Artist      string
	Album       string
	AlbumArtist string
	Genre       string
	Year        int
	TrackNo     int
	DiscNo      int
	HasPicture  bool
	HasLyrics   bool
}

func readMetadata(path string) (tag.Metadata, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer func() { _ = file.Close() }()
	return tag.ReadFrom(file)
}

// readAudioTags reads the embedded tags of an audio file. A file without any
// tag is not an error: the caller falls back to the file name.
func readAudioTags(path string) (audioTags, error) {
	metadata, err := readMetadata(path)
	if err == tag.ErrNoTagsFound {
		return audioTags{}, nil
	}
	if err != nil {
		return audioTags{}, err
	}
	trackNo, _ := metadata.Track()
	discNo, _ := metadata.Disc()
	return audioTags{
		Title:       repairText(metadata.Title()),
		Artist:      repairText(metadata.Artist()),
		Album:       repairText(metadata.Album()),
		AlbumArtist: repairText(metadata.AlbumArtist()),
		Genre:       repairText(metadata.Genre()),
		Year:        metadata.Year(),
		TrackNo:     trackNo,
		DiscNo:      discNo,
		HasPicture:  metadata.Picture() != nil && len(metadata.Picture().Data) > 0,
		HasLyrics:   strings.TrimSpace(metadata.Lyrics()) != "",
	}, nil
}

// readPicture returns the embedded cover art, or nil when there is none.
func readPicture(path string) (*tag.Picture, error) {
	metadata, err := readMetadata(path)
	if err == tag.ErrNoTagsFound {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	picture := metadata.Picture()
	if picture == nil || len(picture.Data) == 0 {
		return nil, nil
	}
	return picture, nil
}

// readEmbeddedLyrics returns the lyrics stored in the tags (USLT / ©lyr / LYRICS).
func readEmbeddedLyrics(path string) (string, error) {
	metadata, err := readMetadata(path)
	if err == tag.ErrNoTagsFound {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	return repairText(metadata.Lyrics()), nil
}

// maxLyricsBytes caps a sidecar .lrc read; real lyric files are a few KB.
const maxLyricsBytes = 1 << 20

func readLyricsFile(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer func() { _ = file.Close() }()
	data, err := io.ReadAll(io.LimitReader(file, maxLyricsBytes+1))
	if err != nil {
		return "", err
	}
	if len(data) > maxLyricsBytes {
		return "", fmt.Errorf("歌词文件过大")
	}
	return decodeText(data), nil
}

// decodeText decodes a text file that is either UTF-8 (with or without BOM)
// or GB18030 — the two encodings Chinese .lrc files come in.
func decodeText(data []byte) string {
	data = bytes.TrimPrefix(data, []byte("\xEF\xBB\xBF"))
	if utf8.Valid(data) {
		return string(data)
	}
	decoded, err := simplifiedchinese.GB18030.NewDecoder().Bytes(data)
	if err != nil {
		return string(data)
	}
	return string(decoded)
}

// repairText undoes the most common tag mojibake in Chinese libraries.
//
// ID3v2.3 text frames marked ISO-8859-1 very often actually hold GBK (old
// Windows taggers) or UTF-8 bytes; the tag library decodes them byte-by-byte
// into Latin-1 runes. When every rune still fits in a byte the original bytes
// can be recovered and re-decoded. GBK is only accepted when every high byte
// pairs up inside the GB2312 range, otherwise genuine Latin-1 text such as
// "Mötley Crüe" would be turned into random Han characters.
func repairText(value string) string {
	value = strings.TrimSpace(strings.Trim(value, "\x00"))
	if value == "" {
		return value
	}
	raw := make([]byte, 0, len(value))
	high := false
	for _, r := range value {
		if r > 0xFF {
			return value
		}
		if r >= 0x80 {
			high = true
		}
		raw = append(raw, byte(r))
	}
	if !high {
		return value
	}
	if utf8.Valid(raw) {
		return string(raw)
	}
	if !looksLikeGB2312(raw) {
		return value
	}
	decoded, err := simplifiedchinese.GBK.NewDecoder().Bytes(raw)
	if err != nil || !utf8.Valid(decoded) || bytes.ContainsRune(decoded, utf8.RuneError) {
		return value
	}
	return string(decoded)
}

func looksLikeGB2312(raw []byte) bool {
	for i := 0; i < len(raw); i++ {
		b := raw[i]
		if b < 0x80 {
			continue
		}
		if b < 0xA1 || b > 0xF7 || i+1 >= len(raw) {
			return false
		}
		next := raw[i+1]
		if next < 0xA1 || next > 0xFE {
			return false
		}
		i++
	}
	return true
}
