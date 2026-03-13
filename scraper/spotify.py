import httpx
import urllib.parse
import asyncio
from .cookie import getCookie
from .ytdl import _get_ytdl_options, _get_best_url, _cleanup
import yt_dlp
from typing import Any, cast


def _get_spotify_data(spotify_url: str) -> dict:
    cookie_content = asyncio.run(getCookie("https://spotmate.online/en1", netscape=False))

    headers = {
        "Content-Type": "application/json",
        "Accept": "application/json",
        "Referer": "https://spotmate.online/en1",
        "Origin": "https://spotmate.online",
        "User-Agent": "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
    }

    if cookie_content:
        headers["Cookie"] = cookie_content
        try:
            parts = cookie_content.split("XSRF-TOKEN=")
            if len(parts) > 1:
                token = parts[1].split(";")[0]
                headers["X-XSRF-TOKEN"] = urllib.parse.unquote(token)
        except Exception:
            pass

    with httpx.Client() as client:
        response = client.post(
            "https://spotmate.online/getTrackData",
            json={"spotify_url": spotify_url},
            headers=headers,
            timeout=15.0,
        )
        response.raise_for_status()
        return response.json()


def search(spotify_url: str) -> dict:
    return _get_spotify_data(spotify_url)


def download(spotify_url: str) -> dict:
    spotify_data = _get_spotify_data(spotify_url)
    title = spotify_data.get("name")
    artist = spotify_data.get("artists", [{}])[0].get("name", "")
    search_query = f"{title} {artist} official audio"

    ydl_opts = asyncio.run(_get_ytdl_options("audio"))
    try:
        with yt_dlp.YoutubeDL(cast(Any, ydl_opts)) as ydl:
            results = cast(
                dict[str, Any],
                ydl.extract_info(f"ytsearch1:{search_query}", download=False),
            )
            entries = results.get("entries", [])
            if not entries:
                raise RuntimeError("No matching audio found on YouTube")
            first = entries[0]
            return {
                "spotify_title": title,
                "spotify_artist": artist,
                "youtube_title": first.get("title"),
                "download_url": _get_best_url(first),
                "thumbnail": spotify_data.get("album", {}).get("images", [{}])[0].get("url"),
            }
    finally:
        _cleanup(ydl_opts)
