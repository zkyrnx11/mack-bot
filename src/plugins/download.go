package plugins

import (
	"fmt"
	"strings"

	"github.com/zkyrnx11/mack/scraper"
)

func init() {
	Register(&Command{
		Pattern:  "yt",
		Aliases:  []string{"ytv", "video"},
		Category: "media",
		Func:     ytVideoCmd,
	})
	Register(&Command{
		Pattern:  "ytaudio",
		Aliases:  []string{"yta", "audio"},
		Category: "media",
		Func:     ytAudioCmd,
	})
	Register(&Command{
		Pattern:  "ytsearch",
		Aliases:  []string{"yts"},
		Category: "media",
		Func:     ytSearchCmd,
	})
	Register(&Command{
		Pattern:  "spotify",
		Aliases:  []string{"sp"},
		Category: "media",
		Func:     spotifyCmd,
	})
	Register(&Command{
		Pattern:  "tweet",
		Aliases:  []string{"tw"},
		Category: "media",
		Func:     tweetCmd,
	})
	Register(&Command{
		Pattern:  "reddit",
		Aliases:  []string{"rd"},
		Category: "media",
		Func:     redditCmd,
	})
	Register(&Command{
		Pattern:  "instagram",
		Aliases:  []string{"ig"},
		Category: "media",
		Func:     instagramCmd,
	})
}

func ytVideoCmd(ctx *Context) error {
	if ctx.Text == "" {
		ctx.Reply("Usage: .yt <url>")
		return nil
	}
	info, err := scraper.YouTubeVideo(ctx.Text)
	if err != nil {
		return fmt.Errorf("yt video: %w", err)
	}
	ctx.Reply(fmt.Sprintf("🎬 *%s*\n🔗 %s", info.Title, info.DownloadURL))
	return nil
}

func ytAudioCmd(ctx *Context) error {
	if ctx.Text == "" {
		ctx.Reply("Usage: .ytaudio <url>")
		return nil
	}
	info, err := scraper.YouTubeAudio(ctx.Text)
	if err != nil {
		return fmt.Errorf("yt audio: %w", err)
	}
	ctx.Reply(fmt.Sprintf("🎵 *%s*\n🔗 %s", info.Title, info.DownloadURL))
	return nil
}

func ytSearchCmd(ctx *Context) error {
	if ctx.Text == "" {
		ctx.Reply("Usage: .ytsearch <query>")
		return nil
	}
	results, err := scraper.YouTubeSearch(ctx.Text, 5)
	if err != nil {
		return fmt.Errorf("yt search: %w", err)
	}
	if len(results) == 0 {
		ctx.Reply("No results found.")
		return nil
	}
	var sb strings.Builder
	sb.WriteString("🔍 *YouTube Search Results*\n\n")
	for i, r := range results {
		sb.WriteString(fmt.Sprintf("%d. *%s* (%ds)\n%s\n\n", i+1, r.Title, r.Duration, r.URL))
	}
	ctx.Reply(sb.String())
	return nil
}

func spotifyCmd(ctx *Context) error {
	if ctx.Text == "" {
		ctx.Reply("Usage: .spotify <url>")
		return nil
	}
	track, err := scraper.SpotifyDownload(ctx.Text)
	if err != nil {
		return fmt.Errorf("spotify: %w", err)
	}
	ctx.Reply(fmt.Sprintf("🎶 *%s* — %s\n🎧 %s\n🔗 %s",
		track.SpotifyTitle, track.SpotifyArtist, track.YouTubeTitle, track.DownloadURL))
	return nil
}

func tweetCmd(ctx *Context) error {
	if ctx.Text == "" {
		ctx.Reply("Usage: .tweet <url>")
		return nil
	}
	result, err := scraper.TwitterDownload(ctx.Text)
	if err != nil {
		return fmt.Errorf("twitter: %w", err)
	}
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("🐦 *%s* by *%s*\n\n", result.Title, result.Author))
	for i, m := range result.Media {
		sb.WriteString(fmt.Sprintf("%d. [%s] %s\n", i+1, m.Type, m.URL))
	}
	ctx.Reply(sb.String())
	return nil
}

func redditCmd(ctx *Context) error {
	if ctx.Text == "" {
		ctx.Reply("Usage: .reddit <url>")
		return nil
	}
	result, err := scraper.RedditDownload(ctx.Text)
	if err != nil {
		return fmt.Errorf("reddit: %w", err)
	}
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("👾 *%s* by *%s*\n\n", result.Title, result.Author))
	for i, m := range result.Media {
		sb.WriteString(fmt.Sprintf("%d. [%s] %s\n", i+1, m.Type, m.URL))
	}
	ctx.Reply(sb.String())
	return nil
}

func instagramCmd(ctx *Context) error {
	if ctx.Text == "" {
		ctx.Reply("Usage: .instagram <url>")
		return nil
	}
	result, err := scraper.InstagramDownload(ctx.Text)
	if err != nil {
		return fmt.Errorf("instagram: %w", err)
	}
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("📸 *%s* by *%s*\n\n", result.Title, result.Author))
	for i, u := range result.URLs {
		sb.WriteString(fmt.Sprintf("%d. %s\n", i+1, u))
	}
	ctx.Reply(sb.String())
	return nil
}
