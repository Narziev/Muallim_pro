#!/bin/bash

set -e

VERSION=${1:-$(git describe --tags 2>/dev/null || echo "dev")}
echo "🔨 Building Muallim App for Android (version: $VERSION)"

# Проверка Go
if ! command -v go &> /dev/null; then
    echo "❌ Go не установлен"
    exit 1
fi

# Проверка Android SDK
if [ -z "$ANDROID_HOME" ]; then
    echo "⚠️  ANDROID_HOME не установлена"
    echo "Установите Android SDK и настройте переменную окружения"
    exit 1
fi

# Установка Fyne
echo "📦 Установка Fyne..."
go install fyne.io/fyne/v2/cmd/fyne@latest
export PATH=$PATH:$(go env GOPATH)/bin

# Сборка APK
echo "📦 Сборка Android APK..."
go mod tidy
fyne package -os android -appID com.muallim.pro -name muallim-pro -icon icon.png

# Создание директории для релизов
mkdir -p release/android

mv muallim-pro.apk release/android/muallim-pro-${VERSION}.apk

echo "✅ Готово! APK файл в release/android/"
ls -lh release/android/
