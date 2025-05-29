# 🎙️ Transcriber

> A powerful Go-based audio transcription tool powered by OpenAI Whisper

[![Go Version](https://img.shields.io/badge/Go-1.19+-blue.svg)](https://golang.org)
[![License](https://img.shields.io/badge/License-MIT-green.svg)](LICENSE)
[![Platform](https://img.shields.io/badge/Platform-Linux%20%7C%20macOS%20%7C%20Windows-lightgrey.svg)]()

## ✨ Features

- 🎯 **Real-time transcription** - Record and transcribe audio on the fly
- 🚀 **Cross-platform** - Works on Linux, macOS, and Windows
- ⚙️ **Configurable** - Flexible configuration options
- 🔧 **Multiple models** - Support for various Whisper model sizes
- 💻 **CLI-friendly** - Easy-to-use command-line interface

## 🚀 Quick Start

### Prerequisites

- Go 1.19 or later
- Audio recording capabilities (microphone)
- At least 4GB RAM (recommended for larger models)

### Installation

#### Option 1: Download pre-built binary (Recommended)
Download the latest release for your platform from [GitHub Releases](https://github.com/nnanto/transcriber/releases):

**Linux/macOS:**
```bash
# Download and install (replace VERSION with latest version, e.g., v1.0.0)
curl -L https://github.com/nnanto/transcriber/releases/download/VERSION/transcriber-linux-amd64.tar.gz | tar -xz
sudo mv transcriber /usr/local/bin/

# Or for macOS
curl -L https://github.com/nnanto/transcriber/releases/download/VERSION/transcriber-darwin-amd64.tar.gz | tar -xz
sudo mv transcriber /usr/local/bin/

# Make executable
chmod +x /usr/local/bin/transcriber
```

**Windows:**
1. Download `transcriber-windows-amd64.zip` from [releases page](https://github.com/nnanto/transcriber/releases)
2. Extract the ZIP file
3. Add the extracted folder to your PATH or move `transcriber.exe` to a folder in your PATH

**Verify installation:**
```bash
transcriber version
```

#### Option 2: Install from source
```bash
git clone https://github.com/nnanto/transcriber.git
cd transcriber
make install
```

#### Option 3: Build locally
```bash
git clone https://github.com/nnanto/transcriber.git
cd transcriber
make build
```

### First Run

1. **Download a Whisper model** (required on first use):
```bash
./transcriber download [--model ggml-base]
```

Available models:
- `ggml-tiny`
- `ggml-base`
- `ggml-small`
- `ggml-medium`
- `ggml-large`
- `ggml-large-v3-turbo-q5_0` (recommended for best performance)

You can also specify a custom model path in the configuration file.


2. **Start transcribing**:
```bash
./transcriber run --output ./transcriptions
```

3. **Check your configuration**:
```bash
./transcriber config
```

## 📖 Usage Guide

### Commands Overview

| Command | Description | Example |
|---------|-------------|---------|
| `run` | Record and transcribe in real-time | `transcriber run --duration 2m` |
| `process` | Process existing audio files | `transcriber process --input ./audio` |
| `config` | Show current configuration | `transcriber config` |
| `download` | Download Whisper models | `transcriber download --model large` |
| `stop` | Stop all running processes | `transcriber stop` |
| `version` | Show version info | `transcriber version` |

### Real-time Transcription

Start recording and transcribing immediately:

```bash
# Record for 30 minutes (default)
transcriber run

# Record for specific duration
transcriber run --duration 5m --output ./my-transcriptions

# Custom configuration
transcriber run --config ./custom-config --duration 1h
```

### Batch Processing

Process existing MP3 files:

```bash
# Process all MP3s in a directory
transcriber process --input ./recordings --output ./transcriptions

# With custom config
transcriber process --input ./audio --output ./text --config ./config
```

### Model Management

Download and manage Whisper models:

```bash
# Download specific model
transcriber download --model ggml-large-v3-turbo-q5_0

# Available models (by size/quality):
transcriber download --model ggml-tiny      # Fastest, least accurate
transcriber download --model ggml-base      # Good balance
transcriber download --model ggml-small     # Better accuracy
transcriber download --model ggml-medium    # High accuracy
transcriber download --model ggml-large     # Best accuracy, slower
```

## ⚙️ Configuration

The configuration file is automatically created at `~/.transcriber/config.json`:

```json
{
  "model_path": "~/.transcriber/models/ggml-base.bin",
  "output_format": "txt",
  "language": "auto",
  "temperature": 0.0,
  "max_duration": "30m",
  "audio_quality": "high"
}
```

### Configuration Options

- **model_path**: Path to the Whisper model file
- **output_format**: Output format (`txt`, `srt`, `vtt`, `json`)
- **language**: Target language (`auto` for auto-detection)
- **temperature**: Sampling temperature (0.0-1.0)
- **max_duration**: Maximum recording duration
- **audio_quality**: Recording quality (`low`, `medium`, `high`)

## 🛠️ Development Guide

### Project Structure

```
transcriber/
├── cmd.go              # CLI command handling
├── main.go             # Application entry point
├── transcriber.go      # Core transcription logic
├── config.go           # Configuration management
├── audio.go            # Audio recording/processing
├── models.go           # Model download/management
├── Makefile            # Build automation
└── README.md           # This file
```

### Building from Source

```bash
# Clone the repository
git clone https://github.com/nnanto/transcriber.git
cd transcriber

# Install dependencies
go mod download

# Build for development (with race detection)
make dev

# Build for production
make build

# Run tests
make test

# Build for all platforms
make release-local
```

## 🐛 Troubleshooting

### Common Issues

#### "Model not found" error
```bash
# Download the required model first
transcriber download --model ggml-base
```

#### Permission denied on macOS/Linux
```bash
# Make sure the binary is executable
chmod +x transcriber

# Or install system-wide
make install
```

#### High CPU usage
- Try using a smaller model (`ggml-tiny` or `ggml-base`)
- Reduce recording quality in config
- Limit recording duration

#### Audio recording issues
- Check microphone permissions
- Verify audio device availability
- Test with shorter durations first

### Getting Help

- Check the [Issues](https://github.com/nnanto/transcriber/issues) page
- Review configuration with `transcriber config`
- Enable verbose logging in development builds

## 📋 System Requirements

### Minimum Requirements
- **OS**: Linux, macOS 10.14+, Windows 10+
- **RAM**: 2GB (4GB recommended)
- **Storage**: 1GB free space
- **Go**: 1.19+ (for building from source)

### Model Size Requirements
- **tiny**: ~39MB, ~125MB RAM
- **base**: ~142MB, ~210MB RAM  
- **small**: ~466MB, ~550MB RAM
- **medium**: ~1.5GB, ~2GB RAM
- **large**: ~2.9GB, ~4GB RAM


## 📄 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

---

<div align="center">
Made with ❤️ by the Transcriber team
</div>
