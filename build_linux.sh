#!/bin/bash

set -e

VERSION=${1:-$(git describe --tags 2>/dev/null || echo "dev")}
echo "🔨 Building Muallim App for Linux (version: $VERSION)"

# Проверка Go
if ! command -v go &> /dev/null; then
    echo "❌ Go не установлен"
    exit 1
fi

# Установка зависимостей
echo "📦 Установка зависимостей..."
sudo apt-get update
sudo apt-get install -y gcc libgl1-mesa-dev xorg-dev rpm

# Сборка бинарника
echo "📦 Сборка Linux бинарника..."
go mod tidy
GOOS=linux GOARCH=amd64 go build -o muallim-pro-linux .

# Создание DEB пакета
echo "📦 Создание DEB пакета..."
mkdir -p deb/DEBIAN
mkdir -p deb/usr/bin
mkdir -p deb/usr/share/applications
mkdir -p deb/usr/share/icons/hicolor/256x256/apps

cat > deb/DEBIAN/control << EOF
Package: muallim-pro
Version: $VERSION
Section: utils
Priority: optional
Architecture: amd64
Maintainer: Muallim Team
Description: Muallim автоматизация десктопное приложение
EOF

cp muallim-pro-linux deb/usr/bin/muallim-pro
chmod +x deb/usr/bin/muallim-pro

cat > deb/usr/share/applications/muallim-pro.desktop << EOF
[Desktop Entry]
Name=Muallim App
Exec=/usr/bin/muallim-pro
Icon=muallim-pro
Type=Application
Categories=Education;
EOF

dpkg-deb --build deb muallim-pro_${VERSION}_amd64.deb

# Создание RPM пакета
echo "📦 Создание RPM пакета..."
mkdir -p rpm/SOURCES
mkdir -p rpm/SPECS
mkdir -p rpm/BUILD

tar -czvf rpm/SOURCES/muallim-pro-${VERSION}.tar.gz \
    --transform 's|^|muallim-pro-${VERSION}/|' \
    main.go controller.go client/ ui/ go.mod go.sum

cat > rpm/SPECS/muallim-pro.spec << EOF
Name: muallim-pro
Version: $VERSION
Release: 1%{?dist}
Summary: Muallim автоматизация приложение
License: MIT
BuildArch: x86_64

%description
Десктопное приложение для автоматизации edonish.tj

%prep
%setup -q

%build
go mod tidy
GOOS=linux GOARCH=amd64 go build -o muallim-pro .

%install
mkdir -p %{buildroot}/usr/bin
mkdir -p %{buildroot}/usr/share/applications
cp muallim-pro %{buildroot}/usr/bin/
chmod +x %{buildroot}/usr/bin/muallim-pro

cat > %{buildroot}/usr/share/applications/muallim-pro.desktop << DESKTOP
[Desktop Entry]
Name=Muallim App
Exec=/usr/bin/muallim-pro
Type=Application
Categories=Education;
DESKTOP

%files
/usr/bin/muallim-pro
/usr/share/applications/muallim-pro.desktop

%clean
rm -rf %{buildroot}
EOF

rpmbuild -bb rpm/SPECS/muallim-pro.spec --define "_topdir $(pwd)/rpm"
cp rpm/RPMS/x86_64/muallim-pro-${VERSION}-1.x86_64.rpm .

# Создание директории для релизов
mkdir -p release/linux

mv muallim-pro-linux release/linux/
mv muallim-pro_${VERSION}_amd64.deb release/linux/
mv muallim-pro-${VERSION}-1.x86_64.rpm release/linux/

echo "✅ Готово! Файлы в release/linux/"
ls -lh release/linux/
