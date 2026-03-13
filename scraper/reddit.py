import httpx
import asyncio
from .cookie import getCookie


def _downr_request(url: str) -> dict:
    cookie_content = asyncio.run(getCookie("https://downr.org/", netscape=False))

    headers = {
        "Content-Type": "application/json",
        "Referer": "https://downr.org/",
        "Origin": "https://downr.org/",
        "User-Agent": "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
    }
    if cookie_content:
        headers["Cookie"] = cookie_content

    with httpx.Client() as client:
        response = client.post(
            "https://downr.org/.netlify/functions/nyt",
            json={"url": url},
            headers=headers,
            timeout=15.0,
        )
        response.raise_for_status()
        return response.json()


def download(url: str) -> dict:
    res_data = _downr_request(url)
    medias = res_data.get("medias", [])
    if not medias:
        raise RuntimeError("No media found")

    results = []
    for media in medias:
        results.append({
            "url": media.get("url"),
            "type": media.get("type"),
            "quality": media.get("quality"),
        })

    return {
        "title": res_data.get("title"),
        "author": res_data.get("author"),
        "media": results,
    }
