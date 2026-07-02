@echo off
title Musketeers Studio
cd /d "%~dp0"

echo.
echo ============================================
echo   Musketeers - Build and Start Dashboard
echo ============================================
echo.

where go >nul 2>&1
if errorlevel 1 (
    echo [ERROR] Go is not installed.
    echo Install from: https://go.dev/dl/
    pause
    exit /b 1
)

echo [1/3] Go version:
go version

echo.
echo [2/3] Building studio...
if not exist bin mkdir bin
go build -o bin\studio.exe .\cmd\studio
if errorlevel 1 (
    echo [ERROR] Build failed - copy errors above and send them.
    pause
    exit /b 1
)

echo.
echo [3/3] Starting http://127.0.0.1:8081/dashboard
echo Keep this window OPEN.
timeout /t 2 /nobreak >nul
start "" "http://127.0.0.1:8081/dashboard"
bin\studio.exe -api-port 8081 -data-dir studio-data -verbose

pause
