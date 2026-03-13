import httpx
import asyncio
import concurrent.futures
from playwright.async_api import async_playwright
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

    return {
        "title": res_data.get("title"),
        "author": res_data.get("author"),
        "urls": [m.get("url") for m in medias if m.get("url")],
    }


def stories(username: str) -> dict:
    def run_sync():
        loop = asyncio.new_event_loop()
        try:
            return loop.run_until_complete(_execute_stories(username))
        finally:
            loop.close()

    async def _execute_stories(uname: str) -> dict:
        async with async_playwright() as p:
            browser = await p.chromium.launch(headless=True)
            context = await browser.new_context(
                user_agent="Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"
            )
            page = await context.new_page()
            captured_data = {"result": None}

            async def handle_response(response):
                if (
                    "api/downloader/stories/" in response.url
                    and response.request.method == "POST"
                ):
                    try:
                        captured_data["result"] = await response.json()
                    except Exception:
                        pass

            page.on("response", handle_response)
            await page.goto(
                "https://inflact.com/instagram-downloader/stories/",
                wait_until="domcontentloaded",
            )
            await page.fill('input[name="url"]', uname)
            await page.click('button[type="submit"]')

            import time as _time
            start = _time.time()
            while captured_data["result"] is None:
                if _time.time() - start > 30:
                    break
                await asyncio.sleep(0.5)

            await browser.close()
            return captured_data["result"]

    with concurrent.futures.ThreadPoolExecutor(max_workers=1) as pool:
        result = pool.submit(run_sync).result()

    if not result or result.get("status") != "success":
        raise RuntimeError("Failed to capture stories")

    stories_data = result.get("data", {})
    story_list = stories_data.get("stories") if isinstance(stories_data, dict) else []
    if not isinstance(story_list, list):
        story_list = []

    return {
        "username": username,
        "count": len(story_list),
        "media": [
            s.get("downloadUrl")
            for s in story_list
            if isinstance(s, dict) and s.get("downloadUrl")
        ],
    }
