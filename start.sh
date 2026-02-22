#!/bin/bash

set -e

echo "📰 News Scraper - Build & Start Script"
echo "======================================="

# Build frontend
echo ""
echo "🔨 Building frontend..."
cd frontend
npm install
npm run build
cd ..

# Build Go backend
echo ""
echo "🔨 Building Go backend..."
go build -o news-scraper main.go

# Run the application
echo ""
echo "🚀 Starting News Scraper server..."
echo "   Open http://localhost:8080 in your browser"
echo ""
./news-scraper
