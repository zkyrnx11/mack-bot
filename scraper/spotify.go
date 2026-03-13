package scraper

// SpotifyTrack holds Spotify track metadata from spotmate.online.
type SpotifyTrack struct {
	SpotifyTitle  string `json:"spotify_title"`
	SpotifyArtist string `json:"spotify_artist"`
	YouTubeTitle  string `json:"youtube_title"`
	DownloadURL   string `json:"download_url"`
	Thumbnail     string `json:"thumbnail"`
}

// SpotifySearch returns raw Spotify track metadata for the given Spotify URL.
func SpotifySearch(spotifyURL string) (map[string]any, error) {
	var result map[string]any
	return result, run(&result, "spotify", "search", spotifyURL)
}

// SpotifyDownload finds a Spotify track and returns a YouTube audio download URL.
func SpotifyDownload(spotifyURL string) (*SpotifyTrack, error) {
	var result SpotifyTrack
	return &result, run(&result, "spotify", "download", spotifyURL)
}
