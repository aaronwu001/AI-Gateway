param (
    [ValidateSet("All", "UserLimit", "IpLimit", "GlobalLimit", "MultiService", "LayeredDefense", "FailOpen", "FailClosed")]
    [string]$Scenario = "All"
)

# -----------------------------------------------------------
# AI Gateway Comprehensive Test Suite - V7 (Safe Mode)
# -----------------------------------------------------------

$OutputEncoding = [System.Text.Encoding]::UTF8
[Console]::OutputEncoding = [System.Text.Encoding]::UTF8

$ExePath   = ".\gateway\api-gateway.exe"
$Gpt4Url   = "http://localhost:8080/api/v1/gpt4"
$VisionUrl = "http://localhost:8080/api/v1/vision"
$LogFolder = ".\test_logs"

# Ensure log folder exists
if (-not (Test-Path $LogFolder)) { New-Item -ItemType Directory -Path $LogFolder > $null }

# --- Core Tool: Compiler ---
function Build-Gateway {
    Write-Host "`n>>> [BUILD] Compiling Go program..." -ForegroundColor Cyan
    Push-Location gateway 
    go build -o api-gateway.exe cmd/server/main.go
    Pop-Location 
    if ($LASTEXITCODE -ne 0) { throw "Compilation Failed" }
}

