# Enhanced Features Guide

## New UI Features

The enhanced UI is now **enabled by default** and provides a much better experience!

### Key Improvements

✨ **Chat History Panel** - See your recent conversation at a glance
🧠 **Thinking Display** - Watch reasoning models think (with `--show-thinking`)
⏱️ **Response Timing** - See how long each response took
📝 **Word Count** - Track response length
📚 **Source Citations** - Clear display of web sources used
🎨 **Better Formatting** - Box-drawn UI with proper message separation

## Usage

### Default (Enhanced UI)

```bash
./web-ollama
```

### Show Thinking Process (for deepseek-r1, etc.)

```bash
./web-ollama --show-thinking
```

This will display the model's reasoning process in a **dimmed section** before showing the final answer.

### Classic UI (if you prefer the old style)

```bash
./web-ollama --classic
```

## UI Walkthrough

### Welcome Screen

```
╔══════════════════════════════════════════════════════════╗
║                                                          ║
║           web-ollama - AI with Web Search               ║
║                                                          ║
╚══════════════════════════════════════════════════════════╝

Model: deepseek-r1:8b
Commands: /exit (quit) | /clear (clear screen) | /history (show all)
```

### Chat History Panel

Shows your last 10 messages (5 exchanges) at the top:

```
┌─ Conversation History ─────────────────────┐
│ 15:23 [You] What's the weather?
│ 15:23 [AI] The current weather is...
│ 15:24 [You] What's 2+2?
│ 15:24 [AI] 2+2 equals 4.
└────────────────────────────────────────────┘
```

### Your Messages

```
┌─ You · 15:24:32
│ What's the latest news on AI?
└
```

### Assistant Responses (with Thinking)

```bash
# With --show-thinking flag:

┌─ Assistant · 15:24:35
│ First, let me analyze what the user is asking...
│ They want current news, so I'll need to search...
│
│ ─── Answer ───
│
│ Based on recent web sources, here are the latest AI developments:
│ [Response continues...]
│
│ 📚 Sources:
│    • https://example.com/ai-news-2025
│    • https://techblog.com/latest-ai
│
│ ⏱️  2.3s · 📝 ~143 words
└
```

### Without Thinking (default)

```
┌─ Assistant · 15:24:35
│ Based on recent web sources, here are the latest AI developments:
│ [Response continues...]
│
│ 📚 Sources:
│    • https://example.com/ai-news-2025
│    • https://techblog.com/latest-ai
│
│ ⏱️  2.3s · 📝 ~143 words
└
```

## New Commands

Type these during the conversation:

- `/exit` or `/quit` - Exit the application
- `/clear` - Clear the screen (keeps history)
- `/history` - Show full conversation history

## Configuration Flags

All previous flags still work, plus new ones:

```bash
# Show thinking process (dimmed before answer)
--show-thinking

# Use classic simple UI
--classic

# Combine with other flags
--show-thinking --verbose --model deepseek-r1:8b
```

## Thinking Display Explained

### What is "Thinking"?

Some models like **deepseek-r1** use "chain of thought" reasoning - they think through the problem step by step before answering.

### How It Works

1. Model outputs thinking tokens (wrapped in `<think>` tags)
2. web-ollama detects these and displays them **dimmed**
3. When thinking ends, shows "─── Answer ───" separator
4. Final answer displays normally

### When to Use `--show-thinking`

✅ **Use it when:**
- Using reasoning models (deepseek-r1, qwen-2.5, etc.)
- Debugging model behavior
- Learning how the model approaches problems
- Want transparency in reasoning

❌ **Skip it when:**
- Using standard models (llama, mistral) - they don't output thinking
- Want cleaner, faster output
- Don't care about the reasoning process

## Performance Impact

The enhanced UI has **minimal performance impact**:
- Binary size: +300KB (~9.0 MB vs 8.7 MB)
- Memory: Same as before
- Speed: Identical streaming performance

## Troubleshooting

### History panel not showing

The panel only appears after you've had at least one exchange. Start chatting!

### Thinking not displaying

1. Make sure you're using `--show-thinking` flag
2. Check that your model actually outputs thinking (deepseek-r1 does, llama doesn't)
3. The thinking must be wrapped in `<think>` tags

### UI looks broken

- Check your terminal width (needs at least 80 columns)
- Make sure your terminal supports ANSI colors
- Try `--classic` flag for simpler output

## Examples

### Basic usage with all features

```bash
./web-ollama --show-thinking
```

### Reasoning model with verbose logging

```bash
./web-ollama --model deepseek-r1:8b --show-thinking --verbose
```

### No search, just chat with thinking

```bash
./web-ollama --no-search --show-thinking
```

## Future Enhancements

Planned improvements:
- [ ] Collapsible thinking sections
- [ ] Save/load specific sessions
- [ ] Export conversations to markdown
- [ ] Search through history
- [ ] Syntax highlighting for code blocks
- [ ] Image support in terminal

---

Enjoy the enhanced experience! 🎉
