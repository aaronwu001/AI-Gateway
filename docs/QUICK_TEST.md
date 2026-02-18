# ⚡ Quick Manual Testing Guide

This guide describes how to perform a **"Three-Terminal" manual test** to verify the API Gateway, Rate Limiter, and Mock Backend integration locally.

## 📋 Prerequisites

* **Docker Desktop** is running.
* **Go 1.21+** is installed.
* **PowerShell** (Recommended for Windows) or Terminal.

---

## 🖥️ Terminal 1: Infrastructure (Redis & Mock Backend)

Start the supporting services (Redis Database and the Python Mock AI Service).

```powershell
# From project root
docker-compose up -d

```

> **Verify:** Run `docker ps`. You should see `ai-gateway-redis` and `ai-mock-backend` running.

---

## 🖥️ Terminal 2: The Gateway (Middleman)

Start the Go Gateway with **Environment Variable Overrides**. We set a low limit (3 requests) to easily trigger the rate limiter.

**⚠️ Important:** You must be inside the `gateway` directory to run this.

```powershell
# 1. Enter the gateway directory
cd gateway

# 2. Set strict limits for testing (Overrides config.yaml)
$env:LIMIT_USER_CAP = "3"
$env:LIMIT_USER_RATE = "1"

# 3. Run the server
go run cmd/server/main.go

```

> **Success:** You should see `🚀 Route Ready` and `🌐 AI Gateway started at :8080`.

---

## 🖥️ Terminal 3: The Client (User)

Act as a user sending requests to the Gateway.

### Test A: Single Successful Request

Send one POST request to verify the full path (Client -> Gateway -> Redis -> Mock Backend).

```powershell
# Copy and paste this block
$Url = "http://localhost:8080/api/v1/gpt4"
$Headers = @{ "X-API-Key" = "manual-test-user"; "Content-Type" = "application/json" }
$Body = '{"prompt": "Hello Gateway!"}'

try {
    $r = Invoke-WebRequest -Uri $Url -Method Post -Headers $Headers -Body $Body -UseBasicParsing
    Write-Host "✅ [200 OK]" -F Green
    Write-Host "Response: $($r.Content)" -F Cyan
} catch {
    Write-Host "❌ [FAILED] $($_.Exception.Message)" -F Red
}

```

### Test B: Rate Limit Trigger (The "Spam" Test)

Send 5 requests rapidly. Since we set the cap to **3**, the 4th and 5th should fail.

```powershell
# Copy and paste this loop
Write-Host "🚀 Spamming 5 requests..." -F Magenta

for ($i=1; $i -le 5; $i++) {
    Start-Sleep -Milliseconds 200
    try {
        $r = Invoke-WebRequest -Uri "http://localhost:8080/api/v1/gpt4" -Method Post -Headers $Headers -Body '{}' -UseBasicParsing
        Write-Host "Req $i : 🟢 200 OK"
    } catch {
        $type = $_.Exception.Response.Headers["X-RateLimit-Type"]
        $status = $_.Exception.Response.StatusCode
        Write-Host "Req $i : 🔴 $status (Blocked by $type)"
    }
}

```

---

## 🎯 Expected Output

If the system is working correctly, **Test B** should output:

```text
Req 1 : 🟢 200 OK
Req 2 : 🟢 200 OK
Req 3 : 🟢 200 OK
Req 4 : 🔴 429 Too Many Requests (Blocked by User)
Req 5 : 🔴 429 Too Many Requests (Blocked by User)

```

## 🧹 Cleanup

When finished, stop the infrastructure to save resources:

```powershell
# From project root
docker-compose down

```
