# web-ollama

A Go-based CLI tool that combines local Ollama LLM models with SearXNG web search to provide contextually-aware, real-time answers. The tool automatically detects when web search is needed, crawls relevant URLs, and streams responses from your local LLM.

## Features

### Core Features
- 🤖 **Local LLM Integration**: Uses your local Ollama instance for privacy and speed
- 🔍 **Smart Web Search**: Auto-detects when queries need web search (via local SearXNG)
- 🕷️ **Parallel Web Crawling**: Fetches and extracts text from top 5 search results
- ⚡ **Real-time Streaming**: See LLM responses as they're generated
- 💾 **Conversation History**: Persists conversations across sessions
- 🔧 **Configurable**: CLI flags for model selection, URLs, and behavior

### ✨ NEW: Enhanced UI Features
- 🎨 **Beautiful Chat Interface**: Box-drawn UI with message history panel
- 🧠 **Thinking Display**: Watch reasoning models think through problems (deepseek-r1, etc.)
- ⏱️ **Response Metrics**: See timing, word count, and sources for each response
- 📚 **Clear Source Citations**: Web sources displayed prominently
- 🎯 **Interactive Commands**: `/exit`, `/clear`, `/history` commands
- 📊 **Conversation Panel**: See your recent chat history at a glance

> **Note**: The enhanced UI is now **enabled by default**! See [FEATURES.md](FEATURES.md) for details.

## Architecture

```
User Input → Query Analyzer → [Auto-detect] → SearXNG → Crawl URLs (parallel)
→ Build Context → Ollama (streaming) → Terminal Display → Save History
```

## Prerequisites

1. **Ollama** - Local LLM runtime
   ```bash
   # Install: https://ollama.ai
   ollama serve
   ollama pull deepseek-r1:8b  # or your preferred model
   ```

2. **SearXNG** - Local search engine
   ```bash
   # Running on http://localhost:9090
   # Ensure JSON API is enabled in settings.yml:
   # search:
   #   formats:
   #     - html
   #     - json
   ```

3. **Go 1.21+** (for building from source)

## Installation

### Option 1: Build from source

```bash
# Clone the repository
git clone <repository-url>
cd web-ollama

# Install dependencies
make deps

# Build the binary
make build

# Install to ~/bin
make install
```

### Option 2: Download pre-built binary

Download the appropriate binary for your platform from the releases page and place it in your PATH.

## Usage

### Basic Usage

```bash
web-ollama
```

This starts an interactive session with default settings:
- Model: `deepseek-r1:8b`
- Ollama: `http://localhost:11434`
- SearXNG: `http://localhost:9090`
- Auto-search: Enabled

### Command-line Flags

```bash
web-ollama [flags]

Flags:
  --model string          Ollama model name (default "deepseek-r1:8b")
  --ollama-url string     Ollama API URL (default "http://localhost:11434")
  --searxng-url string    SearXNG URL (default "http://localhost:9090")
  --no-search            Disable automatic web search
  --auto-search          Enable automatic web search (default true)
  --max-results int      Maximum URLs to crawl (default 5)
  --verbose              Enable verbose logging
```

### Examples

```bash
# Use a different model
web-ollama --model llama2:13b

# Disable auto-search
web-ollama --no-search

# Use custom SearXNG instance
web-ollama --searxng-url http://mysearx.local:8080

# Verbose mode for debugging
web-ollama --verbose
```

### Interactive Commands

Once running, type your questions or use these commands:

- Type your question and press Enter
- `/exit` or `/quit` - Exit the application
- `Ctrl+C` - Graceful shutdown

## How It Works

### Auto-Detection Algorithm

The tool uses a **scoring system** to determine if a query needs web search:

| Pattern | Score | Examples |
|---------|-------|----------|
| Time-sensitive keywords | +40 | "latest", "current", "today", "2025", "news" |
| Factual queries | +30 | "what is", "who is", "price of", "weather" |
| Research queries | +20 | "compare", "best", "review", "vs" |
| Explanation queries | -30 | "explain", "how does", "concept of" |
| Code queries | -40 | "code", "function", "debug", "algorithm" |

