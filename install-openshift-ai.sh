#!/bin/bash

# OpenShift AI MCP Server Installation Script
# Downloads the complete OpenShift-AI enhanced version

set -e

echo "🚀 Installing OpenShift AI MCP Server (Complete Version)"
echo "📊 Includes: DataScience Projects, Models, Applications, Experiments, Pipelines"
echo ""

# Detect platform
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)

case $OS in
  darwin)
    if [[ "$ARCH" == "arm64" ]]; then
      BINARY="kubernetes-mcp-server-darwin-arm64"
    else
      echo "❌ Only ARM64 (Apple Silicon) Mac is supported"
      exit 1
    fi
    ;;
  linux)
    if [[ "$ARCH" == "x86_64" ]]; then
      BINARY="kubernetes-mcp-server-linux-amd64"
    elif [[ "$ARCH" == "aarch64" ]] || [[ "$ARCH" == "arm64" ]]; then
      BINARY="kubernetes-mcp-server-linux-arm64"
    else
      echo "❌ Unsupported Linux architecture: $ARCH"
      exit 1
    fi
    ;;
  *)
    echo "❌ Unsupported OS: $OS"
    exit 1
    ;;
esac

echo "📥 Detected platform: $OS-$ARCH"
echo "📦 Downloading: $BINARY"

# Download from GitHub releases
RELEASE_URL="https://github.com/macayaven/openshift-mcp-server/releases/download/v0.0.53-openshift-ai/$BINARY"

# Create install directory
INSTALL_DIR="$HOME/.local/bin"
mkdir -p "$INSTALL_DIR"

# Download binary
curl -L "$RELEASE_URL" -o "$INSTALL_DIR/kubernetes-mcp-server"
chmod +x "$INSTALL_DIR/kubernetes-mcp-server"

# Add to PATH if not already there
if [[ ":$PATH:" != *":$INSTALL_DIR:"* ]]; then
  echo "export PATH=\"\$PATH:$INSTALL_DIR\"" >> "$HOME/.zshrc" 2>/dev/null || echo "export PATH=\"\$PATH:$INSTALL_DIR\"" >> "$HOME/.bashrc"
  echo "✅ Added $INSTALL_DIR to PATH"
fi

echo ""
echo "✅ Installation complete!"
echo ""
echo "🎯 Usage:"
echo "   kubernetes-mcp-server --toolsets core,config,helm,openshift-ai"
echo ""
echo "🔧 Available toolsets:"
echo "   - core: Basic Kubernetes operations"
echo "   - config: Configuration management" 
echo "   - helm: Helm chart operations"
echo "   - openshift-ai: OpenShift AI/DataScience features (20 tools!)"
echo ""
echo "📚 OpenShift AI Tools included:"
echo "   • 5 DataScience Project tools"
echo "   • 5 Model tools"
echo "   • 4 Application tools" 
echo "   • 4 Experiment tools"
echo "   • 6 Pipeline tools"
echo ""
echo "🚀 Start using it now!"
echo "   kubernetes-mcp-server"