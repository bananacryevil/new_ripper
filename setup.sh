#!/bin/bash

# setup.sh - Install dependencies for new_ripper on a remote Linux resource (Debian/Ubuntu based)

set -e # Exit immediately if a command exits with a non-zero status

echo "[*] Updating package lists..."
sudo apt-get update

echo "[*] Installing Java (required for abyss-dl.jar)..."
if ! command -v java &> /dev/null; then
    sudo apt-get install -y default-jre
    echo "Java installed."
else
    echo "Java is already installed."
fi

echo "[*] Installing Go..."
if ! command -v go &> /dev/null; then
    sudo apt-get install -y golang-go
    echo "Go installed."
else
    echo "Go is already installed."
fi

echo "[*] Installing Chromium Browser (for automation)..."
# Try to install Chromium or Chrome. 
# Note: On Ubuntu, chromium-browser is often a Snap package.
if ! command -v chromium-browser &> /dev/null && ! command -v google-chrome &> /dev/null && ! command -v chromium &> /dev/null; then
    if sudo apt-get install -y chromium-browser; then
        echo "Chromium installed."
    elif sudo apt-get install -y chromium; then
        echo "Chromium installed."
    else
        echo "[!] Could not install Chromium via apt. Attempting to fetch Google Chrome .deb..."
        wget https://dl.google.com/linux/direct/google-chrome-stable_current_amd64.deb
        sudo apt install -y ./google-chrome-stable_current_amd64.deb
        rm google-chrome-stable_current_amd64.deb
        echo "Google Chrome installed."
    fi
else
    echo "A Chromium-based browser is already installed."
fi

echo "[*] Setting up Go dependencies..."
# Ensure we are in the project directory
if [ ! -f "go.mod" ]; then
    echo "Initializing go.mod..."
    go mod init new_ripper
fi

echo "Downloading Go modules..."
go get github.com/chromedp/chromedp
go mod tidy

echo "----------------------------------------------------------------"
echo "[*] Setup complete!"
echo "You can now run the downloader with:"
echo "   go run downloader_v3.go"
echo "----------------------------------------------------------------"
