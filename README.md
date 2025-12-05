# web-ollama

A CLI tool that connects Ollama to the web via local SearXNG search. I built this because I wanted to use Ollama with web browsing capabilities but couldn't find any existing tools that worked with a local search instance.

## What it does

- Runs Ollama models locally with web search capabilities
- Uses your local SearXNG instance for search (no external APIs)
- Automatically detects when a query needs web search
- Crawls URLs in parallel and feeds content to the LLM
- Displays model thinking process for reasoning models like deepseek-r1
- Renders markdown responses
- Saves conversation history

## Requirements

**Ollama** - Must be running locally
```bash
ollama serve
ollama pull deepseek-r1:8b
```

**SearXNG** - Running on port 9090 with JSON API enabled

To enable JSON API, add this to your SearXNG `settings.yml`:
```yaml
search:
  formats:
    - html
    - json
```

**Go 1.21+** - Only needed if building from source

## Installation

```bash
# Build and install
make install

# Add to PATH (one-time setup)
echo 'export PATH="$HOME/bin:$PATH"' >> ~/.zshrc
source ~/.zshrc

# Run it
web-ollama
```

Or use the install script:
```bash
./install.sh
```

## Usage

Start a conversation:
```bash
web-ollama
```

Common flags:
```bash
web-ollama --model llama2          # Use different model
web-ollama --no-search             # Disable web search
web-ollama --hide-thinking         # Hide thinking process
web-ollama --max-results 3         # Crawl fewer URLs
```

Commands during chat:
- `/exit` - Quit
- `/clear` - Clear screen
- `/history` - Show full conversation

## How it works

1. You ask a question
2. Tool analyzes if it needs web search (based on keywords like "latest", "current", etc.)
3. If yes, queries your local SearXNG
4. Crawls top 5 URLs and extracts text
5. Feeds everything to Ollama
6. Streams the response back to you

All processing happens locally. Your SearXNG instance can use whatever search engines you've configured.

## Configuration

Defaults:
- Model: `deepseek-r1:8b`
- Ollama: `http://localhost:11434`
- SearXNG: `http://localhost:9090`
- Max URLs to crawl: 5
- Thinking display: ON
- Auto-search: ON

Override with flags or edit `internal/config/config.go`.

## Project structure

```
web-ollama/
├── main.go                    # Entry point
├── internal/
│   ├── analyzer/             # Query analysis (decides when to search)
│   ├── config/               # Configuration
│   ├── crawler/              # URL crawling and text extraction
│   ├── history/              # Conversation persistence
│   ├── ollama/               # Ollama API client
│   ├── searxng/              # SearXNG API client
│   ├── terminal/             # Terminal I/O
│   └── ui/                   # Enhanced display
└── Makefile                   # Build commands
```

## Building

```bash
make build        # Build binary
make install      # Install to ~/bin
make clean        # Remove binary and history
make deps         # Download dependencies
```

The binary is about 22MB and includes everything needed to run.

## Troubleshooting

**Empty responses from model**
- Check Ollama is running: `curl http://localhost:11434/api/tags`
- Try a different model: `web-ollama --model llama2`

**SearXNG returns 403**
- Enable JSON format in `settings.yml` (see Requirements above)
- Restart SearXNG after config change

**Thinking not showing**
- Only works with reasoning models (deepseek-r1, qwen-2.5, etc.)
- Check it's enabled: thinking is ON by default
- Regular models like llama2 don't output thinking

**Web search not working**
- Verify SearXNG is running: `curl http://localhost:9090`
- Check JSON API: `curl "http://localhost:9090/search?q=test&format=json"`
- Disable if not needed: `web-ollama --no-search`

## Why I built this

I wanted to use Ollama with web browsing but all existing solutions either:
- Required external API keys
- Didn't work with local search engines
- Were overly complex
- Didn't support reasoning models properly

This tool keeps everything local and simple. Your data stays on your machine.

## License

MIT
