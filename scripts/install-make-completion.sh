#!/bin/bash
# =============================================================================
# OpenUSP Make Completion Installer
# Quick installer for bash completion in OpenUSP project
# =============================================================================

set -e

PROJECT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
COMPLETION_SCRIPT="$PROJECT_ROOT/scripts/make-completion-advanced.bash"

echo "🎯 OpenUSP Make Completion Installer"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "📁 Project root: $PROJECT_ROOT"
echo "📄 Completion script: $COMPLETION_SCRIPT"

# Check if completion script exists
if [[ ! -f "$COMPLETION_SCRIPT" ]]; then
    echo "❌ Completion script not found: $COMPLETION_SCRIPT"
    exit 1
fi

echo ""
echo "Choose installation method:"
echo "1) Current session only (temporary)"
echo "2) Current user (~/.bashrc) - recommended"
echo "3) Project-specific (.bashrc in project)"
echo "4) System-wide (requires sudo)"
echo ""

read -p "Enter choice (1-4): " choice

case $choice in
    1)
        echo "🔄 Loading completion for current session..."
        source "$COMPLETION_SCRIPT"
        echo "✅ Completion loaded! Try: make <TAB><TAB>"
        ;;
    2)
        BASHRC="$HOME/.bashrc"
        if [[ "$OSTYPE" == "darwin"* ]]; then
            BASHRC="$HOME/.bash_profile"
        fi
        
        echo "🔄 Installing to $BASHRC..."
        
        # Check if already installed
        if grep -q "make-completion-advanced.bash" "$BASHRC" 2>/dev/null; then
            echo "⚠️  Completion already installed in $BASHRC"
        else
            echo "" >> "$BASHRC"
            echo "# OpenUSP Make Completion" >> "$BASHRC"
            echo "source \"$COMPLETION_SCRIPT\"" >> "$BASHRC"
            echo "✅ Added to $BASHRC"
        fi
        
        echo "🔄 Loading completion..."
        source "$COMPLETION_SCRIPT"
        echo "✅ Completion loaded! Restart terminal or run: source $BASHRC"
        ;;
    3)
        PROJECT_BASHRC="$PROJECT_ROOT/.bashrc"
        echo "🔄 Creating project-specific .bashrc..."
        
        cat > "$PROJECT_BASHRC" << EOF
#!/bin/bash
# OpenUSP Project Environment
# Load this with: source .bashrc

# Make completion
source "\$(dirname "\${BASH_SOURCE[0]}")/scripts/make-completion-advanced.bash"

# Project environment
export OPENUSP_PROJECT_ROOT="\$(dirname "\${BASH_SOURCE[0]}")"
export PATH="\$OPENUSP_PROJECT_ROOT/build:\$PATH"

echo "🎯 OpenUSP project environment loaded"
echo "📁 Project root: \$OPENUSP_PROJECT_ROOT"
echo "💡 Make completion available - try: make <TAB><TAB>"
EOF
        
        chmod +x "$PROJECT_BASHRC"
        source "$PROJECT_BASHRC"
        
        echo "✅ Created $PROJECT_BASHRC"
        echo "💡 To use: source .bashrc (from project root)"
        ;;
    4)
        SYSTEM_COMPLETION="/etc/bash_completion.d/openusp-make"
        echo "🔄 Installing system-wide completion..."
        
        if [[ ! -d "/etc/bash_completion.d" ]]; then
            echo "❌ /etc/bash_completion.d not found. Install bash-completion package first."
            exit 1
        fi
        
        sudo cp "$COMPLETION_SCRIPT" "$SYSTEM_COMPLETION"
        sudo chmod +r "$SYSTEM_COMPLETION"
        source "$COMPLETION_SCRIPT"
        
        echo "✅ Installed to $SYSTEM_COMPLETION"
        echo "💡 Available for all users after they restart their shell"
        ;;
    *)
        echo "❌ Invalid choice"
        exit 1
        ;;
esac

echo ""
echo "🎉 Installation complete!"
echo ""
echo "📖 Usage examples:"
echo "   make <TAB><TAB>          # Show all targets"
echo "   make build<TAB><TAB>     # Show build targets"
echo "   make start<TAB><TAB>     # Show start targets"
echo "   make-help build          # Show build targets with descriptions"
echo "   make-help                # Show all targets"
echo ""
echo "🔧 Test it now:"
echo "   make build-<TAB><TAB>"