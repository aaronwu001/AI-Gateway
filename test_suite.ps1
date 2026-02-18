# -----------------------------------------------------------
# AI Gateway Test Suite (Full Version)
# Tests Proxy Forwarding, User Limits, IP Limits, and Global Limits
# -----------------------------------------------------------

$ExePath = ".\gateway\api-gateway.exe"
$Url     = "http://localhost:8080/api/v1/gpt4"
$LogOut  = "server_output.log"
$LogErr  = "server_error.log"

# 0. KILL ZOMBIES
Write-Host ">>> Checking for zombie processes..." -ForegroundColor Cyan
Stop-Process -Name "api-gateway" -Force -ErrorAction SilentlyContinue
Start-Sleep -Seconds 1
Write-Host ">>> Port 8080 should be free now.`n" -ForegroundColor Gray

# 1. Compile (Enter gateway folder to build)
Write-Host ">>> Compiling Go program..." -ForegroundColor Cyan
Push-Location gateway 
go build -o api-gateway.exe cmd/server/main.go
$buildStatus = $LASTEXITCODE
Pop-Location 
if ($buildStatus -ne 0) { Write-Host "COMPILE FAILED" -F Red; exit }
Write-Host ">>> Compilation Success!`n" -ForegroundColor Green

# -----------------------------------------------------------
# Test Runner Function
# -----------------------------------------------------------
function Run-TestCase {
    param (
        [string]$Name,
        [hashtable]$EnvVars,
        [scriptblock]$TestLogic
    )

    Write-Host "==================================================" -F Cyan
    Write-Host "SCENARIO: $Name" -F Cyan
    
    # ✨ FIX: 清空 Redis 資料庫 (FlushDB)
    # 這是為了避免上一次測試的 Token 殘留影響這一次
    Write-Host ">>> Flushing Redis..." -ForegroundColor DarkGray
    docker exec $(docker ps -qf "name=redis") redis-cli FLUSHDB > $null

    foreach ($key in $EnvVars.Keys) {
        [Environment]::SetEnvironmentVariable($key, $EnvVars[$key], "Process")
    }

    # Start Server
    $p = Start-Process -FilePath $ExePath -PassThru -RedirectStandardOutput $LogOut -RedirectStandardError $LogErr -WindowStyle Hidden
    Start-Sleep -Seconds 2

    try {
        & $TestLogic
    } finally {
        if ($p) { Stop-Process -Id $p.Id -Force -ErrorAction SilentlyContinue }
        Start-Sleep -Seconds 1
        
        # Check for errors (Filtering out normal startup logs)
        if (Test-Path $LogErr) {
            $errs = Get-Content $LogErr
            $realErrs = $errs | Where-Object { 
                $_ -notmatch "Server starting" -and 
                $_ -notmatch "Config Loaded" -and 
                $_ -notmatch "Forwarding" -and 
                $_ -notmatch "AI Gateway running" 
            }
            if ($realErrs) {
                Write-Host "`n[SERVER ERRORS]" -ForegroundColor Magenta
                $realErrs
            }
            Remove-Item $LogErr -Force
        }
        if (Test-Path $LogOut) { Remove-Item $LogOut -Force }
    }
    Write-Host "`n"
}

# -----------------------------------------------------------
# Scenario 1: Proxy Latency & User Limit
# Goal: 3 requests pass (slow), 4th request blocked (fast)
# -----------------------------------------------------------
Run-TestCase -Name "Integration: Proxy Latency + User Quota (Max 3)" -EnvVars @{
    "LIMIT_USER_CAP"="3"; 
    "LIMIT_USER_RATE"="0"; # No refill during test
    "LIMIT_GLOBAL_CAP"="1000"; "LIMIT_GLOBAL_RATE"="1000"
} -TestLogic {
    for ($i=1; $i -le 5; $i++) {
        $start = Get-Date
        try {
            $r = Invoke-WebRequest -Uri $Url -Headers @{"X-API-Key"="sk-test"} -UseBasicParsing
            $duration = ((Get-Date) - $start).TotalSeconds
            
            if ($duration -ge 1.5) {
                Write-Host "Request $i : SUCCESS (200) - Proxy Latency: $([math]::Round($duration, 2))s" -F Green
            } else {
                Write-Host "Request $i : SUCCESS (200) - BUT TOO FAST!" -F Yellow
            }
        } catch {
            $code = $_.Exception.Response.StatusCode.value__
            if ($code -eq 429) {
                Write-Host "Request $i : BLOCKED (429) - User Limit Works!" -F Green
            } else {
                Write-Host "Request $i : ERROR ($code)" -F Red
            }
        }
    }
}

# -----------------------------------------------------------
# Scenario 2: IP Limit
# Goal: Block requests from the same IP after 5 tries
# -----------------------------------------------------------
Run-TestCase -Name "Verify IP Limit (Max 5)" -EnvVars @{
    "LIMIT_USER_CAP"="100"; "LIMIT_USER_RATE"="100";
    "LIMIT_IP_CAP"="5"; 
    "LIMIT_IP_RATE"="0"; # No refill
    "LIMIT_GLOBAL_CAP"="100"; "LIMIT_GLOBAL_RATE"="100"
} -TestLogic {
    # Use different API keys to bypass User Limit, forcing it to hit IP Limit
    for ($i=1; $i -le 7; $i++) {
        $key = "key-$i"
        try {
            $r = Invoke-WebRequest -Uri $Url -Headers @{"X-API-Key"=$key} -UseBasicParsing
            Write-Host "Request $i : SUCCESS (200)" -F Green
        } catch {
            $code = $_.Exception.Response.StatusCode.value__
            $msg = $_.ErrorDetails.Message
            if ($msg -match "this IP") {
                Write-Host "Request $i : BLOCKED by IP Limit ($code) - PASS!" -F Green
            } else {
                Write-Host "Request $i : ERROR ($code) - $msg" -F Yellow
            }
        }
    }
}

# -----------------------------------------------------------
# Scenario 3: Global Limit
# Goal: Block EVERYTHING after 3 requests (System wide)
# -----------------------------------------------------------
Run-TestCase -Name "Verify Global Service Limit (Max 3 Total)" -EnvVars @{
    "LIMIT_USER_CAP"="100"; "LIMIT_USER_RATE"="100";
    "LIMIT_IP_CAP"="100"; "LIMIT_IP_RATE"="100";
    "LIMIT_GLOBAL_CAP"="3"; 
    "LIMIT_GLOBAL_RATE"="0" # No refill
} -TestLogic {
    for ($i=1; $i -le 5; $i++) {
        try {
            $r = Invoke-WebRequest -Uri $Url -Headers @{"X-API-Key"="global-test"} -UseBasicParsing
            Write-Host "Request $i : SUCCESS (200)" -F Green
        } catch {
            $code = $_.Exception.Response.StatusCode.value__
            if ($code -eq 503) {
                Write-Host "Request $i : BLOCKED by Global Limit (503) - PASS!" -F Green
            } else {
                Write-Host "Request $i : ERROR ($code)" -F Yellow
            }
        }
    }
}

Write-Host "ALL TESTS COMPLETED!" -F Cyan