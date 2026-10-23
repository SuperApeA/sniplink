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