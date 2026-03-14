package plugins

import (
	"fmt"
	"strconv"
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
		Pattern:  "igstories",
		Aliases:  []string{"stories", "igstory"},
		Category: "media",
		Func:     igstoriesCmd,
	})
}

func ytVideoCmd(ctx *Context) error {
	if ctx.Text == "" {
		ctx.Reply("Usage: .yt <url>")
		return nil
	}
	_, stop := ctx.SendLoader()
	info, err := scraper.YouTubeVideo(ctx.Text)
	if err != nil {
		stop(fmt.Sprintf("❌ Error: %v", err))
		return fmt.Errorf("yt video: %w", err)
	}

	vidBytes, err := scraper.FetchMediaHTTP(info.DownloadURL)
	if err != nil {
		stop(fmt.Sprintf("❌ Failed to download video: %v", err))
		return fmt.Errorf("fetch video: %w", err)
	}

	caption := fmt.Sprintf("🎬 *%s*", info.Title)
	if err := ctx.SendVideo(vidBytes, "video/mp4", caption); err != nil {
		stop(fmt.Sprintf("❌ Failed to send video: %v", err))
		return fmt.Errorf("send video: %w", err)
	}

	stop("")
	return nil
}

func ytAudioCmd(ctx *Context) error {
	if ctx.Text == "" {
		ctx.Reply("Usage: .ytaudio <url>")
		return nil
	}
	_, stop := ctx.SendLoader()
	info, err := scraper.YouTubeAudio(ctx.Text)
	if err != nil {
		stop(fmt.Sprintf("❌ Error: %v", err))
		return fmt.Errorf("yt audio: %w", err)
	}

	audBytes, err := scraper.FetchMediaHTTP(info.DownloadURL)
	if err != nil {
		stop(fmt.Sprintf("❌ Failed to download audio: %v", err))
		return fmt.Errorf("fetch audio: %w", err)
	}

	if err := ctx.SendAudio(audBytes, "audio/mpeg"); err != nil {
		stop(fmt.Sprintf("❌ Failed to send audio: %v", err))
		return fmt.Errorf("send audio: %w", err)
	}

	stop(fmt.Sprintf("🎵 *%s*", info.Title))
	return nil
}