**Threshold**: Score > 40 triggers web search

### Search & Crawl Flow

1. **Search**: Query local SearXNG instance
2. **Select**: Get top 5 results by relevance score
3. **Crawl**: Fetch URLs in parallel (5 workers)
4. **Extract**: Clean HTML → plain text (~500 words per source)
5. **Context**: Feed to LLM with conversation history
6. **Stream**: Display response in real-time

### Conversation History

- Stored in: `~/.web-ollama/history.json`
- Format: JSON with sessions and messages
- Metadata: Tracks which messages used web search
- Pruning: Keeps last 10 sessions
- Context: Last 10 messages included in prompts

## Project Structure

```
web-ollama/
├── main.go                          # Entry point, orchestration
├── internal/
│   ├── config/                      # Configuration management
│   ├── analyzer/                    # Query analysis for search triggers
│   ├── searxng/                     # SearXNG client
│   ├── crawler/                     # Parallel web crawler
│   ├── ollama/                      # Ollama client with streaming
│   ├── history/                     # Conversation persistence
│   └── terminal/                    # UI and display
├── go.mod                           # Dependencies
├── Makefile                         # Build automation
└── README.md                        # This file
```

## Troubleshooting

### "Ollama is unreachable"

```bash
# Make sure Ollama is running
ollama serve

# Test manually
curl http://localhost:11434/api/tags
```

### "Model not found"

```bash
# Pull the model
ollama pull deepseek-r1:8b

# List available models
ollama list
```

### "SearXNG returned 403 Forbidden"

Edit your SearXNG `settings.yml` to enable JSON API:

```yaml
search:
  formats:
    - html
    - json
```

Then restart SearXNG.

### "Web search will be disabled"

This is a non-fatal warning. The tool will continue without web search. Either:
1. Fix SearXNG and restart the tool
2. Use `--no-search` flag to suppress the warning

## Configuration

### Environment

The tool respects standard environment variables:
- `HOME` - For locating history file

### Files

- `~/.web-ollama/history.json` - Conversation history
- `~/.web-ollama/history.json.backup` - Backup if corruption detected

## Development

### Building

```bash
make build          # Build for current platform
make build-all      # Cross-compile for all platforms
make test           # Run tests
make clean          # Remove binary and history
```

### Testing

```bash
# Run all tests
go test -v ./...

# Test specific package
go test -v ./internal/analyzer
```

### Dependencies

- `github.com/google/uuid` - UUID generation for sessions
- `golang.org/x/net` - HTML parsing

All other dependencies are from Go's standard library.

## Performance

- **Binary Size**: ~8-12 MB (static binary)
- **Memory Usage**: ~50-100 MB (depends on model context)
- **Crawl Time**: ~5-15 seconds for 5 URLs (parallel)
- **Response Time**: Depends on LLM model and query complexity

## Security

- ✅ All data stays local (no cloud APIs)
- ✅ Conversation history stored with 0600 permissions
- ✅ No external network access except configured services
- ✅ Timeout protection on all network operations
- ✅ Content size limits prevent memory exhaustion

## Roadmap

Future enhancements:

- [ ] Session management (`/new`, `/history`, `/load`)
- [ ] Export conversations to Markdown
- [ ] Content caching with TTL
- [ ] Multi-model comparison mode
- [ ] Configuration file support (`~/.web-ollama/config.yml`)
- [ ] RAG database for persistent knowledge

## Contributing

Contributions are welcome! Please:

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests if applicable
5. Submit a pull request

## License

[Add your license here]

## Acknowledgments

- [Ollama](https://ollama.ai) - Local LLM runtime
- [SearXNG](https://github.com/searxng/searxng) - Privacy-respecting metasearch engine
- [Go](https://golang.org) - Programming language
