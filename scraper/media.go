package scraper

// MediaItem represents a single downloadable media item.
type MediaItem struct {
	URL       string `json:"url"`
	Type      string `json:"type"`
	Thumbnail string `json:"thumbnail"`
	Quality   string `json:"quality"`
}

// MediaResult holds a collection of media items from a post.
type MediaResult struct {
	Title  string      `json:"title"`
	Author string      `json:"author"`
	Media  []MediaItem `json:"media"`
	URLs   []string    `json:"urls"` // used by Instagram download
}

// TwitterDownload returns video/media URLs for a Twitter/X post.
func TwitterDownload(url string) (*MediaResult, error) {
	var result MediaResult
	return &result, run(&result, "twitter", "download", url)
}

// RedditDownload returns video/media URLs for a Reddit post.
func RedditDownload(url string) (*MediaResult, error) {
	var result MediaResult
	return &result, run(&result, "reddit", "download", url)
}

// InstagramDownload returns media URLs for an Instagram post.
func InstagramDownload(url string) (*MediaResult, error) {
	var result MediaResult
	return &result, run(&result, "instagram", "download", url)
}

// InstagramStories holds story media download URLs.
type InstagramStories struct {
	Username string   `json:"username"`
	Count    int      `json:"count"`
	Media    []string `json:"media"`
}

// InstagramStoriesGet returns story media URLs for a given Instagram username.
func InstagramStoriesGet(username string) (*InstagramStories, error) {
	var result InstagramStories
	return &result, run(&result, "instagram", "stories", username)
}
