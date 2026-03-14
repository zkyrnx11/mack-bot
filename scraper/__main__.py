"""
Mack-Bot scraper CLI — outputs JSON to stdout, errors to stderr with exit code 1.

Usage:
  python -m scraper ytdl video <url>
  python -m scraper ytdl audio <url>
  python -m scraper ytdl search <query> [--limit N]
  python -m scraper ytdl search-download <query>
  python -m scraper spotify search <url>
  python -m scraper spotify download <url>
  python -m scraper instagram stories <username>
  python -m scraper cookies <url> [--netscape]
"""

import argparse
import asyncio
import json
import sys


def _out(data) -> None:
    print(json.dumps(data, ensure_ascii=False))


def _err(msg: str) -> None:
    print(json.dumps({"error": str(msg)}), file=sys.stderr)
    sys.exit(1)


# ── ytdl ──────────────────────────────────────────────────────────────────────

def cmd_ytdl(args):
    from . import ytdl
    try:
        if args.ytdl_cmd == "video":
            _out(ytdl.download_video(args.url))
        elif args.ytdl_cmd == "audio":
            _out(ytdl.download_audio(args.url))
        elif args.ytdl_cmd == "search":
            _out(ytdl.search(args.query, limit=args.limit))
        elif args.ytdl_cmd == "search-download":
            _out(ytdl.search_and_download(args.query, format_type=args.format))
        else:
            _err(f"Unknown ytdl subcommand: {args.ytdl_cmd}")
    except Exception as e:
        _err(e)


# ── spotify ───────────────────────────────────────────────────────────────────

def cmd_spotify(args):
    from . import spotify
    try:
        if args.spotify_cmd == "search":
            _out(spotify.search(args.url))
        elif args.spotify_cmd == "download":
            _out(spotify.download(args.url))
        else:
            _err(f"Unknown spotify subcommand: {args.spotify_cmd}")
    except Exception as e:
        _err(e)


# ── instagram ─────────────────────────────────────────────────────────────────

def cmd_instagram(args):
    from . import instagram
    try:
        if args.instagram_cmd == "stories":
            _out(instagram.stories(args.username))
        else:
            _err(f"Unknown instagram subcommand: {args.instagram_cmd}")
    except Exception as e:
        _err(e)


# ── cookies ───────────────────────────────────────────────────────────────────

def cmd_cookies(args):
    from .cookie import getCookie
    try:
        result = asyncio.run(getCookie(args.url, netscape=args.netscape))
        _out({"url": args.url, "cookies": result})
    except Exception as e:
        _err(e)


# ── parser ────────────────────────────────────────────────────────────────────

def build_parser() -> argparse.ArgumentParser:
    parser = argparse.ArgumentParser(
        prog="python -m scraper",
        description="Mack-Bot media scraper CLI",
    )
    sub = parser.add_subparsers(dest="cmd", required=True)

    # ytdl
    ytdl_p = sub.add_parser("ytdl", help="YouTube download commands")
    ytdl_sub = ytdl_p.add_subparsers(dest="ytdl_cmd", required=True)

    yt_video = ytdl_sub.add_parser("video", help="Get video download URL")
    yt_video.add_argument("url")

    yt_audio = ytdl_sub.add_parser("audio", help="Get audio download URL")
    yt_audio.add_argument("url")

    yt_search = ytdl_sub.add_parser("search", help="Search YouTube")
    yt_search.add_argument("query")
    yt_search.add_argument("--limit", type=int, default=5)

    yt_sad = ytdl_sub.add_parser("search-download", help="Search and return first result URL")
    yt_sad.add_argument("query")
    yt_sad.add_argument("--format", default="video", choices=["video", "audio"])

    # spotify
    spotify_p = sub.add_parser("spotify", help="Spotify commands")
    spotify_sub = spotify_p.add_subparsers(dest="spotify_cmd", required=True)

    sp_search = spotify_sub.add_parser("search", help="Get Spotify track metadata")
    sp_search.add_argument("url")

    sp_dl = spotify_sub.add_parser("download", help="Get Spotify track + YouTube audio URL")
    sp_dl.add_argument("url")

    # instagram
    ig_p = sub.add_parser("instagram", help="Instagram commands")
    ig_sub = ig_p.add_subparsers(dest="instagram_cmd", required=True)

    ig_stories = ig_sub.add_parser("stories", help="Download Instagram stories")
    ig_stories.add_argument("username")

    # cookies
    ck_p = sub.add_parser("cookies", help="Fetch cookies from a URL via headless browser")
    ck_p.add_argument("url")
    ck_p.add_argument("--netscape", action="store_true", help="Output in Netscape cookie format")

    return parser


def main():
    parser = build_parser()
    args = parser.parse_args()

    dispatch = {
        "ytdl": cmd_ytdl,
        "spotify": cmd_spotify,
        "instagram": cmd_instagram,
        "cookies": cmd_cookies,
    }
    dispatch[args.cmd](args)


if __name__ == "__main__":
    main()
