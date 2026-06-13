$headers1 = @{
    "Authorization" = "Bearer nr-1781198048151738100"
    "Content-Type" = "application/json"
}

$headers2 = @{
    "Authorization" = "Bearer nr-1781198048088641800"
    "Content-Type" = "application/json"
}

# Unicode escaped JSON bodies to prevent Windows PowerShell encoding issues
$joinFounder = '{"channel_id":"founder"}'
$publishFounder = '{"channel_id":"founder","content":"\u0645\u0631\u062d\u0628\u0627"}'

$joinAlmoassas = '{"channel_id":"\u0627\u0644\u0645\u0624\u0633\u0633"}'
$publishAlmoassas = '{"channel_id":"\u0627\u0644\u0645\u0624\u0633\u0633","content":"\u0645\u0631\u062d\u0628\u0627"}'

# Agent 1
try {
    Invoke-RestMethod -Uri "http://127.0.0.1:8080/api/channels/join" -Method Post -Headers $headers1 -Body $joinFounder -ErrorAction Stop
    Invoke-RestMethod -Uri "http://127.0.0.1:8080/api/channels/publish" -Method Post -Headers $headers1 -Body $publishFounder -ErrorAction Stop
    
    Invoke-RestMethod -Uri "http://127.0.0.1:8080/api/channels/join" -Method Post -Headers $headers1 -Body $joinAlmoassas -ErrorAction Stop
    Invoke-RestMethod -Uri "http://127.0.0.1:8080/api/channels/publish" -Method Post -Headers $headers1 -Body $publishAlmoassas -ErrorAction Stop
    Write-Host "Agent 1 published successfully."
} catch {
    Write-Error "Agent 1 failed: $_"
}

# Agent 2
try {
    Invoke-RestMethod -Uri "http://127.0.0.1:8081/api/channels/join" -Method Post -Headers $headers2 -Body $joinFounder -ErrorAction Stop
    Invoke-RestMethod -Uri "http://127.0.0.1:8081/api/channels/publish" -Method Post -Headers $headers2 -Body $publishFounder -ErrorAction Stop
    
    Invoke-RestMethod -Uri "http://127.0.0.1:8081/api/channels/join" -Method Post -Headers $headers2 -Body $joinAlmoassas -ErrorAction Stop
    Invoke-RestMethod -Uri "http://127.0.0.1:8081/api/channels/publish" -Method Post -Headers $headers2 -Body $publishAlmoassas -ErrorAction Stop
    Write-Host "Agent 2 published successfully."
} catch {
    Write-Error "Agent 2 failed: $_"
}
