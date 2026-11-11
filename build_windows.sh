#!/bin/bash
# ════════════════════════════════════════════════════════════════════
# MUALLIM PRO — Windows Build Script (Go + Fyne)
# Cross-compiles from Linux/macOS to Windows, then optionally
# builds an NSIS installer if `makensis` is available.
# ════════════════════════════════════════════════════════════════════

set -e

VERSION=${1:-$(git describe --tags 2>/dev/null || echo "dev")}
echo "🔨 Building MUALLIM PRO for Windows (version: $VERSION)"

# Check Go
if ! command -v go &> /dev/null; then
    echo "❌ Go не установлен"
    exit 1
fi

# Ensure dependencies are tidy
echo "📦 Установка зависимостей..."
go mod tidy

# Output directory
mkdir -p release/windows

# Cross-compile (requires CGO for Fyne — use a C cross-compiler if available)
echo "📦 Сборка Windows бинарника..."
if command -v x86_64-w64-mingw32-gcc &> /dev/null; then
    echo "   Using mingw-w64 cross-compiler"
    CGO_ENABLED=1 \
    CC=x86_64-w64-mingw32-gcc \
    CXX=x86_64-w64-mingw32-g++ \
    GOOS=windows \
    GOARCH=amd64 \
    go build -o release/windows/muallim-pro.exe .
else
    echo "   ⚠️  x86_64-w64-mingw32-gcc not found — building without CGO"
    echo "   (Fyne features that require CGO may be limited)"
    CGO_ENABLED=0 \
    GOOS=windows \
    GOARCH=amd64 \
    go build -o release/windows/muallim-pro.exe .
fi

echo "✅ Бинарник собран: release/windows/muallim-pro.exe"

# Build NSIS installer if makensis is available
if command -v makensis &> /dev/null; then
    echo "📦 Создание NSIS установщика..."
    # installer.nsi expects a version string and the binary in a known place
    cp release/windows/muallim-pro.exe release/windows/muallim-pro.exe || true
    makensis -DVERSION="$VERSION" installer.nsi 2>&1 | tail -20 \
        && echo "✅ Установщик собран" \
        || echo "⚠️  NSIS build failed — see output above"
else
    echo "ℹ️  makensis не найден — установщик не собран. Установите NSIS для создания .exe установщика."
fi

echo ""
echo "=========================================="
echo "  Windows Build Summary"
echo "=========================================="
ls -lh release/windows/ 2>/dev/null
