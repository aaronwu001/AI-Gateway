# Go AI Gateway (Resilient Reverse Proxy Edition)

A high-performance, distributed **AI Gateway** designed to protect and route traffic for LLM deployments. It combines a **Reverse Proxy** with a robust **3-layer rate limiting strategy**, ensuring atomic operations via Redis Lua scripts and system resilience through configurable failure modes.

## 🚀 Key Features

* **🛡️ 3-Layer Defense Matrix (Per Service)**:
1. **Global Service Limit**: Protects specific downstream models (e.g., GPT-4) from infrastructure collapse (Returns `503 Service Unavailable`).
2. **IP-Based Limit**: Defends against DDoS attacks and bot abuse (Returns `429 Too Many Requests`).
3. **Identity-Based Limit**: Enforces business logic quotas via API Keys (Returns `429 Too Many Requests`).


* **🔀 Multi-Service Reverse Proxy**: Dynamically routes traffic to different backend services (e.g., `/api/v1/gpt4`, `/api/v1/vision`) based on `config.yaml`.
* **⚡ Atomic Operations**: Uses custom Lua scripts to prevent race conditions during high concurrency.
* **🔌 Resilient Failure Strategies**: Configurable **Fail-Open** (allow traffic when Redis is down) or **Fail-Closed** (block traffic) modes.
* **mag_right Observability**: Injects diagnostic headers (`X-RateLimit-Type`) to clarify why a request was blocked (Global vs. IP vs. User).
* **🧪 Robust Automation**: Includes a modular PowerShell test suite with **log isolation** and environment cleanup.

## 🛠️ Architecture

The system operates as a smart middleware layer:

`Client` -> `[Middleware: Rate Limit Check]` -> `[Redis]` -> `[Reverse Proxy]` -> `AI Service (Python/FastAPI)`

**Failure Handling:**

* If Redis is **Online**: Enforces limits strictly.
* If Redis is **Offline**:
* *Mode: Open* -> Bypasses checks, allows traffic (Availability over Consistency).
* *Mode: Closed* -> Blocks all traffic (Consistency over Availability).



## 📂 Project Structure

```text
ai-gateway/              # Monorepo Root
├── gateway/             # Go Application
│   ├── cmd/server/      # Main entry point & Router
│   ├── internal/
│   │   ├── config/      # YAML Configuration Loader
│   │   ├── limiter/     # Redis & Lua Logic
│   │   └── server/      # Middleware & HTTP Handlers
│   ├── config.yaml      # Service & Limit Configuration
│   └── go.mod
├── mock-backend/        # Python Service (Mock AI Models)
├── test_logs/           # Isolated logs from test runs (Gitignored)
├── test_suite.ps1       # Modular Automated Testing Tool
└── docker-compose.yml   # Redis Infrastructure

```

## ⚡ Getting Started

### Prerequisites

* [Go 1.21+](https://go.dev/)
* [Docker](https://www.docker.com/) (for Redis)

### 1. Start Infrastructure

Spin up the Redis instance:

```powershell
docker-compose up -d

```

### 2. Configuration

Ensure `gateway/config.yaml` exists. Example:

```yaml
server:
  port: 8080
redis:
  addr: "localhost:6379"
  password: ""
  failure_mode: "open" # Options: "open" or "closed"
services:
  - name: "gpt4-service"
    path: "/api/v1/gpt4"
    target_url: "http://localhost:5001"
    rate_limit:
      global_rate: 1000
      global_capacity: 1000
      # ... other limits

```

### 3. Build & Run

```powershell
# From project root
go build -o gateway/api-gateway.exe gateway/cmd/server/main.go
./gateway/api-gateway.exe

```

## 🧪 Automated Testing

We provide a professional-grade **Test Suite** (`test_suite.ps1`) that performs integration testing with **Environment Variable Injection** and **Log Isolation**.

### Usage

**Run All Tests (Recommended):**

```powershell
./test_suite.ps1 -Scenario All

```

**Run Specific Scenarios:**

```powershell
./test_suite.ps1 -Scenario FailClosed
./test_suite.ps1 -Scenario UserLimit

```

### What it tests:

1. **User/IP/Global Limits**: Verifies precise blocking at specific thresholds.
2. **Header Verification**: Checks `X-RateLimit-Type` to ensure the *correct* rule triggered the block.
3. **Resilience**: Kills the Redis container to verify **Fail-Open** and **Fail-Closed** behavior.
4. **Routing**: Confirms traffic reaches the correct downstream service.

## 📝 Configuration & Overrides

The Gateway uses a hybrid configuration model. `config.yaml` is the default, but you can override settings via Environment Variables (useful for CI/CD or Testing).

| Feature | YAML Key | Env Variable Override |
| --- | --- | --- |
| **Failure Mode** | `redis.failure_mode` | `REDIS_FAILURE_MODE` |
| **Global Cap** | `services[].rate_limit.global_capacity` | `LIMIT_GLOBAL_CAP` |
| **IP Rate** | `services[].rate_limit.ip_rate` | `LIMIT_IP_RATE` |
| **User Cap** | `services[].rate_limit.user_capacity` | `LIMIT_USER_CAP` |

## 🗺️ Roadmap

* [x] **Phase 1**: Basic HTTP Server & Proxy
* [x] **Phase 2**: Redis Integration & Lua Atomic Counters
* [x] **Phase 3**: 3-Layer Rate Limiting logic
* [x] **Phase 4**: Reverse Proxy & Multi-Service Routing
* [x] **Phase 5**: YAML Config, Resilience (Fail-Open/Closed), & Advanced Testing
* [ ] **Phase 6 (Current)**: **Response Modifier & Token Counting** (Intercepting JSON streams)
* [ ] **Phase 7**: Real Backend Integration & Dockerizing the Gateway

## 📄 License

This project is licensed under the MIT License.
