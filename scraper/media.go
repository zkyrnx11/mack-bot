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
}

// InstagramStories holds story media download URLs.
type InstagramStories struct {
	Username string   `json:"username"`
	Count    int      `json:"count"`
	Media    []MediaItem `json:"media"`
}

// InstagramStoriesGet returns story media URLs for a given Instagram username.
func InstagramStoriesGet(username string) (*InstagramStories, error) {
	var result InstagramStories
	return &result, run(&result, "instagram", "stories", username)
}
