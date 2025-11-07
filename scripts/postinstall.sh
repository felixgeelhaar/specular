#!/bin/bash
# Post-installation script for specular (specular) Linux packages

set -e

echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "  specular (Specular) installed successfully!"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""
echo "📦 Installation complete!"
echo ""
echo "🚀 Quick Start:"
echo "   specular version              # Check version"
echo "   specular --help               # View all commands"
echo ""
echo "📚 Next Steps:"
echo "   1. Install Docker (required for execution sandboxing)"
echo "      https://docs.docker.com/get-docker/"
echo ""
echo "   2. Configure an AI provider:"
echo "      • Anthropic Claude:  export ANTHROPIC_API_KEY='your-key'"
echo "      • OpenAI GPT:        export OPENAI_API_KEY='your-key'"
echo "      • Google Gemini:     export GOOGLE_API_KEY='your-key'"
echo "      • Ollama (local):    https://ollama.ai/download"
echo ""
echo "   3. Initialize a project:"
echo "      specular init --preset api-service"
echo ""
echo "🎯 Shell Completion:"
echo "   Bash:  source /usr/share/bash-completion/completions/specular"
echo "   Zsh:   autoload -Uz compinit && compinit"
echo "   Fish:  # Automatically loaded on next shell start"
echo ""
echo "   Restart your shell or source the completion file to activate."
echo ""
echo "📖 Documentation:"
echo "   https://github.com/felixgeelhaar/specular#readme"
echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

# Exit successfully
exit 0
