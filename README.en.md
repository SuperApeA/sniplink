[简体中文](README.md) | English
# sniplink

A high-concurrency URL shortener service written in Go.

[![Go Version](https://img.shields.io/badge/Go-1.22+-00ADD8?style=flat-square&logo=go)](https://go.dev)
[![Framework](https://img.shields.io/badge/Framework-Gin-008080?style=flat-square)](https://github.com/gin-gonic/gin)
[![ORM](https://img.shields.io/badge/ORM-GORM-1B9AAA?style=flat-square)](https://gorm.io)
[![Database](https://img.shields.io/badge/Database-MySQL-4479A1?style=flat-square&logo=mysql&logoColor=white)](https://www.mysql.com)
[![Cache](https://img.shields.io/badge/Cache-Redis-DC382D?style=flat-square&logo=redis&logoColor=white)](https://redis.io)
[![Deploy](https://img.shields.io/badge/Deploy-Docker-2496ED?style=flat-square&logo=docker&logoColor=white)](https://www.docker.com)

![Homepage screenshot](https://foruda.gitee.com/images/1792857357279554857/bb4f3e52_7824773.png "image-20261024232731937.png")

## Features

- **Short code generation**: Redis `INCR` for ID issuance + Base62 encoding, with a DB unique index as a safety net — inherently collision-free.
- **Redirect caching**: Cache-Aside pattern with null-value caching to prevent cache penetration.
- **Visit statistics**: In-memory aggregation with scheduled batch flush to the DB — avoids database writes on every redirect, keeping the main path fast.
- **Engineering quality**: Graceful shutdown, health checks, dynamic rate limiting, fully config-driven.
- **Deployment**: Multi-stage Docker build, minimal image size, out-of-the-box.

## Key Design Decisions

| Challenge | Solution |
| :--- | :--- |
| **Short code generation** | Redis `INCR` ID generator + Base62 encoding, no collision-retry pressure |
| **Cache penetration** | Null-value caching (60s TTL) + DB unique index as a fallback |
| **Redirect status code** | **302** temporary redirect for accurate click statistics; switch to 301 if you want maximum performance |
| **Visit statistics** | In-memory channel-based batch aggregation, flushed to MySQL periodically, avoiding row-lock contention |
| **Route conflicts** | Short links handled by Gin's `NoRoute` fallback; APIs explicitly registered under `/api/v1`, completely avoiding wildcard vs. static route conflicts |

## Architecture

```text
Client ──> Gin ──> Service ──┬─> Redis (short code → long URL cache + ID generator)
                             └─> MySQL (persistence)
```

## Project Structure

```
sniplink/
├── cmd/server/             # Entry point, dependency injection & lifecycle management
├── configs/                # Config file templates (config.example.yaml)
├── internal/               # Internal business logic (no external imports allowed)
│   ├── cache/              # Redis cache wrapper
│   ├── config/             # Viper config loading & parsing
│   ├── handler/            # HTTP controllers (Gin)
│   ├── idgen/              # Redis-based distributed ID generator
│   ├── middleware/         # Rate limiting and other middleware
│   ├── model/              # GORM database models
│   ├── repository/         # MySQL data access layer (DAO)
│   ├── router/             # Route registration & wiring
│   └── service/            # Core business logic
├── scripts/                # Environment setup scripts (setup_go_env.sh)
├── test/                   # Project tests
│   ├── benchmark/          # Benchmark scripts & reports
│   └── cloudflare-tunnel/  # Public network mapping tests
├── web/                    # Frontend static assets
├── docker-compose.yml      # Local dev middleware orchestration (MySQL/Redis)
└── Makefile                # Automated build & run scripts
```

## Quick Start

### 1. Dev Environment Setup (Ubuntu)

On a fresh Ubuntu machine, bootstrap the Go environment, domestic (China) proxies, and build tools in one shot:

```bash
bash scripts/setup_go_env.sh
```

### 2. Start Dependencies & Run the Service

```
make docker-up  # start MySQL and Redis containers
make migrate    # create tables
make run        # start the sniplink service (requires go run)
```

*Note: GORM `AutoMigrate` is enabled by default. In production, it is recommended to disable it and use `make migrate` with SQL scripts for manual table creation.*

### 3. Public Access (via Cloudflare Tunnel — free, for development-stage testing)

This project can expose your local service to the public internet through Cloudflare Tunnel (intranet penetration), giving you a real HTTPS domain without purchasing a cloud server or a public IP.

**3.1 Install cloudflared (via APT source)**

Installing from the official APT source is recommended so it auto-updates with the system and comes with signature verification (more secure):

```bash
# Add the Cloudflare official GPG signing key
sudo mkdir -p --mode=0755 /usr/share/keyrings
curl -fsSL https://pkg.cloudflare.com/cloudflare-public-v2.gpg | sudo tee /usr/share/keyrings/cloudflare-public-v2.gpg >/dev/null
# Add the APT repository
echo 'deb [signed-by=/usr/share/keyrings/cloudflare-public-v2.gpg] https://pkg.cloudflare.com/cloudflared any main' | sudo tee /etc/apt/sources.list.d/cloudflared.list
# Update and install
sudo apt-get update && sudo apt-get install -y cloudflared
```

**3.2 Start the Tunnel**

Make sure your sniplink service (`make run`) is listening on port `8080`, then open a new terminal window and run:

```
cloudflared tunnel --url http://localhost:8080
```

**3.3 Get the Public Address**

The terminal will print a `trycloudflare.com` address, for example:

```
Your quick Tunnel has been created! Visit it at (it may take some time to be reachable):
https://petite-arnold-novel-mem.trycloudflare.com
```

**Just open that address in your browser and experience your short-link service on the public internet!**

*Note: Because the demo uses Cloudflare's temporary tunnel, the domain is long, so the generated short links aren't exactly 'short' in total length. In production, simply bind the service to a very short custom domain (e.g., a.cn) to get truly short links.*
*The short link is even longer than the original one — a bit of dark humor, 😄. Just buy a short domain and bind it: the generated short link becomes 'short-domain/custom-code' — take Bilibili's bt2.tv as a reference.*

> **Notes**:
>
> 1. **Temporary tunnel**: `trycloudflare` provides a temporary test domain — it changes every time the `cloudflared` command-line process is restarted.
> 2. **Keep it online**: As soon as your local machine shuts down or the `cloudflared` terminal is closed, the public link dies immediately.
> 3. **Dynamic domain adaptation**: Since the public domain changes, the frontend (`web/index.html`) builds short-link URLs dynamically with `window.location.origin`, and the backend API uses relative paths — adapting to public deployment seamlessly **without changing any code**.
> 4. **Real IP retrieval**: Thanks to Gin's default trust of local proxies (`TrustedProxies`), when requests are forwarded by local `cloudflared`, Gin automatically extracts the user's real public IP from the `CF-Connecting-IP` or `X-Forwarded-For` HTTP headers. This means future IP-based anti-abuse rate limiting can pinpoint real users precisely.

### 4. Test the API (Locally)

**Create a short link:**

```
curl -X POST http://localhost:8080/api/v1/links \
  -H "Content-Type: application/json" \
  -d '{"url":"https://go.dev", "custom_code":"golang"}'
```

**Test the redirect (browser or curl):**

Visit `http://localhost:8080/golang` — it will return `302 Found` and redirect to `https://go.dev`.

## Configuration

This project uses **Viper** for configuration management:

1. `configs/config.yaml` file configuration (local development)
2. **Environment variable overrides** (production deployment)

In production, configuration can be dynamically overridden by injecting environment variables (e.g., `MYSQL_DSN`, `REDIS_ADDR`), enabling image reuse and isolating sensitive information.

## API Test Screenshots

![Short link creation & conflict detection](https://foruda.gitee.com/images/1792778841888796397/e7516498_7824773.png "image-20261023150610260.png")
![Short link redirect test](https://foruda.gitee.com/images/1792778876749411559/3d67d0a6_7824773.png "image-20261023150726460.png")

## Performance Benchmark

To validate throughput and stability under high concurrency, this project was stress-tested with `wrk` across multiple dimensions, including bottleneck analysis at different concurrency levels.

### Development & Benchmark Environment

To ensure the data is authentic and reproducible, all benchmarks ran in a local WSL2 virtualized environment:

| Component | Config / Version |
| :--- | :--- |
| **CPU** | Intel(R) Core(TM) i5-8300H CPU @ 2.30GHz (WSL2 allocated 4 cores) |
| **Memory** | 7.8 GiB (WSL2 allocated) |
| **OS** | Ubuntu 24.04.5 LTS |
| **Linux kernel** | 6.18.33.2-microsoft-standard-WSL2 |
| **Go version** | go1.22.2 linux/amd64 |
| **Docker** | Docker version 29.8.2 |
| **MySQL** | MySQL 8.0.46 (Docker deployment) |
| **Redis** | Redis server v=7.4.11 (Docker deployment) |
| **Benchmark tool** | wrk debian/4.1.0-4build2 (epoll) |

> **Note**: Due to WSL2 virtualization network forwarding overhead (requests traverse the Windows host's Hyper-V virtual switch) and the test machine being allocated only 4 CPU cores, the metrics (QPS and P99 latency) are expected to improve significantly on a real physical Linux cloud server (e.g., 8 cores / 16 GB).

### Test Scenarios & Commands

- **Test endpoint**: short-link redirect (cache-hit scenario) -> `GET http://localhost:8080/golang`
- **Benchmark command**: `wrk -t4 -c<concurrency> -d10s --latency http://localhost:8080/golang`
- **Parameters**: 4 benchmark threads, 10-second duration, latency statistics enabled.

### Results

By varying the number of concurrent connections (`-c`), the throughput curve is as follows:

| Connections | Peak QPS (Req/Sec) | Avg Latency | P99 Latency (Tail) | System state & bottleneck |
| :---: | :---: | :---: | :---: | :--- |
| **100** | 9,260 | 11.51 ms | 42.77 ms | **Excellent** — the system handles it with ease. |
| **200** | **9,811** | 20.99 ms | 61.23 ms | **Peak throughput** — QPS approaches the 10k mark. |
| **400** | 9,595 | 42.47 ms | 122.17 ms | Slight QPS dip, P99 exceeds 100 ms, CPU starts straining. |
| **800** | 9,699 | 84.73 ms | 257.76 ms | Throughput stays strong, but tail latency rises noticeably — requests begin to queue. |
| **1600** | 9,425 | 173.23 ms | 528.21 ms | Hits the hardware physical limit, P99 exceeds 500 ms, a few Socket Timeouts occur. |

> **Note**: See the `sniplink/test/benchmark` directory for test details and screenshots.


## Project Links

- **GitHub**: [https://github.com/SuperApeA/sniplink](https://github.com/SuperApeA/sniplink)
- **Gitee**: [https://gitee.com/SupermusicA/sniplink](https://gitee.com/SupermusicA/sniplink)

## Author

- **GitHub**: [@SuperApeA](https://github.com/SuperApeA)
- **Gitee**: [@SupermusicA](https://gitee.com/SupermusicA)

Feel free to open an Issue. If this project helps you, please give it a ⭐️ Star!