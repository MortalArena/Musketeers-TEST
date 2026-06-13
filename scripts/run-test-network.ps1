# Set PoW difficulty to 1 for rapid local test network startup
$env:NR_POW_DIFFICULTY = "1"

# Clean up any existing logs and data to ensure fresh keys/tokens
Remove-Item -Path "seed.log", "seed_err.log", "agent1.log", "agent1_err.log", "agent2.log", "agent2_err.log" -ErrorAction SilentlyContinue
Remove-Item -Path "seed-data", "agent-data-1", "agent-data-2" -Recurse -Force -ErrorAction SilentlyContinue

Write-Host "1. Starting Seed (Bootstrap) node..." -ForegroundColor Cyan
$seedProcess = Start-Process .\bin\seed.exe -ArgumentList "-port 4001 -data ./seed-data" -RedirectStandardOutput "seed.log" -RedirectStandardError "seed_err.log" -PassThru -NoNewWindow

# Wait for seed to initialize and print address
Start-Sleep -Seconds 5
$bootstrapAddr = ""
if (Test-Path "seed.log") {
    $content = Get-Content "seed.log"
    foreach ($line in $content) {
        if ($line -like "*/ip4/127.0.0.1/tcp/4001/p2p/*") {
            $bootstrapAddr = $line.Trim()
            break
        }
    }
}

if (-not $bootstrapAddr) {
    Write-Error "Failed to retrieve bootstrap address from seed.log"
    Exit
}

Write-Host "Bootstrap address found: $bootstrapAddr" -ForegroundColor Green

Write-Host "2. Starting Agent 1 (REST port 8080)..." -ForegroundColor Cyan
$agent1Process = Start-Process .\bin\agent.exe -ArgumentList "-port 4002 -rest 8080 -data ./agent-data-1 -bootstrap `"$bootstrapAddr`"" -RedirectStandardOutput "agent1.log" -RedirectStandardError "agent1_err.log" -PassThru -NoNewWindow

Write-Host "3. Starting Agent 2 (REST port 8081)..." -ForegroundColor Cyan
$agent2Process = Start-Process .\bin\agent.exe -ArgumentList "-port 4003 -rest 8081 -data ./agent-data-2 -bootstrap `"$bootstrapAddr`"" -RedirectStandardOutput "agent2.log" -RedirectStandardError "agent2_err.log" -PassThru -NoNewWindow

# Wait for agents to initialize and print tokens
Start-Sleep -Seconds 6

$token1 = ""
$did1 = ""
if (Test-Path "agent1_err.log") {
    $content = Get-Content "agent1_err.log"
    foreach ($line in $content) {
        if ($line -like "*REST API token*") {
            $token1 = ($line -split "token=")[1].Trim()
        }
        if ($line -like "*did=*") {
            # Extract DID
            $parts = $line -split "did="
            $did1 = ($parts[1] -split " ")[0].Trim()
        }
    }
}

$token2 = ""
$did2 = ""
if (Test-Path "agent2_err.log") {
    $content = Get-Content "agent2_err.log"
    foreach ($line in $content) {
        if ($line -like "*REST API token*") {
            $token2 = ($line -split "token=")[1].Trim()
        }
        if ($line -like "*did=*") {
            # Extract DID
            $parts = $line -split "did="
            $did2 = ($parts[1] -split " ")[0].Trim()
        }
    }
}

Write-Host "`n=== NeuroRoot Test Cluster Started ===" -ForegroundColor Green
Write-Host "Seed Node PID: $($seedProcess.Id)"
Write-Host "Agent 1 PID: $($agent1Process.Id) | DID: $did1"
Write-Host "Agent 2 PID: $($agent2Process.Id) | DID: $did2"
Write-Host "======================================"

$url1 = "http://127.0.0.1:8080/dashboard?token=$token1"
$url2 = "http://127.0.0.1:8081/dashboard?token=$token2"

Write-Host "`nOpening Agent 1 Dashboard: $url1" -ForegroundColor Yellow
Start-Process $url1

Write-Host "Opening Agent 2 Dashboard: $url2" -ForegroundColor Yellow
Start-Process $url2

Write-Host "`nTo stop the cluster, run the stop-test-network.bat script created in this folder." -ForegroundColor Cyan

# Output processes to a batch file for easy shutdown
$stopBatchContent = "@echo off`r`ntaskkill /F /PID $($seedProcess.Id)`r`ntaskkill /F /PID $($agent1Process.Id)`r`ntaskkill /F /PID $($agent2Process.Id)`r`ndel stop-test-network.bat`r`n"
$stopBatchContent | Out-File -FilePath "stop-test-network.bat" -Encoding ascii

Write-Host "`nKeeping script active to maintain child processes. Press Ctrl+C to terminate." -ForegroundColor Cyan
while ($true) {
    Start-Sleep -Seconds 10
}
