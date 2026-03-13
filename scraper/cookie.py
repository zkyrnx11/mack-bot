import asyncio
from playwright.async_api import async_playwright
import concurrent.futures
import time


_semaphore = asyncio.Semaphore(10)


def _run_in_proactor(url: str, netscape: bool) -> str:
    loop = asyncio.new_event_loop()
    try:
        return loop.run_until_complete(_fetch_cookies(url, netscape))
    finally:
        loop.close()


async def _fetch_cookies(url: str, netscape: bool) -> str:
    async with async_playwright() as p:
        browser = await p.chromium.launch(headless=True)
        context = await browser.new_context()
        page = await context.new_page()

        await page.goto(url, wait_until="domcontentloaded")
        cookies = await context.cookies()
        await browser.close()

        if netscape:
            lines = [
                "# Netscape HTTP Cookie File",
                "# http://curl.haxx.se/rfc/cookie_spec.html",
                "# This is a generated file! Do not edit.",
                "",
            ]
            for c in cookies:
                domain = c.get("domain", "")
                include_subdomains = "TRUE" if domain.startswith(".") else "FALSE"
                path = c.get("path", "/")
                secure = "TRUE" if c.get("secure") else "FALSE"

                raw_expires = c.get("expires")
                if raw_expires is None or raw_expires == -1:
                    expires = int(time.time() + 31536000)
                else:
                    expires = int(raw_expires)

                name = c.get("name", "")
                value = c.get("value", "")

                line = "\t".join([
                    domain,
                    include_subdomains,
                    path,
                    secure,
                    str(expires),
                    name,
                    value,
                ])
                lines.append(line)
            return "\n".join(lines)

        return "; ".join(
            f"{name}={value}"
            for c in cookies
            if (name := c.get("name")) and (value := c.get("value"))
        )


async def getCookie(url: str, netscape: bool = False) -> str:
    async with _semaphore:
        loop = asyncio.get_running_loop()
        with concurrent.futures.ThreadPoolExecutor(max_workers=1) as pool:
            return await loop.run_in_executor(pool, _run_in_proactor, url, netscape)
