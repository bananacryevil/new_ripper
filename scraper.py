import requests
import re
import time
import sys
import concurrent.futures

# Configuration
BASE_URL = "https://wlext.is/series/sin-senos-no-hay-paraiso-2008/?server=shorticu&episode={}"
START_EPISODE = 1
END_EPISODE = 167
OUTPUT_FILE = "episodes.txt"
BASH_SCRIPT = "download.sh"

def fetch_episode(i):
    episode_num = f"{i:03d}"
    url = BASE_URL.format(episode_num)
    key = "NULL"
    
    try:
        # print(f"Fetching episode {episode_num}...", end="", flush=True)
        response = requests.get(url, timeout=10)
        response.raise_for_status()
        
        # Extract key using regex
        match = re.search(r'src="https://short\.icu/([^"]+)"', response.text)
        
        if match:
            key = match.group(1)
            # print(f" Found key: {key}")
        else:
            # print(" Key NOT found")
            pass
            
    except Exception as e:
        print(f"Error fetching {episode_num}: {e}")
    
    return episode_num, key

def scrape_episodes():
    episodes = {}
    
    print(f"Scraping episodes {START_EPISODE} to {END_EPISODE} with 10 workers...")
    
    results = []
    with concurrent.futures.ThreadPoolExecutor(max_workers=10) as executor:
        future_to_episode = {executor.submit(fetch_episode, i): i for i in range(START_EPISODE, END_EPISODE + 1)}
        
        for future in concurrent.futures.as_completed(future_to_episode):
            episode_num, key = future.result()
            episodes[episode_num] = key
            print(f"Episode {episode_num}: {key}")

    # Sort episodes by number
    sorted_episodes = dict(sorted(episodes.items()))
    
    # Write to file
    with open(OUTPUT_FILE, "w") as f:
        for num, key in sorted_episodes.items():
            f.write(f"{num}:{key}\n")
            
    return sorted_episodes

def create_bash_script(episodes):
    print(f"Creating {BASH_SCRIPT}...")
    with open(BASH_SCRIPT, "w") as f:
        f.write("#!/bin/bash\n\n")
        f.write("# Create output directory if it doesn't exist\n")
        f.write("mkdir -p ./tv/SSNHP\n\n")
        
        for num, key in episodes.items():
            if key != "NULL":
                # java -jar abyss-dl.jar ${KEY} h -o ./tv/SSNHP/{$NUM}.mp4
                cmd = f"java -jar abyss-dl.jar {key} h -o ./tv/SSNHP/{num}.mp4"
                f.write(f"echo \"Downloading Episode {num}...\"\n")
                f.write(f"{cmd}\n")
            else:
                f.write(f"echo \"Skipping Episode {num} (Key not found)\"\n")

    # Make the script executable
    import os
    os.chmod(BASH_SCRIPT, 0o755)
    print("Done.")

if __name__ == "__main__":
    episodes = scrape_episodes()
    create_bash_script(episodes)
