# Changelog

## Version 2.0 - Enhanced UI (Current)

### ✨ New Features

**Markdown Rendering**
- Beautiful markdown rendering with syntax highlighting
- Code blocks, lists, headers, bold, italic all rendered properly
- Raw response streams first, then rendered markdown shown
- Powered by Glamour terminal renderer

**Thinking Display (Now Default!)**
- Thinking is now **enabled by default** for reasoning models
- Watch deepseek-r1 and similar models think through problems
- Thinking shown dimmed/grayed before the final answer
- Use `--hide-thinking` flag to disable if desired

**Removed Features**
- Conversation history panel removed (by user request)
- Cleaner, focused UI

### 🔧 Configuration

**New Flags:**
- `--show-thinking` (default: true) - Show model thinking
- `--hide-thinking` - Hide model thinking process

**Default Behavior:**
- Thinking: **ON** by default
- Markdown: **ON** by default
- History panel: **OFF**

### 🐛 Bug Fixes

- Fixed empty responses from models (was treating all output as thinking)
- Properly parse deepseek-r1's `"thinking"` JSON field
- Fixed thinking detection for models that don't output `<think>` tags

### 📦 Build

- Binary size: ~22 MB (includes markdown renderer)
- All dependencies: glamour, uuid, golang.org/x/net

---

## Version 1.0 - Initial Release

### Features

- Local Ollama integration
- SearXNG web search
- Parallel web crawling (top 5 URLs)
- Smart query analysis
- Real-time streaming
- Conversation history persistence
- Terminal UI with colors and spinners

### Configuration

- CLI flags for model, URLs, search behavior
- Auto-search enabled by default
- Default model: deepseek-r1:8b

