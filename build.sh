#!/bin/bash

# Snake Game Build Script
# Creates executables for multiple platforms

echo "🐍 Building Snake Game..."

# Create dist directory
mkdir -p dist

echo "🌐 Building Web Server..."
# Mac ARM64
GOOS=darwin GOARCH=arm64 go build -ldflags="-s -w" -o dist/webserver_mac_arm64 ./cmd/webserver
# Mac AMD64
GOOS=darwin GOARCH=amd64 go build -ldflags="-s -w" -o dist/webserver_mac_amd64 ./cmd/webserver
# Windows
GOOS=windows GOARCH=amd64 go build -ldflags="-s -w" -o dist/webserver_windows.exe ./cmd/webserver
# Linux
GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o dist/webserver_linux ./cmd/webserver

echo "📦 Packaging static files..."
# 习惯上我们需要把 web 文件夹也带上
mkdir -p dist/web
cp -r web/static dist/web/

echo ""
echo "✅ Build complete! Files in ./dist:"
ls -lh dist/
