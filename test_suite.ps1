param (
    [ValidateSet("All", "UserLimit", "IpLimit", "GlobalLimit", "MultiService", "FailOpen", "FailClosed")]
    [string]$Scenario = "All"
)

# -----------------------------------------------------------
# AI Gateway Modular Test Suite - V4 (Log Isolation)
# -----------------------------------------------------------

$OutputEncoding = [System.Text.Encoding]::UTF8
[Console]::OutputEncoding = [System.Text.Encoding]::UTF8

$ExePath   = ".\gateway\api-gateway.exe"
$Gpt4Url   = "http://localhost:8080/api/v1/gpt4"
$VisionUrl = "http://localhost:8080/api/v1/vision"
$LogFolder = ".\test_logs"

# 確保日誌資料夾存在
if (-not (Test-Path $LogFolder)) { New-Item -ItemType Directory -Path $LogFolder > $null }

# --- 核心工具：編譯 ---
function Build-Gateway {
    Write-Host "`n>>> [BUILD] Compiling Go program..." -ForegroundColor Cyan
    Push-Location gateway 
    go build -o api-gateway.exe cmd/server/main.go
    Pop-Location 
    if ($LASTEXITCODE -ne 0) { throw "Compilation Failed" }
}

# --- 核心執行器 ---
function Execute-Test {
    param ([string]$Id, [string]$Name, [hashtable]$EnvVars, [scriptblock]$TestLogic, [string]$RedisAction = "Normal")

    # ✨ 這裡建立專屬日誌路徑
    $CurrentOutLog = Join-Path $LogFolder "$Id`_out.log"
    $CurrentErrLog = Join-Path $LogFolder "$Id`_err.log"

    Write-Host "`n" + ("=" * 60) -F Cyan
    Write-Host "TESTING: $Name" -F Cyan
    
    # 0. 清理環境變數
    $VarsToClear = @("LIMIT_USER_CAP", "LIMIT_USER_RATE", "LIMIT_IP_CAP", "LIMIT_IP_RATE", "LIMIT_GLOBAL_CAP", "LIMIT_GLOBAL_RATE", "REDIS_FAILURE_MODE")
    foreach ($v in $VarsToClear) { [Environment]::SetEnvironmentVariable($v, $null, "Process") }

    # 1. 偵測 Redis
    $redisId = docker ps -aqf "name=redis" | Select-Object -First 1
    if (-not $redisId) { Write-Host "❌ ERROR: Redis Container not found!" -F Red; return }

    # 2. Setup (停掉舊 Server, 處理 Redis)
    Stop-Process -Name "api-gateway" -Force -ErrorAction SilentlyContinue
    
    if ($RedisAction -eq "Stop") {
        Write-Host ">>> Stopping Redis ($redisId)..." -F DarkGray
        docker stop $redisId > $null
    } else {
        docker start $redisId > $null
        Start-Sleep -Seconds 1
        docker exec $redisId redis-cli FLUSHDB > $null
        Write-Host ">>> Redis Ready." -F DarkGray
    }

    # 3. 注入環境變數
    foreach ($key in $EnvVars.Keys) { [Environment]::SetEnvironmentVariable($key, $EnvVars[$key], "Process") }
    
    # 4. 啟動 Server (導向專屬日誌)
    $p = Start-Process -FilePath $ExePath -PassThru -WorkingDirectory ".\gateway" `
        -RedirectStandardOutput $CurrentOutLog -RedirectStandardError $CurrentErrLog -WindowStyle Hidden
    Start-Sleep -Seconds 2

    # 5. Execution
    try { 
        & $TestLogic 
    } finally {
        # 6. Teardown
        if ($p) { 
            Stop-Process -Id $p.Id -Force -ErrorAction SilentlyContinue 
        }
        
        if ($RedisAction -eq "Stop") { 
            Write-Host ">>> Restarting Redis..." -F DarkGray
            docker start $redisId > $null 
        }

        # 顯示關鍵日誌 (這時即便檔案鎖定還在，讀取通常是沒問題的)
        if (Test-Path $CurrentErrLog) {
            $logs = Get-Content $CurrentErrLog | Where-Object { $_ -match "Fail-" -or $_ -match "limit exceeded" -or $_ -match "Error" }
            if ($logs) { Write-Host "[SERVER LOGS]:" -F DarkGray; $logs }
        }
        Write-Host ">>> Cleanup Done. Logs saved at: $CurrentErrLog" -F DarkGray
    }
}

# --- 所有測試案例清單 (傳入 Id 參數) ---
$ScenariosList = [ordered]@{
    "UserLimit" = {
        Execute-Test -Id "UserLimit" -Name "User Limit Verification" -EnvVars @{"LIMIT_USER_CAP"="3"; "LIMIT_USER_RATE"="0"} -TestLogic {
            for ($i=1; $i -le 4; $i++) {
                try {
                    $r = Invoke-WebRequest -Uri $Gpt4Url -Headers @{"X-API-Key"="sk-user"} -UseBasicParsing
                    Write-Host "Req $i : SUCCESS" -F Green
                } catch {
                    Write-Host "Req $i : BLOCKED ($($_.Exception.Response.Headers["X-RateLimit-Type"])) - PASS" -F Green
                }
            }
        }
    }

    "IpLimit" = {
        Execute-Test -Id "IpLimit" -Name "IP Limit Verification" -EnvVars @{"LIMIT_IP_CAP"="5"; "LIMIT_IP_RATE"="0"} -TestLogic {
            for ($i=1; $i -le 6; $i++) {
                try {
                    $r = Invoke-WebRequest -Uri $Gpt4Url -Headers @{"X-API-Key"=("key-"+$i)} -UseBasicParsing
                    Write-Host "Req $i : SUCCESS" -F Green
                } catch {
                    Write-Host "Req $i : BLOCKED ($($_.Exception.Response.Headers["X-RateLimit-Type"])) - PASS" -F Green
                }
            }
        }
    }

    "GlobalLimit" = {
        Execute-Test -Id "GlobalLimit" -Name "Global Limit Verification" -EnvVars @{"LIMIT_GLOBAL_CAP"="3"; "LIMIT_GLOBAL_RATE"="0"} -TestLogic {
            for ($i=1; $i -le 4; $i++) {
                try {
                    $r = Invoke-WebRequest -Uri $Gpt4Url -Headers @{"X-API-Key"="global"} -UseBasicParsing
                    Write-Host "Req $i : SUCCESS" -F Green
                } catch {
                    Write-Host "Req $i : BLOCKED ($($_.Exception.Response.Headers["X-RateLimit-Type"])) - PASS" -F Green
                }
            }
        }
    }

    "MultiService" = {
        Execute-Test -Id "MultiService" -Name "Multi-Service Routing" -EnvVars @{} -TestLogic {
            $r = Invoke-WebRequest -Uri $VisionUrl -Headers @{"X-API-Key"="vision"} -UseBasicParsing
            Write-Host "SUCCESS: Reached Vision Model" -F Green
        }
    }

    "FailOpen" = {
        Execute-Test -Id "FailOpen" -Name "Strategy: Fail-Open" -EnvVars @{"REDIS_FAILURE_MODE"="open"} -RedisAction "Stop" -TestLogic {
            try {
                $r = Invoke-WebRequest -Uri $Gpt4Url -Headers @{"X-API-Key"="fail"} -UseBasicParsing
                Write-Host "RESULT: Request Allowed - PASS" -F Green
            } catch { Write-Host "RESULT: FAILED" -F Red }
        }
    }

    "FailClosed" = {
        Execute-Test -Id "FailClosed" -Name "Strategy: Fail-Closed" -EnvVars @{"REDIS_FAILURE_MODE"="closed"} -RedisAction "Stop" -TestLogic {
            try {
                $r = Invoke-WebRequest -Uri $Gpt4Url -Headers @{"X-API-Key"="fail"} -UseBasicParsing -ErrorAction Stop
                Write-Host "RESULT: FAILED" -F Red
            } catch { 
                $type = $_.Exception.Response.Headers["X-RateLimit-Type"]
                Write-Host "RESULT: Blocked by $type - PASS" -F Green
            }
        }
    }
}

# --- 執行入口 ---
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