package scraper

import "fmt"

// VideoResult holds metadata for a downloadable video or audio track.
type VideoResult struct {
	Title       string `json:"title"`
	DownloadURL string `json:"download_url"`
	Thumbnail   string `json:"thumbnail"`
	Resolution  string `json:"resolution"`
	OriginalURL string `json:"original_url"`
}

// SearchResult holds a single YouTube search entry.
type SearchResult struct {
	Title     string `json:"title"`
	URL       string `json:"url"`
	Duration  int    `json:"duration"`
	Thumbnail string `json:"thumbnail"`
}

// YouTubeVideo returns a direct video stream URL for the given YouTube URL.
func YouTubeVideo(url string) (*VideoResult, error) {
	var result VideoResult
	return &result, run(&result, "ytdl", "video", url)
}

// YouTubeAudio returns a direct audio stream URL for the given YouTube URL.
func YouTubeAudio(url string) (*VideoResult, error) {
	var result VideoResult
	return &result, run(&result, "ytdl", "audio", url)
}

// YouTubeSearch searches YouTube and returns up to limit results.
func YouTubeSearch(query string, limit int) ([]SearchResult, error) {
	var results []SearchResult
	return results, run(&results, "ytdl", "search", query, "--limit", fmt.Sprintf("%d", limit))
}

// YouTubeVideoSearchDownload searches YouTube and returns the first result's video download URL.
func YouTubeVideoSearchDownload(query string) (*VideoResult, error) {
	var result VideoResult
	return &result, run(&result, "ytdl", "search-download", query, "--format", "video")
}

// YouTubeAudioSearchDownload searches YouTube and returns the first result's audio download URL.
func YouTubeAudioSearchDownload(query string) (*VideoResult, error) {
	var result VideoResult
	return &result, run(&result, "ytdl", "search-download", query, "--format", "audio")
}
