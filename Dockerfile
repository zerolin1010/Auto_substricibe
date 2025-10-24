# 构建阶段
FROM golang:1.22-alpine AS builder

# 安装必要工具
RUN apk add --no-cache git make gcc musl-dev

# 设置工作目录
WORKDIR /build

# 复制 go mod 文件
COPY go.mod go.sum ./

# 下载依赖
RUN go mod download

# 复制源代码
COPY . .

# 构建
ARG VERSION=dev
ARG COMMIT=unknown
ARG DATE=unknown

RUN CGO_ENABLED=1 GOOS=linux go build \
    -ldflags "-X main.version=${VERSION} -X main.commit=${COMMIT} -X main.date=${DATE} -s -w" \
    -o syncer \
    ./cmd/syncer

# 运行阶段
FROM alpine:latest

# 安装必要的运行时依赖
RUN apk add --no-cache ca-certificates tzdata

# 设置时区
ENV TZ=Asia/Shanghai

# 创建非 root 用户
RUN addgroup -g 1000 syncer && \
    adduser -D -u 1000 -G syncer syncer

# 设置工作目录
WORKDIR /app

# 从构建阶段复制二进制文件
COPY --from=builder /build/syncer /app/syncer

# 创建数据目录
RUN mkdir -p /app/data && \
    chown -R syncer:syncer /app

# 切换到非 root 用户
USER syncer

# 健康检查（可选）
HEALTHCHECK --interval=5m --timeout=10s --start-period=30s --retries=3 \
    CMD pgrep syncer || exit 1

# 入口点
ENTRYPOINT ["/app/syncer"]

# 默认命令
CMD ["-mode=daemon"]
