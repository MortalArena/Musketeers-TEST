# Musketeers Studio — تشغيل سريع
# شغّل هذا الملف بالنقر المزدوج أو من PowerShell

$ErrorActionPreference = "Stop"
$Root = Split-Path -Parent $MyInvocation.MyCommand.Path
Set-Location $Root

Write-Host "Building studio..." -ForegroundColor Cyan
go build -o bin\studio.exe .\cmd\studio
if ($LASTEXITCODE -ne 0) {
    Write-Host "BUILD FAILED — see errors above" -ForegroundColor Red
    Read-Host "Press Enter to close"
    exit 1
}

Write-Host "Starting Musketeers Studio..." -ForegroundColor Green
Write-Host "Dashboard will be at: http://127.0.0.1:8081/dashboard" -ForegroundColor Yellow
Write-Host "Press Ctrl+C to stop" -ForegroundColor Gray
Write-Host ""

.\bin\studio.exe -api-port 8081 -data-dir .\studio-data -verbose
