@echo off
setlocal

echo 🔧 Shadow API - Local Setup

REM Check if Go is installed
where go >nul 2>nul
if errorlevel 1 (
    echo ❌ Go is not installed. Please install Go 1.22+
    exit /b 1
)

REM Download dependencies
echo 📦 Downloading dependencies...
call go mod download

REM Create data directory for SQLite
if not exist "data" mkdir data

REM Run migrations for SQLite
where goose >nul 2>nul
if errorlevel 1 (
    echo ⚠️  goose not installed. Installing...
    call go install github.com/pressly/goose/v3/cmd/goose@latest
)

echo 🗄️  Running migrations...
call goose -dir ./migrations sqlite ./data/shadow.db up

echo ✅ Setup complete!
echo.
echo 🚀 To start the API:
echo    go run ./cmd/api
echo.
echo 📡 API will be available at http://localhost:8080
