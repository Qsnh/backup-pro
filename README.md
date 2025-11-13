# BackupPro - S3 自动备份工具

基于 Golang 实现的自动备份工具，将指定目录打包压缩为 tar.gz 格式并上传到 S3，支持定时备份、版本管理、校验和验证以及通知功能。

## 功能特性

- **自动备份**：将指定目录打包为 tar.gz 格式并上传到 S3
- **版本管理**：支持设置保留最多 N 份备份，自动清理旧备份
- **定时任务**：内置 Cron 定时调度，支持灵活的定时规则
- **校验验证**：自动计算 SHA256 校验和，确保备份完整性
- **通知系统**：支持邮件和 Webhook 通知备份结果
- **日志记录**：详细的操作日志和错误记录
- **灵活配置**：基于 YAML 配置文件，易于管理
- **S3 兼容**：支持 AWS S3 和兼容 S3 的服务（如 MinIO）

## 安装

### 从源码构建

```bash
# 克隆仓库
git clone https://github.com/Qsnh/backup-pro.git
cd backup-pro

# 安装依赖
go mod tidy

# 构建
go build -o backup-pro .

# 可选：安装到系统路径
sudo mv backup-pro /usr/local/bin/
```

### 下载预编译二进制（即将提供）

```bash
# 下载最新版本
wget https://github.com/Qsnh/backup-pro/releases/latest/download/backup-pro-linux-amd64

# 添加执行权限
chmod +x backup-pro-linux-amd64
sudo mv backup-pro-linux-amd64 /usr/local/bin/backup-pro
```

## 配置

### 1. 创建配置文件

复制示例配置文件并根据实际情况修改：

```bash
cp config.yaml.example config.yaml
```

### 2. 配置说明

```yaml
# 备份配置
backup:
  source_dir: "/path/to/your/data"     # 要备份的目录
  temp_dir: "/tmp/backup-pro"              # 临时目录
  retention_count: 7                   # 保留最新的 7 个备份
  name_prefix: "backup"                # 备份文件名前缀

# S3 配置
s3:
  endpoint: ""                         # 自定义端点（MinIO 等）
  region: "us-east-1"                  # AWS 区域
  bucket: "my-backup-bucket"           # S3 桶名
  access_key_id: "YOUR_ACCESS_KEY"     # 访问密钥 ID
  secret_access_key: "YOUR_SECRET_KEY" # 访问密钥
  prefix: "backups/"                   # S3 对象前缀
  use_path_style: false                # 路径样式（MinIO 需要 true）

# 定时任务配置
schedule:
  enabled: true                        # 启用定时任务
  cron_expr: "0 0 2 * * *"            # 每天凌晨 2:00 执行
  run_at_start: false                  # 启动时立即执行

# 通知配置
notification:
  enabled: true                        # 启用通知
  email:
    enabled: true                      # 启用邮件通知
    smtp_host: "smtp.example.com"
    smtp_port: 587
    username: "your-email@example.com"
    password: "your-password"
    from: "backup@example.com"
    to:
      - "admin@example.com"
  webhook:
    enabled: true                      # 启用 Webhook 通知
    url: "https://your-webhook-url.com/notify"
    method: "POST"

# 日志配置
logging:
  level: "info"                        # 日志级别
  output: "stdout"                     # 输出方式
  file: "/var/log/backup-pro/backup-pro.log"  # 日志文件
```

## 使用方法

### 1. 定时备份模式（默认）

启动程序后会根据配置的 Cron 表达式自动执行备份：

```bash
backup-pro -config config.yaml
```

### 2. 单次备份模式

立即执行一次备份后退出：

```bash
backup-pro -config config.yaml -once
```

### 3. 使用 systemd 管理（推荐）

创建 systemd 服务文件 `/etc/systemd/system/backup-pro.service`：

```ini
[Unit]
Description=backup-pro S3 Backup Service
After=network.target

[Service]
Type=simple
User=root
WorkingDirectory=/opt/backup-pro
ExecStart=/usr/local/bin/backup-pro -config /opt/backup-pro/config.yaml
Restart=always
RestartSec=10

[Install]
WantedBy=multi-user.target
```

启动服务：

```bash
sudo systemctl daemon-reload
sudo systemctl enable backup-pro
sudo systemctl start backup-pro
sudo systemctl status backup-pro
```

查看日志：

```bash
sudo journalctl -u backup-pro -f
```

