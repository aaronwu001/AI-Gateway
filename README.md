
# Go AI Gateway (Rate Limiter Edition)

![Go](https://img.shields.io/badge/Go-1.21+-00ADD8?style=flat&logo=go)
![Redis](https://img.shields.io/badge/Redis-7.0+-DC382D?style=flat&logo=redis)
![License](https://img.shields.io/badge/License-MIT-blue.svg)

A high-performance, distributed API Gateway prototype designed to protect AI model deployments. It features a robust **3-layer rate limiting strategy** powered by Redis and Lua scripts to ensure atomic operations and prevent race conditions under high concurrency.

## 🚀 Key Features

* **🛡️ 3-Layer Defense Matrix**:
    1.  **Global Service Limit**: Protects downstream infrastructure from total system collapse (Returns `503 Service Unavailable`).
    2.  **IP-Based Limit**: Defends against DDoS attacks and bot abuse from specific sources (Returns `429 Too Many Requests`).
    3.  **Identity-Based Limit**: Enforces business logic quotas via API Keys (Returns `429 Too Many Requests`).
* **⚡ Atomic Operations**: Uses custom Lua scripts within Redis to guarantee race-condition-free counting and token bucket management.
* **🧪 Automated Testing Suite**: Built-in PowerShell test runner that simulates multi-vector attacks to verify all protection layers.
* **⚙️ Configurable**: Fully adjustable rate limits via Environment Variables.

## 🛠️ Architecture

The system currently operates as a protective middleware layer utilizing a high-performance flow:

`Client Request` -> `Go Gateway` -> `Redis (Atomic Check)` -> `[Allow] -> Backend / [Deny] -> 429 Error`

*(Note: Currently, the backend is a Mock AI Service. Phase 4 will introduce Reverse Proxy capabilities to real Python/FastAPI backends.)*

## 📂 Project Structure

```text
ai-gateway/              # Monorepo Root
├── gateway/             # Go Application (The Rate Limiter)
│   ├── cmd/server/      # Main entry point
│   ├── internal/        # Core logic (Redis/Lua/Middleware)
│   └── go.mod
├── mock-backend/        # Python Service (Mock AI Models)
├── scripts/             # Utility scripts
├── docker-compose.yml   # Infrastructure orchestration
└── README.md            # Project documentation

```

## ⚡ Getting Started

### Prerequisites

* [Go 1.21+]()
* [Docker]() (for Redis)

### 1. Start Infrastructure

Spin up the Redis instance using Docker Compose:

```powershell
docker-compose up -d

```

### 2. Build the Gateway

Compile the Go application (ensure you are in the `gateway` directory or point to it):

```powershell
# From project root
go build -o gateway/api-gateway.exe gateway/cmd/server/main.go

```

### 3. Run the Server

Start the gateway (uses default configuration):

```powershell
./gateway/api-gateway.exe

```

## 🧪 Testing

We include a comprehensive **One-Shot Test Suite** (`test_suite.ps1`) that:

1. Compiles the latest code.
2. Automatically manages the server process (starts/stops).
3. Injects different Environment Variables to test specific scenarios.
4. Verifies **User Quotas**, **IP Limits**, and **Global Circuit Breaking**.

**Run the test suite:**

```powershell
./test_suite.ps1

```

You should see green **PASS** indicators for all scenarios.

## 📝 Configuration

You can override the default rate limits by setting these Environment Variables before running the server:

| Variable | Description | Default |
| --- | --- | --- |
| `LIMIT_GLOBAL_RATE` | Tokens added per second (System-wide) | `1000` |
| `LIMIT_GLOBAL_CAP` | Max burst capacity (System-wide) | `1000` |
| `LIMIT_IP_RATE` | Tokens added per second (Per IP) | `100` |
| `LIMIT_IP_CAP` | Max burst capacity (Per IP) | `100` |
| `LIMIT_USER_RATE` | Tokens added per second (Per API Key) | `10` |
| `LIMIT_USER_CAP` | Max burst capacity (Per API Key) | `10` |

## 🗺️ Roadmap

* [x] Phase 1: Basic Go HTTP Server
* [x] Phase 2: Redis Integration & Lua Scripting
* [x] Phase 3: 3-Layer Rate Limiting & Automated Testing
* [ ] **Phase 4 (Next): Reverse Proxy to Real AI Models**
* [ ] Phase 5: YAML Configuration & Service Discovery
* [ ] Phase 6: Dockerization of the Gateway itself

## 📄 License

This project is licensed under the MIT License.
