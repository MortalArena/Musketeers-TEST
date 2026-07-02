$psi = New-Object System.Diagnostics.ProcessStartInfo
$psi.FileName = "C:\Users\mynew\Desktop\New folder (4)\musketeers\bin\studio.exe"
$psi.Arguments = "-api-port 8081 -data-dir studio-data -verbose"
$psi.WorkingDirectory = "C:\Users\mynew\Desktop\New folder (4)\musketeers"
$psi.UseShellExecute = $false
$psi.CreateNoWindow = $true
[System.Diagnostics.Process]::Start($psi)
