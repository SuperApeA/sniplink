# sniplink

A high-concurrency URL shortener written in Go.

## 特性

- 短码生成:Redis INCR 发号 + Base62 编码,DB 唯一索引兜底
- 跳转缓存:Cache-Aside,含空值缓存防穿透
- 访问统计:内存聚合 + 定时批量刷库,不阻塞跳转链路
- 工程化:优雅关闭、健康检查、限流、结构化日志
- 部署:多阶段构建 Docker 镜像,镜像体积 ~20MB

## 架构

[架构图:客户端 -> Gin -> Service -> Redis/MySQL]

详细设计与取舍见 [docs/architecture.md](docs/architecture.md)。

## 性能

压测环境:本机 8 核 / 16GB,MySQL 与 Redis 同机 Docker。

| 场景             | QPS  | P99 延迟 |
| ---------------- | ---- | -------- |
| 跳转(缓存命中)   | XXXX | XX ms    |
| 跳转(缓存未命中) | XXX  | XX ms    |
| 创建短链         | XXX  | XX ms    |

压测命令与调优过程见 [docs/bench.md](docs/bench.md)。

**注意**:表格里的 XXXX 必须是你真跑出来的数字。哪怕只有 3000 QPS,写真实数字 + 标注环境,比虚报一个 5 万更可信,因为面试官会追问"你怎么压的、瓶颈在哪"。

## 快速开始

### 1.开发环境初始化

如果你使用的是全新的 Ubuntu 机器，只需在项目根目录运行以下命令，即可一键初始化 Go 环境、配置国内代理，并安装必要的构建工具（如 make、gcc 等）。

```
bash scripts/setup_go_env.sh
```

#### 2.安装数据库和启动服务

```bash
docker compose up -d
go run ./cmd/server
curl -X POST localhost:8080/api/v1/links -d '{"url":"https://go.dev"}'
```



## 接口测试示例

![短链创建与冲突检测](https://foruda.gitee.com/images/1792778841888796397/e7516498_7824773.png "image-20261023150610260.png")
![短链跳转测试](https://foruda.gitee.com/images/1792778876749411559/3d67d0a6_7824773.png "image-20261023150726460.png")