#!/bin/bash

# Check if whisper-cli is installed
if ! command -v whisper-cli &> /dev/null; then
    echo "whisper-cli not found. Installing whisper-cpp..."
    brew install whisper-cpp
fi
# Check if ffmpeg is installed
if ! command -v ffmpeg &> /dev/null; then
    echo "ffmpeg not found. Installing ffmpeg..."
    brew install ffmpeg
fi

# Pull latest version of transcriber
curl -L https://github.com/nnanto/transcriber/releases/download/latest/transcriber-darwin-amd64.tar.gz | tar -xz
sudo mv transcriber-* /usr/local/bin/transcriber
chmod +x /usr/local/bin/transcriber

# Verify installation
transcriber version

# Download model
transcriber download-model