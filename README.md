# sniplink
一个用 Go 语言编写的高并发 URL 短链服务项目‌
A high-concurrency URL shortener written in Go.

## 特性

- 短码生成:Redis INCR 发号 + Base62 编码,DB 唯一索引兜底
- 跳转缓存:Cache-Aside,含空值缓存防穿透
- 访问统计:内存聚合 + 定时批量刷库,不阻塞跳转链路
- 工程化:优雅关闭、健康检查、限流、结构化日志
- 部署:多阶段构建 Docker 镜像,镜像体积 ~20MB

## 架构

```
客户端 ──> Gin ──> Service ──┬─> Redis(短码→长链缓存 + 发号器)
                            └─> MySQL(持久化)
```

## 快速开始

### 1.开发环境初始化

如果你使用的是全新的 Ubuntu 机器，只需在项目根目录运行以下命令，即可一键初始化 Go 环境、配置国内代理，并安装必要的构建工具（如 make、gcc 等）。

```
bash scripts/setup_go_env.sh
```

### 2.安装数据库和启动服务

```bash
make migrate #建表
make docker-up #安装数据库并启动
make run #启动服务
```

### **3.测试接口**

```
curl -X POST localhost:8080/api/v1/links -d '{"url":"https://go.dev"}'
```



## 接口测试示例

![短链创建与冲突检测](https://foruda.gitee.com/images/1792778841888796397/e7516498_7824773.png "image-20261023150610260.png")
![短链跳转测试](https://foruda.gitee.com/images/1792778876749411559/3d67d0a6_7824773.png "image-20261023150726460.png")