func ytSearchCmd(ctx *Context) error {
	if ctx.Text == "" {
		ctx.Reply("Usage: .ytsearch <query>")
		return nil
	}
	id, stop := ctx.SendLoader()
	results, err := scraper.YouTubeSearch(ctx.Text, 5)
	if err != nil {
		stop(fmt.Sprintf("❌ Error: %v", err))
		return fmt.Errorf("yt search: %w", err)
	}
	if len(results) == 0 {
		stop("No results found.")
		return nil
	}
	var sb strings.Builder
	sb.WriteString("🔍 *YouTube Search Results*\n\n")
	sb.WriteString("Reply to this message with a number to select:\n\n")
	for i, r := range results {
		sb.WriteString(fmt.Sprintf("%d. *%s* (%ds)\n%s\n\n", i+1, r.Title, r.Duration, r.URL))
	}
	stop(strings.TrimSpace(sb.String()))

	InteractiveSessions.Store(id, func(replyCtx *Context) {
		num, err := strconv.Atoi(strings.TrimSpace(replyCtx.Text))
		if err != nil || num < 1 || num > len(results) {
			replyCtx.Reply("❌ Invalid selection. Please reply with a valid number.")
			return
		}

		selected := results[num-1]

		menuMsg, _ := replyCtx.ReplySync(fmt.Sprintf("🎬 *%s*\n\nReply to this message to select format:\n1. Video\n2. Audio", selected.Title))

		if menuMsg.ID != "" {
			InteractiveSessions.Store(menuMsg.ID, func(formatCtx *Context) {
				choice := strings.TrimSpace(formatCtx.Text)
				switch choice {
				case "1":
					_, vidStop := formatCtx.SendLoader()
					info, err := scraper.YouTubeVideo(selected.URL)
					if err != nil {
						vidStop(fmt.Sprintf("❌ Error: %v", err))
						return
					}
					vidBytes, err := scraper.FetchMediaHTTP(info.DownloadURL)
					if err != nil {
						vidStop(fmt.Sprintf("❌ Failed to download video: %v", err))
						return
					}
					caption := fmt.Sprintf("🎬 *%s*", info.Title)
					if err := formatCtx.SendVideo(vidBytes, "video/mp4", caption); err != nil {
						vidStop(fmt.Sprintf("❌ Failed to send video: %v", err))
						return
					}
					vidStop("")
					InteractiveSessions.Delete(menuMsg.ID) // cleanup
				case "2":
					_, audStop := formatCtx.SendLoader()
					info, err := scraper.YouTubeAudio(selected.URL)
					if err != nil {
						audStop(fmt.Sprintf("❌ Error: %v", err))
						return
					}
					audBytes, err := scraper.FetchMediaHTTP(info.DownloadURL)
					if err != nil {
						audStop(fmt.Sprintf("❌ Failed to download audio: %v", err))
						return
					}
					if err := formatCtx.SendAudio(audBytes, "audio/mpeg"); err != nil {
						audStop(fmt.Sprintf("❌ Failed to send audio: %v", err))
						return
					}
					audStop(fmt.Sprintf("🎵 *%s*", info.Title))
					InteractiveSessions.Delete(menuMsg.ID) // cleanup
				default:
					formatCtx.Reply("❌ Invalid choice. Please reply with 1 for Video or 2 for Audio.")
				}
			})
		}
		InteractiveSessions.Delete(id) // Optional cleanup of the main search menu
	})

	return nil
}

func spotifyCmd(ctx *Context) error {
	if ctx.Text == "" {
		ctx.Reply("Usage: .spotify <url>")
		return nil
	}
	_, stop := ctx.SendLoader()
	track, err := scraper.SpotifyDownload(ctx.Text)
	if err != nil {
		stop(fmt.Sprintf("❌ Error: %v", err))
		return fmt.Errorf("spotify: %w", err)
	}

	spBytes, err := scraper.FetchMediaHTTP(track.DownloadURL)
	if err != nil {
		stop(fmt.Sprintf("❌ Failed to download track: %v", err))
		return fmt.Errorf("fetch spotify: %w", err)
	}

	if err := ctx.SendAudio(spBytes, "audio/mpeg"); err != nil {
		stop(fmt.Sprintf("❌ Failed to send track: %v", err))
		return fmt.Errorf("send spotify: %w", err)
	}

	stop(fmt.Sprintf("🎶 *%s* — %s", track.SpotifyTitle, track.SpotifyArtist))
	return nil
}

func igstoriesCmd(ctx *Context) error {
	if ctx.Text == "" {
		ctx.Reply("Usage: .igstories <username>")
		return nil
	}
	_, stop := ctx.SendLoader()
	result, err := scraper.InstagramStoriesGet(ctx.Text)
	if err != nil {
		stop(fmt.Sprintf("❌ Error: %v", err))
		return fmt.Errorf("instagram stories: %w", err)
	}

	validMedia := false
	for _, u := range result.Media {
		mediaBytes, err := scraper.FetchMediaHTTP(u)
		if err != nil {
			continue
		}
		if err := ctx.SendVideo(mediaBytes, "video/mp4", ""); err == nil {
			validMedia = true
		} else if err := ctx.SendImage(mediaBytes, "image/jpeg", ""); err == nil {
			validMedia = true
		}
	}
	if !validMedia && result.Count > 0 {
		stop("❌ Failed to send any media.")
		return nil
	} else if result.Count == 0 {
		stop("❌ No stories found.")
		return nil
	}
	stop(fmt.Sprintf("📸 *%d* stories found for *%s*", result.Count, result.Username))
	return nil
}
