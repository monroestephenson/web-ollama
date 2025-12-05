# Quick Start Guide

## Prerequisites

Before running web-ollama, ensure you have:

### 1. Ollama Running

```bash
# Install Ollama from https://ollama.ai
# Start Ollama server
ollama serve

# In another terminal, pull a model
ollama pull deepseek-r1:8b
# or another model like:
# ollama pull llama2:13b
# ollama pull mistral
```

### 2. SearXNG Running (Optional but Recommended)

```bash
# SearXNG should be running on http://localhost:9090
# Make sure JSON API is enabled in settings.yml:
#
# search:
#   formats:
#     - html
#     - json
```

If you don't have SearXNG, you can run with `--no-search` flag.

## Installation

### Option 1: Build from Source

```bash
cd web-ollama
make deps    # Download dependencies
make build   # Build binary
```

### Option 2: Install to ~/bin

```bash
make install
# Adds web-ollama to ~/bin (make sure it's in your PATH)
```

## Usage

### Basic Usage (with all defaults)

```bash
./web-ollama
```

This will:
- Use model: `deepseek-r1:8b`
- Connect to Ollama at: `http://localhost:11434`
- Connect to SearXNG at: `http://localhost:9090`
- Auto-search enabled
- Save history to: `~/.web-ollama/history.json`

### Common Scenarios

#### 1. Use a Different Model

```bash
./web-ollama --model llama2:13b
```

#### 2. Disable Web Search

```bash
./web-ollama --no-search
```

#### 3. Custom SearXNG URL

```bash
./web-ollama --searxng-url http://my-searxng:8080
```

#### 4. Verbose Mode (for debugging)

```bash
./web-ollama --verbose
```

#### 5. Limit Search Results

```bash
./web-ollama --max-results 3
```

## Example Interaction

```
╔════════════════════════════════════════╗
║   web-ollama - AI with Web Search     ║
╚════════════════════════════════════════╝

Model: deepseek-r1:8b
Type your questions or '/exit' to quit

> What's the latest news on AI?

⠋ Searching the web...
⠋ Crawling 5 URLs...
📚 Gathered information from 5 sources