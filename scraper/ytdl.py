from typing import Any, cast
import yt_dlp
import tempfile
import os
from .cookie import getCookie
import asyncio


async def _get_ytdl_options(format_type: str) -> dict[str, Any]:
    cookie_content = await getCookie("https://youtube.com/", netscape=True)
    opts: dict[str, Any] = {
        "quiet": True,
        "noplaylist": True,
        "js_runtimes": {"node": {}},
    }

    if cookie_content:
        with tempfile.NamedTemporaryFile(
            mode="w", delete=False, suffix=".txt", encoding="utf-8"
        ) as f:
            f.write(cookie_content)
            opts["cookiefile"] = f.name

    if format_type == "audio":
        opts.update({
            "format": "bestaudio/best",
            "postprocessors": [{
                "key": "FFmpegExtractAudio",
                "preferredcodec": "mp3",
                "preferredquality": "128",
            }],
        })
    else:
        opts["format"] = "best[height<=480]/bestvideo[height<=480]+bestaudio/best"

    return opts


def _get_best_url(info: dict[str, Any]) -> str | None:
    return info.get("url") or (info.get("formats", [{}])[-1].get("url"))


def _cleanup(opts: dict[str, Any]) -> None:
    if "cookiefile" in opts and os.path.exists(opts["cookiefile"]):
        os.remove(opts["cookiefile"])


def search(query: str, limit: int = 5) -> list[dict]:
    ydl_opts = asyncio.run(_get_ytdl_options("video"))
    try:
        with yt_dlp.YoutubeDL(cast(Any, ydl_opts)) as ydl:
            results = cast(
                dict[str, Any],
                ydl.extract_info(f"ytsearch{limit}:{query}", download=False),
            )
            entries = results.get("entries", []) if results else []
            return [
                {
                    "title": e.get("title"),
                    "url": e.get("webpage_url"),
                    "duration": e.get("duration"),
                    "thumbnail": e.get("thumbnail"),
                }
                for e in entries
            ]
    finally:
        _cleanup(ydl_opts)


def download_video(url: str) -> dict:
    ydl_opts = asyncio.run(_get_ytdl_options("video"))
    try:
        with yt_dlp.YoutubeDL(cast(Any, ydl_opts)) as ydl:
            info = cast(dict[str, Any], ydl.extract_info(url, download=False))
            return {
                "title": info.get("title"),
                "download_url": _get_best_url(info),
                "thumbnail": info.get("thumbnail"),
                "resolution": info.get("resolution"),
            }
    finally:
        _cleanup(ydl_opts)


def download_audio(url: str) -> dict:
    ydl_opts = asyncio.run(_get_ytdl_options("audio"))
    try:
        with yt_dlp.YoutubeDL(cast(Any, ydl_opts)) as ydl:
            info = cast(dict[str, Any], ydl.extract_info(url, download=False))
            return {
                "title": info.get("title"),
                "download_url": _get_best_url(info),
                "thumbnail": info.get("thumbnail"),
            }
    finally:
        _cleanup(ydl_opts)


def search_and_download(query: str, format_type: str = "video") -> dict:
    ydl_opts = asyncio.run(_get_ytdl_options(format_type))
    try:
        with yt_dlp.YoutubeDL(cast(Any, ydl_opts)) as ydl:
            results = cast(
                dict[str, Any],
                ydl.extract_info(f"ytsearch1:{query}", download=False),
            )
            entries = results.get("entries", [])
            if not entries:
                raise RuntimeError("No results found")
            first = entries[0]
            return {
                "title": first.get("title"),
                "download_url": _get_best_url(first),
                "original_url": first.get("webpage_url"),
                "thumbnail": first.get("thumbnail"),
            }
    finally:
        _cleanup(ydl_opts)
