package media

import (
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

var (
	audioExts = setOf(".mp3", ".flac", ".m4a", ".aac", ".ogg", ".oga", ".opus", ".wav", ".weba", ".alac", ".aiff", ".wma", ".m4b", ".ape", ".dsf")
	// Containers the browser may not decode (mkv/avi/…) are still listed: Chrome
	// and Edge play many of them, and the player reports a clear error otherwise.
	videoExts    = setOf(".mp4", ".m4v", ".webm", ".mov", ".mkv", ".ogv", ".avi", ".wmv", ".flv", ".ts", ".m2ts", ".mpg", ".mpeg", ".3gp")
	imageExts    = setOf(".jpg", ".jpeg", ".png", ".webp", ".avif", ".gif", ".bmp")
	subtitleExts = setOf(".vtt", ".srt")
)

func setOf(values ...string) map[string]bool {
	set := make(map[string]bool, len(values))
	for _, value := range values {
		set[value] = true
	}
	return set
}

func extOf(name string) string { return strings.ToLower(filepath.Ext(name)) }

func stem(name string) string { return strings.TrimSuffix(name, filepath.Ext(name)) }

// kindOf classifies a file by extension; "" means not a media file.
func kindOf(name string) string {
	ext := extOf(name)
	switch {
	case audioExts[ext]:
		return kindAudio
	case videoExts[ext]:
		return kindVideo
	}
	return ""
}

var (
	bracketPattern = regexp.MustCompile(`\[[^\]]*\]|【[^】]*】|\{[^}]*\}`)
	// Everything from the first release-quality token on is noise
	// ("Movie.2019.1080p.BluRay.x264-GROUP" → "Movie.2019").
	qualityPattern = regexp.MustCompile(`(?i)(^|[\s.\-_(])(2160p|1080[pi]|720p|576p|480p|4k|uhd|hdr10?|dv|x26[45]|h\.?26[45]|hevc|avc|aac|ac3|e-?ac3|dts|ddp?5\.1|web-?dl|web-?rip|webrip|blu-?ray|bdrip|brrip|dvdrip|hdtv|hdrip|remux|10bit|8bit|proper|repack|中英字幕|中字|国语|粤语|双语)([\s.\-_)]|$)`)
	yearPattern    = regexp.MustCompile(`(^|[\s.(])((?:19|20)\d{2})([\s.)]|$)`)
	seasonEpisode  = regexp.MustCompile(`(?i)\bS(\d{1,2})[\s.]?E(\d{1,3})\b`)
	crossEpisode   = regexp.MustCompile(`\b(\d{1,2})x(\d{2,3})\b`)
	chineseEpisode = regexp.MustCompile(`第\s*(\d{1,4})\s*[集话話期]`)
	chineseSeason  = regexp.MustCompile(`第\s*(\d{1,2})\s*季`)
	plainEpisode   = regexp.MustCompile(`(?i)(?:^|[\s.\-_\[(])(?:EP?|Episode)\s?(\d{1,3})(?:$|[\s.\-_\])])`)
	leadingNumber  = regexp.MustCompile(`^(\d{1,3})(?:[\s.\-_]+|$)`)
	spaces         = regexp.MustCompile(`\s+`)
)

// videoName is what the library can tell about a video from its file name.
type videoName struct {
	Title   string
	Year    int
	Season  int
	Episode int
}

// parseVideoName turns a release-style file name into a display title plus
// whatever season / episode / year markers it carries.
func parseVideoName(name string) videoName {
	base := stem(name)
	result := videoName{}

	if match := seasonEpisode.FindStringSubmatch(base); match != nil {
		result.Season, _ = strconv.Atoi(match[1])
		result.Episode, _ = strconv.Atoi(match[2])
	} else if match := crossEpisode.FindStringSubmatch(base); match != nil {
		result.Season, _ = strconv.Atoi(match[1])
		result.Episode, _ = strconv.Atoi(match[2])
	} else {
		if match := chineseEpisode.FindStringSubmatch(base); match != nil {
			result.Episode, _ = strconv.Atoi(match[1])
		} else if match := plainEpisode.FindStringSubmatch(base); match != nil {
			result.Episode, _ = strconv.Atoi(match[1])
		}
		if match := chineseSeason.FindStringSubmatch(base); match != nil {
			result.Season, _ = strconv.Atoi(match[1])
		}
	}

	title := bracketPattern.ReplaceAllString(base, " ")
	// Dots and underscores are word separators in release names; keep them
	// when the name has real spaces, where a dot is more likely punctuation.
	if !strings.Contains(strings.TrimSpace(title), " ") {
		title = strings.NewReplacer(".", " ", "_", " ").Replace(title)
	} else {
		title = strings.ReplaceAll(title, "_", " ")
	}
	if loc := qualityPattern.FindStringIndex(title); loc != nil && loc[0] > 0 {
		title = title[:loc[0]]
	}
	if match := yearPattern.FindStringSubmatchIndex(title); match != nil && match[4] > 0 {
		result.Year, _ = strconv.Atoi(title[match[4]:match[5]])
		title = title[:match[4]]
	}
	title = strings.Trim(spaces.ReplaceAllString(title, " "), " -._()")
	if title == "" {
		title = strings.TrimSpace(base)
	}
	result.Title = title
	return result
}

// trackName is the fallback for audio files without usable tags: the common
// "01 - Artist - Title" / "Artist - Title" / "Title" naming schemes.
type trackName struct {
	TrackNo int
	Artist  string
	Title   string
}

func parseTrackName(name string) trackName {
	base := strings.TrimSpace(stem(name))
	result := trackName{}
	if match := leadingNumber.FindStringSubmatch(base); match != nil && len(match[0]) < len(base) {
		result.TrackNo, _ = strconv.Atoi(match[1])
		base = strings.TrimSpace(base[len(match[0]):])
	}
	if artist, title, ok := strings.Cut(base, " - "); ok && strings.TrimSpace(artist) != "" && strings.TrimSpace(title) != "" {
		result.Artist = strings.TrimSpace(artist)
		result.Title = strings.TrimSpace(title)
		return result
	}
	result.Title = base
	return result
}

// Sidecar names follow the conventions Plex / Jellyfin / Kodi users already
// keep on disk, so an existing library lights up without any renaming.
var (
	folderPosterNames   = []string{"poster", "folder", "cover", "movie", "show"}
	folderBackdropNames = []string{"fanart", "backdrop", "background", "landscape"}
	albumCoverNames     = []string{"cover", "folder", "front", "album", "albumart", "albumartsmall"}
	posterSuffixes      = []string{"", "-poster", ".poster", "-thumb", "-cover"}
	backdropSuffixes    = []string{"-fanart", ".fanart", "-backdrop", "-landscape"}
)

// findImage returns the first sibling image whose stem (case-insensitive)
// matches one of the wanted names, in the order the names are given.
func findImage(siblings []*Entry, wanted []string) string {
	byStem := make(map[string]string, len(siblings))
	for _, sibling := range siblings {
		if !imageExts[extOf(sibling.Name)] {
			continue
		}
		key := strings.ToLower(stem(sibling.Name))
		if _, exists := byStem[key]; !exists {
			byStem[key] = sibling.ID
		}
	}
	for _, name := range wanted {
		if id, ok := byStem[strings.ToLower(name)]; ok {
			return id
		}
	}
	return ""
}

func withSuffixes(base string, suffixes []string) []string {
	names := make([]string, len(suffixes))
	for i, suffix := range suffixes {
		names[i] = base + suffix
	}
	return names
}

// Subtitle is a sidecar subtitle file next to a video.
type Subtitle struct {
	ID     string `json:"id"`
	Label  string `json:"label"`
	Lang   string `json:"lang"`
	Format string `json:"format"`
}

var subtitleLabels = map[string]string{
	"zh": "中文", "chs": "简体中文", "sc": "简体中文", "zh-cn": "简体中文", "zh-hans": "简体中文", "gb": "简体中文",
	"cht": "繁體中文", "tc": "繁體中文", "zh-tw": "繁體中文", "zh-hant": "繁體中文", "big5": "繁體中文",
	"en": "English", "eng": "English", "ja": "日本語", "jp": "日本語", "jpn": "日本語", "ko": "한국어", "kor": "한국어",
	"chs&eng": "简英双语", "chs_eng": "简英双语", "zh-en": "中英双语",
}

// findSubtitles matches "<stem>.srt" and "<stem>.<lang>.srt" next to a video.
func findSubtitles(video *Entry, siblings []*Entry) []Subtitle {
	videoStem := strings.ToLower(stem(video.Name))
	subtitles := []Subtitle{}
	for _, sibling := range siblings {
		ext := extOf(sibling.Name)
		if !subtitleExts[ext] {
			continue
		}
		lower := strings.ToLower(stem(sibling.Name))
		if lower != videoStem && !strings.HasPrefix(lower, videoStem+".") {
			continue
		}
		lang := strings.TrimPrefix(lower[len(videoStem):], ".")
		label := subtitleLabels[lang]
		if label == "" {
			label = lang
		}
		if label == "" {
			label = "字幕"
		}
		subtitles = append(subtitles, Subtitle{ID: sibling.ID, Label: label, Lang: lang, Format: strings.TrimPrefix(ext, ".")})
	}
	return subtitles
}

// hasSidecarLyrics reports whether "<stem>.lrc" sits next to the track.
func hasSidecarLyrics(track *Entry, siblings []*Entry) bool {
	want := strings.ToLower(stem(track.Name)) + ".lrc"
	for _, sibling := range siblings {
		if strings.ToLower(sibling.Name) == want {
			return true
		}
	}
	return false
}