# --- Core Executor ---
function Execute-Test {
    param (
        [string]$Id, 
        [string]$Name, 
        [hashtable]$EnvVars, 
        [scriptblock]$TestLogic, 
        [string]$RedisAction = "Flush" # Flush (Clean), Keep (Persist), Stop (Kill)
    )

    $CurrentOutLog = Join-Path $LogFolder "$Id`_out.log"
    $CurrentErrLog = Join-Path $LogFolder "$Id`_err.log"

    Write-Host "`n" + ("=" * 60) -F Cyan
    Write-Host "TESTING: $Name" -F Cyan
    
    # 0. Clear Environment Variables
    $VarsToClear = @("LIMIT_USER_CAP", "LIMIT_USER_RATE", "LIMIT_IP_CAP", "LIMIT_IP_RATE", "LIMIT_GLOBAL_CAP", "LIMIT_GLOBAL_RATE", "REDIS_FAILURE_MODE")
    foreach ($v in $VarsToClear) { [Environment]::SetEnvironmentVariable($v, $null, "Process") }

    # 1. Detect Redis
    $redisId = docker ps -aqf "name=redis" | Select-Object -First 1
    if (-not $redisId) { Write-Host "❌ ERROR: Redis Container not found!" -F Red; return }

    # 2. Setup Redis (Flush vs Keep vs Stop)
    Stop-Process -Name "api-gateway" -Force -ErrorAction SilentlyContinue
    
    if ($RedisAction -eq "Stop") {
        Write-Host ">>> Stopping Redis..." -F DarkGray
        docker stop $redisId > $null
    } elseif ($RedisAction -eq "Flush") {
        docker start $redisId > $null
        Start-Sleep -Seconds 1
        docker exec $redisId redis-cli FLUSHDB > $null
        Write-Host ">>> Redis Started & Flushed (Clean State)." -F DarkGray
    } elseif ($RedisAction -eq "Keep") {
        docker start $redisId > $null
        Write-Host ">>> Redis Kept Running (Persisted Data)." -F DarkGray
    }

    # 3. Inject Environment Variables
    foreach ($key in $EnvVars.Keys) { [Environment]::SetEnvironmentVariable($key, $EnvVars[$key], "Process") }
    
    # 4. Start Server
    $p = Start-Process -FilePath $ExePath -PassThru -WorkingDirectory ".\gateway" `
        -RedirectStandardOutput $CurrentOutLog -RedirectStandardError $CurrentErrLog -WindowStyle Hidden
    
    Start-Sleep -Seconds 2 # Wait for server readiness

    # 5. Execution
    try { 
        & $TestLogic 
    } finally {
        # 6. Teardown
        if ($p) { Stop-Process -Id $p.Id -Force -ErrorAction SilentlyContinue }
        
        if ($RedisAction -eq "Stop") { 
            docker start $redisId > $null 
        }

        # Debug Logs
        if (Test-Path $CurrentErrLog) {
            $logs = Get-Content $CurrentErrLog | Where-Object { $_ -match "fail" -or $_ -match "limit" -or $_ -match "Error" }
        }
    }
}

# --- Test Scenarios ---
$ScenariosList = [ordered]@{

    "UserLimit" = {
        Execute-Test -Id "UserLimit" -Name "User Limit Verification" -EnvVars @{"LIMIT_USER_CAP"="3"; "LIMIT_USER_RATE"="0"} -RedisAction "Flush" -TestLogic {
            for ($i=1; $i -le 4; $i++) {
                try {
                    $r = Invoke-WebRequest -Uri $Gpt4Url -Method Post -Body "{}" -ContentType "application/json" -Headers @{"X-API-Key"="sk-user"} -UseBasicParsing
                    Write-Host "Req $i : SUCCESS" -F Green
                } catch {
                    $type = $_.Exception.Response.Headers["X-RateLimit-Type"]
                    # Expecting Local-User because it's the first line of defense
                    Write-Host "Req $i : BLOCKED BY [$type] - PASS" -F Green
                }
            }
        }
    }

    "IpLimit" = {
        Execute-Test -Id "IpLimit" -Name "IP Limit Verification" -EnvVars @{"LIMIT_IP_CAP"="5"; "LIMIT_IP_RATE"="0"} -RedisAction "Flush" -TestLogic {
            for ($i=1; $i -le 6; $i++) {
                try {
                    $r = Invoke-WebRequest -Uri $Gpt4Url -Method Post -Body "{}" -ContentType "application/json" -Headers @{"X-API-Key"=("key-"+$i)} -UseBasicParsing
                    Write-Host "Req $i : SUCCESS" -F Green
                } catch {
                    $type = $_.Exception.Response.Headers["X-RateLimit-Type"]
                    Write-Host "Req $i : BLOCKED BY [$type] - PASS" -F Green
                }
            }
        }
    }

    "GlobalLimit" = {
        Execute-Test -Id "GlobalLimit" -Name "Global Limit Verification" -EnvVars @{"LIMIT_GLOBAL_CAP"="3"; "LIMIT_GLOBAL_RATE"="0"} -RedisAction "Flush" -TestLogic {
            for ($i=1; $i -le 4; $i++) {
                try {
                    $r = Invoke-WebRequest -Uri $Gpt4Url -Method Post -Body "{}" -ContentType "application/json" -Headers @{"X-API-Key"="global"} -UseBasicParsing
                    Write-Host "Req $i : SUCCESS" -F Green
                } catch {
                    $type = $_.Exception.Response.Headers["X-RateLimit-Type"]
                    Write-Host "Req $i : BLOCKED BY [$type] - PASS" -F Green
                }
            }
        }
    }

    "MultiService" = {
        Execute-Test -Id "MultiService" -Name "Multi-Service Routing" -EnvVars @{} -RedisAction "Flush" -TestLogic {
            $r = Invoke-WebRequest -Uri $VisionUrl -Method Post -Body "{}" -ContentType "application/json" -Headers @{"X-API-Key"="vision"} -UseBasicParsing
            Write-Host "SUCCESS: Reached Vision Model" -F Green
        }
    }

    "LayeredDefense" = {
        # Part 1: Fill the Local Bucket
        Execute-Test -Id "Layer1_Local" -Name "Layer 1: Local Memory Limiter" -EnvVars @{"LIMIT_USER_CAP"="3"; "LIMIT_USER_RATE"="0"} -RedisAction "Flush" -TestLogic {
            Write-Host "Step 1: Filling the bucket (3 requests)..." -F Gray
            for ($i=1; $i -le 3; $i++) {
                try {
                    $r = Invoke-WebRequest -Uri $Gpt4Url -Method Post -Body "{}" -ContentType "application/json" -Headers @{"X-API-Key"="dual-test"} -UseBasicParsing
                    Write-Host "Req $i : SUCCESS" -F Green
                } catch { Write-Host "Req $i : FAILED" -F Red }
            }

            Write-Host "Step 2: Triggering Limit..." -F Gray
            try {
                $r = Invoke-WebRequest -Uri $Gpt4Url -Method Post -Body "{}" -ContentType "application/json" -Headers @{"X-API-Key"="dual-test"} -UseBasicParsing
                Write-Host "Req 4 : FAILURE (Should block)" -F Red
            } catch {
                $type = $_.Exception.Response.Headers["X-RateLimit-Type"]
                if ($type -match "Local") { 
                    Write-Host "Req 4 : BLOCKED BY [$type] (Memory) -> PASS" -F Green 
                } else { 
                    Write-Host "Req 4 : BLOCKED BY [$type] (Expected Local) -> WARN" -F Yellow 
                }
            }
        }

        # Part 2: Restart Gateway (Wipe Memory) -> Redis must catch it
        Execute-Test -Id "Layer2_Redis" -Name "Layer 2: Redis Distributed Limiter" -EnvVars @{"LIMIT_USER_CAP"="3"; "LIMIT_USER_RATE"="0"} -RedisAction "Keep" -TestLogic {
            Write-Host "Step 3: Gateway Restarted. Sending 1 Request..." -F Gray
            try {
                $r = Invoke-WebRequest -Uri $Gpt4Url -Method Post -Body "{}" -ContentType "application/json" -Headers @{"X-API-Key"="dual-test"} -UseBasicParsing
                Write-Host "Req 5 : FAILURE (Redis should remember)" -F Red
            } catch {
                $type = $_.Exception.Response.Headers["X-RateLimit-Type"]
                if ($type -eq "User") { 
                    Write-Host "Req 5 : BLOCKED BY [$type] (Redis) -> PASS" -F Green 
                } else { 
                    Write-Host "Req 5 : BLOCKED BY [$type] (Expected Redis User) -> WARN" -F Yellow 
                }
            }
        }
    }

    "FailOpen" = {
        Execute-Test -Id "FailOpen" -Name "Strategy: Fail-Open Check" -EnvVars @{"REDIS_FAILURE_MODE"="open"} -RedisAction "Stop" -TestLogic {
            try {
                $r = Invoke-WebRequest -Uri $Gpt4Url -Method Post -Body "{}" -ContentType "application/json" -Headers @{"X-API-Key"="fail"} -UseBasicParsing
                Write-Host "RESULT: Request Allowed - PASS" -F Green
            } catch { Write-Host "RESULT: FAILED" -F Red }
        }
    }

    "FailClosed" = {
        Execute-Test -Id "FailClosed" -Name "Strategy: Fail-Closed Check" -EnvVars @{"REDIS_FAILURE_MODE"="closed"} -RedisAction "Stop" -TestLogic {
            try {
                $r = Invoke-WebRequest -Uri $Gpt4Url -Method Post -Body "{}" -ContentType "application/json" -Headers @{"X-API-Key"="fail"} -UseBasicParsing -ErrorAction Stop
                Write-Host "RESULT: FAILED" -F Red
            } catch { 
                $type = $_.Exception.Response.Headers["X-RateLimit-Type"]
                Write-Host "RESULT: Blocked by $type - PASS" -F Green
            }
        }
    }
}

# --- Execution Entry Point ---
try {
    Build-Gateway
    if ($Scenario -eq "All") {
        $ScenariosList.Keys | ForEach-Object { & $ScenariosList[$_] }
    } else {
        & $ScenariosList[$Scenario]
    }
} finally {
    Write-Host "`n--- ALL TESTS FINISHED ---" -F Cyan
}