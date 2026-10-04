package media

import "testing"

func TestParseVideoName(t *testing.T) {
	cases := []struct {
		name string
		want videoName
	}{
		{"Inception.2010.1080p.BluRay.x264-GROUP.mkv", videoName{Title: "Inception", Year: 2010}},
		{"The.Office.S02E05.720p.WEB-DL.mp4", videoName{Title: "The Office S02E05", Season: 2, Episode: 5}},
		{"[字幕组] 进击的巨人 第03集 [1080P].mp4", videoName{Title: "进击的巨人 第03集", Episode: 3}},
		{"我的电影 (2019).mp4", videoName{Title: "我的电影", Year: 2019}},
		{"Show 1x07.avi", videoName{Title: "Show 1x07", Season: 1, Episode: 7}},
		{"家庭录像.mov", videoName{Title: "家庭录像"}},
		{"EP12.mp4", videoName{Title: "EP12", Episode: 12}},
	}
	for _, tc := range cases {
		if got := parseVideoName(tc.name); got != tc.want {
			t.Errorf("parseVideoName(%q) = %+v, want %+v", tc.name, got, tc.want)
		}
	}
}

func TestParseTrackName(t *testing.T) {
	cases := []struct {
		name string
		want trackName
	}{
		{"01 - 周杰伦 - 晴天.mp3", trackName{TrackNo: 1, Artist: "周杰伦", Title: "晴天"}},
		{"03. Yesterday.flac", trackName{TrackNo: 3, Title: "Yesterday"}},
		{"Adele - Hello.m4a", trackName{Artist: "Adele", Title: "Hello"}},
		{"1979.mp3", trackName{Title: "1979"}},
	}
	for _, tc := range cases {
		if got := parseTrackName(tc.name); got != tc.want {
			t.Errorf("parseTrackName(%q) = %+v, want %+v", tc.name, got, tc.want)
		}
	}
}

func TestFindSubtitlesMatchesStemAndLanguage(t *testing.T) {
	video := &Entry{ID: "1", Name: "Movie.mp4"}
	siblings := []*Entry{
		video,
		{ID: "2", Name: "Movie.srt"},
		{ID: "3", Name: "movie.chs.vtt"},
		{ID: "4", Name: "Movie2.srt"},
		{ID: "5", Name: "Movie.ass"},
	}
	subtitles := findSubtitles(video, siblings)
	if len(subtitles) != 2 {
		t.Fatalf("subtitles = %+v, want 2", subtitles)
	}
	if subtitles[0].ID != "2" || subtitles[0].Label != "字幕" || subtitles[0].Format != "srt" {
		t.Errorf("first subtitle = %+v", subtitles[0])
	}
	if subtitles[1].ID != "3" || subtitles[1].Label != "简体中文" || subtitles[1].Format != "vtt" {
		t.Errorf("second subtitle = %+v", subtitles[1])
	}
}

func TestFindImagePrefersEarlierNames(t *testing.T) {
	siblings := []*Entry{
		{ID: "1", Name: "folder.JPG"},
		{ID: "2", Name: "Cover.png"},
		{ID: "3", Name: "cover.txt"},
	}
	if got := findImage(siblings, albumCoverNames); got != "2" {
		t.Errorf("findImage = %q, want cover.png", got)
	}
	if got := findImage(siblings, []string{"poster"}); got != "" {
		t.Errorf("findImage = %q, want none", got)
	}
}
