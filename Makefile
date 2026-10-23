.PHONY: run build test bench clean tidy docker-up docker-down build-linux

# 默认执行 run
default: run

# 开发时运行
run:
	go run ./cmd/server

# 整理依赖并格式化代码
tidy:
	go mod tidy
	go fmt ./...

# 编译当前系统二进制文件
build:
	go build -o sniplink ./cmd/server

# 编译 Linux 可执行文件（用于生产部署，面试可讲交叉编译）
build-linux:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o sniplink-linux ./cmd/server

# 跑测试并输出覆盖率
test:
	go test -v -cover ./...

# 跑性能压测（结合你之前写的压测脚本）
bench:
	@echo "开始压测跳转接口 http://localhost:8080/1 ..."
	wrk -t4 -c100 -d10s http://localhost:8080/1

# 启动基础依赖服务
docker-up:
	docker compose up -d
	docker ps -a | grep sniplink

# 停止并清理容器
docker-down:
	docker compose down

# 清理编译产物
clean:
	rm -f sniplink sniplink-linux