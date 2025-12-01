import asyncio
import os
import subprocess
from playwright.async_api import async_playwright

# Configuration
EPISODES_FILE = "episodes.txt"
OUTPUT_DIR = "./tv/SSNHP"
CONCURRENCY_LIMIT = 3  # Lower concurrency for browser automation

async def download_episode(sem, browser, num, key):
    async with sem:
        page = await browser.new_page()
        try:
            url = f"https://wlext.is/series/sin-senos-no-hay-paraiso-2008/?server=shorticu&episode={num}"
            print(f"[{num}] Opening URL: {url}")
            
            # Go to the URL and wait for network idle to ensure scripts loaded
            await page.goto(url, timeout=60000)
            try:
                await page.wait_for_load_state("networkidle", timeout=30000)
            except:
                print(f"[{num}] Warning: Network idle timeout, proceeding anyway...")

            # Construct output path
            output_path = os.path.join(OUTPUT_DIR, f"{num}.mp4")
            
            # Run the java command
            cmd = [
                "java", "-jar", "abyss-dl.jar",
                key, "h", "-o", output_path
            ]
            
            print(f"[{num}] Starting download with key: {key}")
            
            # Run the subprocess
            process = await asyncio.create_subprocess_exec(
                *cmd,
                stdout=asyncio.subprocess.PIPE,
                stderr=asyncio.subprocess.PIPE
            )
            
            stdout, stderr = await process.communicate()
            
            if process.returncode == 0:
                print(f"[{num}] Download completed successfully.")
            else:
                print(f"[{num}] Download failed. Error:\n{stderr.decode()}")

        except Exception as e:
            print(f"[{num}] Error: {e}")
        finally:
            await page.close()

async def main():
    # Ensure output directory exists
    os.makedirs(OUTPUT_DIR, exist_ok=True)

    # Read episodes
    episodes = []
    with open(EPISODES_FILE, "r") as f:
        lines = f.readlines()
        # Skip header
        for line in lines[1:]:
            line = line.strip()
            if not line:
                continue
            parts = line.split(":")
            if len(parts) == 2:
                episodes.append((parts[0], parts[1]))

    async with async_playwright() as p:
        # Launch browser
        browser = await p.chromium.launch(headless=True) # Try headless first
        
        sem = asyncio.Semaphore(CONCURRENCY_LIMIT)
        tasks = []
        
        for num, key in episodes:
            task = asyncio.create_task(download_episode(sem, browser, num, key))
            tasks.append(task)
        
        await asyncio.gather(*tasks)
        
        await browser.close()

if __name__ == "__main__":
    asyncio.run(main())
