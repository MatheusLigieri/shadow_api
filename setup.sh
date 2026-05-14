#!/bin/bash
set -e

echo "🔧 Shadow API - Local Setup"

# Check if Go is installed
if ! command -v go &> /dev/null; then
    echo "❌ Go is not installed. Please install Go 1.22+"
    exit 1
fi

# Download dependencies
echo "📦 Downloading dependencies..."
go mod download

# Create data directory for SQLite
mkdir -p ./data

# Run migrations for SQLite
if ! command -v goose &> /dev/null; then
    echo "⚠️  goose not installed. Installing..."
    go install github.com/pressly/goose/v3/cmd/goose@latest
fi

echo "🗄️  Running migrations..."
goose -dir ./migrations sqlite ./data/shadow.db up

echo "✅ Setup complete!"
echo ""
echo "🚀 To start the API:"
echo "   go run ./cmd/api"
echo ""
echo "📡 API will be available at http://localhost:8080"