### 4. 使用 Docker 运行

创建 `Dockerfile`：

```dockerfile
FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY . .
RUN go mod download
RUN go build -o backup-pro .

FROM alpine:latest
RUN apk --no-cache add ca-certificates tzdata
WORKDIR /root/
COPY --from=builder /app/backup-pro .
COPY config.yaml .
CMD ["./backup-pro", "-config", "config.yaml"]
```

构建并运行：

```bash
docker build -t backup-pro .
docker run -d --name backup-pro \
  -v /path/to/config.yaml:/root/config.yaml \
  -v /path/to/backup/data:/data:ro \
  backup-pro
```

## Cron 表达式说明

backup-pro 使用支持秒级精度的 Cron 表达式，格式为：

```
秒 分 时 日 月 星期
```

常用示例：

- `0 0 2 * * *` - 每天凌晨 2:00
- `0 0 */6 * * *` - 每 6 小时
- `0 30 1 * * 0` - 每周日凌晨 1:30
- `0 0 0 1 * *` - 每月 1 号凌晨 0:00
- `0 */30 * * * *` - 每 30 分钟

## S3 兼容服务配置

### AWS S3

```yaml
s3:
  region: "us-east-1"
  bucket: "my-backup-bucket"
  access_key_id: "YOUR_ACCESS_KEY"
  secret_access_key: "YOUR_SECRET_KEY"
  use_path_style: false
```

### MinIO

```yaml
s3:
  endpoint: "http://minio.example.com:9000"
  region: "us-east-1"
  bucket: "my-backup-bucket"
  access_key_id: "minioadmin"
  secret_access_key: "minioadmin"
  use_path_style: true
```

### Alibaba Cloud OSS

```yaml
s3:
  endpoint: "https://oss-cn-hangzhou.aliyuncs.com"
  region: "oss-cn-hangzhou"
  bucket: "my-backup-bucket"
  access_key_id: "YOUR_ACCESS_KEY"
  secret_access_key: "YOUR_SECRET_KEY"
  use_path_style: false
```

## 通知配置

### 邮件通知

支持标准 SMTP 协议：

```yaml
notification:
  email:
    enabled: true
    smtp_host: "smtp.gmail.com"
    smtp_port: 587
    username: "your-email@gmail.com"
    password: "your-app-password"
    from: "backup@example.com"
    to:
      - "admin@example.com"
```

### Webhook 通知

发送 JSON 格式的 POST 请求：

```json
{
  "success": true,
  "title": "备份成功",
  "message": "备份已成功上传到 S3: backup_20250113_020000.tar.gz",
  "file_name": "backup_20250113_020000.tar.gz",
  "file_size": 1024000,
  "checksum": "abc123...",
  "time": "2025-01-13T02:00:00Z"
}
```

## 故障排查

### 1. 连接 S3 失败

检查 S3 配置是否正确，网络连接是否正常：

```bash
# 测试 AWS CLI 连接
aws s3 ls s3://your-bucket --region us-east-1
```

### 2. 权限错误

确保 S3 访问密钥具有以下权限：
- `s3:PutObject` - 上传对象
- `s3:ListBucket` - 列出桶内容
- `s3:DeleteObject` - 删除旧备份

### 3. 备份文件过大

如果备份文件过大导致上传超时，可以：
- 增加网络超时时间
- 分割大文件
- 使用 S3 分段上传（未来版本支持）

### 4. 查看详细日志

设置日志级别为 `debug`：

```yaml
logging:
  level: "debug"
```

## 开发

### 项目结构

```
backup-pro/
├── main.go              # 主程序入口
├── config/
│   └── config.go        # 配置管理
├── backup/
│   ├── archiver.go      # tar.gz 打包
│   └── uploader.go      # S3 上传
├── scheduler/
│   └── scheduler.go     # Cron 调度
├── notification/
│   └── notifier.go      # 通知功能
└── logger/
    └── logger.go        # 日志系统
```

### 运行测试

```bash
go test ./...
```

### 贡献

欢迎提交 Issue 和 Pull Request！

## 许可证

MIT License

## 作者

tengteng

## 更新日志

### v1.0.0 (2025-01-13)

- 初始版本发布
- 支持 tar.gz 压缩备份
- S3 上传和版本管理
- 定时任务调度
- 邮件和 Webhook 通知
- SHA256 校验和验证
