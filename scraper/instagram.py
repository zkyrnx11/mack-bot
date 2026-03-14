import httpx
import asyncio
import concurrent.futures
from playwright.async_api import async_playwright
from .cookie import getCookie


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
            {
                "url": s.get("downloadUrl"),
                "type": "video" if s.get("type") == "video" else "photo"
            }
            for s in story_list
            if isinstance(s, dict) and s.get("downloadUrl")
        ],
    }
