# Installation Guide

## Quick Install

```bash
# Install to ~/bin
make install
```

## Add to Your PATH

### For Zsh (macOS default)

Add this line to your `~/.zshrc`:

```bash
# Add ~/bin to PATH for web-ollama
export PATH="$HOME/bin:$PATH"
```

Then reload your shell:

```bash
source ~/.zshrc
```

### For Bash

Add this line to your `~/.bashrc` or `~/.bash_profile`:

```bash
# Add ~/bin to PATH for web-ollama
export PATH="$HOME/bin:$PATH"
```

Then reload:

```bash
source ~/.bashrc
```

## Verify Installation

```bash
# Check if web-ollama is in your PATH
which web-ollama

# Should output: /Users/monroestephenson/bin/web-ollama

# Run it!
web-ollama
```

## Auto-Install Script

Or use this one-liner to add to your `.zshrc` automatically:

```bash
echo 'export PATH="$HOME/bin:$PATH"' >> ~/.zshrc && source ~/.zshrc
```

## Uninstall

```bash
rm ~/bin/web-ollama
# Remove the PATH line from ~/.zshrc manually
```

## Build from Source

If you make changes to the code:

```bash
# Rebuild and reinstall
make install

# Or just build locally
make build
./web-ollama
```

## Update

To update to a new version:

```bash
cd web-ollama
git pull  # if using git
make install
```

---

## What Gets Installed

- **Binary**: `~/bin/web-ollama` (~22 MB)
- **Config**: Uses `~/.web-ollama/` for history
- **No other files**: Everything is self-contained

## Aliases (Optional)

You can add shortcuts to your `.zshrc`:

```bash
# Aliases for web-ollama
alias wo='web-ollama'
alias wot='web-ollama --show-thinking'
alias wons='web-ollama --no-search'
```

Then you can use:
- `wo` - Quick launch
- `wot` - With thinking visible
- `wons` - No search mode
