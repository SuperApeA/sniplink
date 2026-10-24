# sniplink
一个用 Go 语言编写的高并发 URL 短链服务项目‌。

[![Go Version](https://img.shields.io/badge/Go-1.22+-00ADD8?style=flat-square&logo=go)](https://go.dev)
[![Framework](https://img.shields.io/badge/Framework-Gin-008080?style=flat-square)](https://github.com/gin-gonic/gin)
[![ORM](https://img.shields.io/badge/ORM-GORM-1B9AAA?style=flat-square)](https://gorm.io)
[![Database](https://img.shields.io/badge/Database-MySQL-4479A1?style=flat-square&logo=mysql&logoColor=white)](https://www.mysql.com)
[![Cache](https://img.shields.io/badge/Cache-Redis-DC382D?style=flat-square&logo=redis&logoColor=white)](https://redis.io)
[![Deploy](https://img.shields.io/badge/Deploy-Docker-2496ED?style=flat-square&logo=docker&logoColor=white)](https://www.docker.com)

## 特性

- **短码生成**：Redis `INCR` 发号 + Base62 编码，DB 唯一索引兜底，天然无冲突。
- **跳转缓存**：Cache-Aside 模式，配合空值缓存防止缓存穿透。
- **访问统计**：内存聚合 + 定时批量刷库，拒绝每次跳转写 DB，保证主链路性能。
- **工程化**：优雅关闭（Graceful Shutdown）、健康检查、动态限流、配置化驱动。
- **部署**：多阶段构建 Docker 镜像，极小体积，开箱即用。

## 关键设计点

| 问题           | 方案                                                         |
| :------------- | :----------------------------------------------------------- |
| **短码生成**   | Redis `INCR` 发号器 + Base62 编码，无碰撞重试压力            |
| **缓存穿透**   | 空值缓存（60s 过期）+ DB 唯一索引兜底                        |
| **跳转状态码** | 采用 **302** 临时重定向，便于精准统计点击量；若追求极致性能可换 301 |
| **访问统计**   | 内存 Channel 批量聚合，定时刷入 MySQL，避免行锁冲突          |
| **路由冲突**   | 短链走 Gin 的 `NoRoute` 兜底，API 走 `/api/v1` 显式注册，彻底避免通配符与静态路由冲突 |

## 架构

```text
客户端 ──> Gin ──> Service ──┬─> Redis (短码→长链缓存 + 发号器)
                             └─> MySQL (持久化)
```

## 项目目录结构

```
sniplink/
├── cmd/server/         # 程序入口，依赖注入与生命周期管理
├── configs/            # 配置文件模板 (config.example.yaml)
├── internal/           # 内部业务逻辑（禁止外部 import）
│   ├── cache/          # Redis 缓存封装
│   ├── config/         # Viper 配置加载与解析
│   ├── handler/        # HTTP 控制器 (Gin)
│   ├── idgen/          # 基于 Redis 的分布式发号器
│   ├── middleware/     # 限流等中间件
│   ├── model/          # GORM 数据库模型
│   ├── repository/     # MySQL 数据访问层 (DAO)
│   ├── router/         # 路由注册与装配
│   └── service/        # 核心业务逻辑
├── scripts/            # 环境初始化脚本 (setup_go_env.sh)
├── docker-compose.yml  # 本地开发中间件编排 (MySQL/Redis)
└── Makefile            # 自动化构建与运行脚本
```

## 快速开始

### 1. 开发环境初始化 (Ubuntu)

如果是全新的 Ubuntu 机器，一键初始化 Go 环境、国内代理及构建工具：

```
bash scripts/setup_go_env.sh
```

### 2. 启动依赖与运行服务

```
make docker-up  # 启动 MySQL 和 Redis 容器
make migrate #建表
make run        # 启动 sniplink 服务 (依赖 go run)
```

*注：默认开启 GORM `AutoMigrate` 自动建表。生产环境建议关闭，使用 `make migrate` 配合 SQL 脚本手动建表。*

### 3. 测试接口

**创建短链：**

```
curl -X POST http://localhost:8080/api/v1/links \
  -H "Content-Type: application/json" \
  -d '{"url":"https://go.dev", "custom_code":"golang"}'
```

**测试跳转 (浏览器或 curl)：**
访问 `http://localhost:8080/golang`，将返回 `302 Found` 并跳转至 `https://go.dev`。

## 配置管理

本项目采用 **Viper** 进行配置管理：
1. `configs/config.yaml` 文件配置（本地开发）
2. **环境变量覆盖**（生产部署）

生产环境可通过注入环境变量动态替换配置（如 `MYSQL_DSN`, `REDIS_ADDR` 等），实现镜像复用与敏感信息隔离。


## 接口测试示例

![短链创建与冲突检测](https://foruda.gitee.com/images/1792778841888796397/e7516498_7824773.png "image-20261023150610260.png")
![短链跳转测试](https://foruda.gitee.com/images/1792778876749411559/3d67d0a6_7824773.png "image-20261023150726460.png")

## 性能压测 (Benchmark)

为了验证系统在高并发场景下的吞吐能力与稳定性，本项目使用 `wrk` 进行了多维度的压力测试，并针对不同并发数进行了瓶颈分析。

### 压测环境

为了确保数据的真实性与可复现性，本次压测在本地 WSL2 虚拟化环境下进行：

| 组件           | 配置 / 版本信息                                           |
| :------------- | :-------------------------------------------------------- |
| **CPU**        | Intel(R) Core(TM) i5-8300H CPU @ 2.30GHz (WSL2 分配 4 核) |
| **内存**       | 7.8 GiB (WSL2 分配)                                       |
| **操作系统**   | Ubuntu 24.04.5 LTS                                        |
| **Linux 内核** | 6.18.33.2-microsoft-standard-WSL2                         |
| **Go 版本**    | go1.22.2 linux/amd64                                      |
| **Docker**     | Docker version 29.8.2                                     |
| **MySQL**      | MySQL 8.0.46 (Docker 部署)                                |
| **Redis**      | Redis server v=7.4.11 (Docker 部署)                       |
| **压测工具**   | wrk debian/4.1.0-4build2 (epoll)                          |

> **注**：受限于 WSL2 虚拟化网络转发开销（请求需经过 Windows 宿主机的 Hyper-V 虚拟交换机），以及测试机仅分配 4 核 CPU。在真实的物理 Linux 云服务器（如 8核16G）环境下，各项指标（QPS 与 P99 延迟）预计会显著提升。

### 测试场景与命令

* **测试接口**：短链跳转 (缓存命中场景) -> `GET http://localhost:8080/golang`
* **压测命令**：`wrk -t4 -c<并发数> -d10s --latency http://localhost:8080/golang`
* **参数说明**：4 个压测线程，持续 10 秒，开启延迟统计。

### 测试结果

通过调整不同并发连接数（`-c`），得出系统的吞吐量曲线如下：

| 并发数 (Connections) | 峰值 QPS (Req/Sec) | 平均延迟 (Avg Latency) | P99 延迟 (Tail Latency) | 系统状态与瓶颈表现                                          |
| :------------------: | :----------------: | :--------------------: | :---------------------: | :---------------------------------------------------------- |
|       **100**        |       9,260        |        11.51 ms        |        42.77 ms         | **表现极佳**，系统游刃有余。                                |
|       **200**        |     **9,811**      |        20.99 ms        |        61.23 ms         | **吞吐量巅峰**，QPS 逼近一万大关。                          |
|       **400**        |       9,595        |        42.47 ms        |        122.17 ms        | QPS 微降，P99 破 100ms，CPU 开始吃力。                      |
|       **800**        |       9,699        |        84.73 ms        |        257.76 ms        | 吞吐量依然坚挺，但长尾延迟明显，请求开始排队。              |
|       **1600**       |       9,425        |       173.23 ms        |        528.21 ms        | 达到硬件物理极限，P99 突破 500ms，出现少量 Socket Timeout。 |
> **注**：测试截图查看sniplink/test