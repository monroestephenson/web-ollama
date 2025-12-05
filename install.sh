#!/bin/bash
# Install web-ollama and add to PATH

set -e

echo "🚀 Installing web-ollama..."

# Build and install
make install

# Check if ~/bin is in PATH
if echo "$PATH" | grep -q "$HOME/bin"; then
    echo "✓ ~/bin is already in your PATH"
else
    echo "⚠ ~/bin is not in your PATH yet"
    echo ""
    echo "Adding to ~/.zshrc..."

    # Backup zshrc
    if [ -f ~/.zshrc ]; then
        cp ~/.zshrc ~/.zshrc.backup
        echo "✓ Backed up ~/.zshrc to ~/.zshrc.backup"
    fi

    # Add to zshrc if not already there
    if ! grep -q 'export PATH="$HOME/bin:$PATH"' ~/.zshrc 2>/dev/null; then
        echo '' >> ~/.zshrc
        echo '# Added by web-ollama installer' >> ~/.zshrc
        echo 'export PATH="$HOME/bin:$PATH"' >> ~/.zshrc
        echo "✓ Added ~/bin to PATH in ~/.zshrc"
    else
        echo "✓ PATH already configured in ~/.zshrc"
    fi
fi

echo ""
echo "✅ Installation complete!"
echo ""
echo "To use web-ollama right now, run:"
echo "  export PATH=\"\$HOME/bin:\$PATH\""
echo "  web-ollama"
echo ""
echo "Or reload your shell:"
echo "  source ~/.zshrc"
echo "  web-ollama"
echo ""
echo "Optional aliases you can add to ~/.zshrc:"
echo "  alias wo='web-ollama'"
echo "  alias wot='web-ollama --show-thinking'"
echo "  alias wons='web-ollama --no-search'"
