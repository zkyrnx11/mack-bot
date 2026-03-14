import httpx
import asyncio
import json
import sys
from .cookie import getCookie

def _downr_request(url: str) -> dict:
    cookie_content = asyncio.run(getCookie("https://downr.org/", netscape=False))

    headers = {
        "Content-Type": "application/json",
        "Accept": "application/json, text/plain, */*",
        "User-Agent": "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/122.0.0.0 Safari/537.36",
        "Origin": "https://downr.org/",
        "Referer": "https://downr.org/",
        "Sec-Fetch-Dest": "empty",
        "Sec-Fetch-Mode": "cors",
        "Sec-Fetch-Site": "same-origin",
        "sec-ch-ua": '"Chromium";v="122", "Not(A:Brand";v="24", "Google Chrome";v="122"',
        "sec-ch-ua-mobile": "?0",
        "sec-ch-ua-platform": '"Windows"',
    }
    
    if cookie_content:
        headers["Cookie"] = cookie_content

    with httpx.Client(http2=True) as client: # Use http2 to match modern browsers
        response = client.post(
            "https://downr.org/.netlify/functions/nyt",
            json={"url": url},
            headers=headers,
            timeout=15.0,
        )
        
        # If still getting 403, the cookie from cookie.py is likely invalid/flagged
        if response.status_code == 403:
            print(f"[DEBUG] Body: {response.text}", file=sys.stderr)
            
        response.raise_for_status()
        return response.json()
def download(url: str) -> dict:
    res_data = _downr_request(url)
    medias = res_data.get("medias", [])
    if not medias:
        raise RuntimeError("No media found")

    results = []
    for media in medias:
        formats = media.get("formats", [])
        mp4_formats = [f for f in formats if f.get("container") == "mp4"]

        if not mp4_formats:
            download_url = media.get("url")
        else:
            best_format = max(mp4_formats, key=lambda x: x.get("bitrate", 0))
            download_url = best_format.get("url")

        results.append({
            "url": download_url,
            "type": media.get("type"),
            "thumbnail": media.get("thumbnail"),
        })

    return {
        "title": res_data.get("title"),
        "author": res_data.get("author"),
        "media": results,
    }