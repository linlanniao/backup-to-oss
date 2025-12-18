# backup-to-oss

一个简单高效的数据备份工具，支持将本地目录、文件、Consul snapshot 和 etcd snapshot 压缩后自动上传到阿里云 OSS（对象存储服务）。

## 功能特性

- ✅ **目录备份**：支持单个或多个目录备份，支持 glob 模式文件排除
- ✅ **文件备份**：支持单个或多个文件备份
- ✅ **Consul 备份**：支持从 Consul 服务器获取 snapshot 并备份到 OSS
- ✅ **etcd 备份**：支持从 etcd 服务器获取 snapshot 并备份到 OSS（支持 TLS 和认证）
- ✅ **多种压缩方式**：支持 zstd（默认）、gzip 或无压缩
- ✅ **自动上传到 OSS**：备份完成后自动上传到阿里云 OSS
- ✅ **灵活的配置方式**：支持通过 `.env` 文件、环境变量或命令行参数配置
- ✅ **自动获取公网 IP**：用于路径标识，便于区分不同服务器的备份
- ✅ **详细的日志输出**：支持不同日志级别和日志文件输出
- ✅ **保留备份文件选项**：可选择是否保留本地备份文件
- ✅ **跨平台支持**：支持 Linux、macOS

## 一键安装/升级

### 快速安装

```bash
curl -fsSL https://raw.githubusercontent.com/linlanniao/backup-to-oss/main/install.sh | bash
```

或者下载脚本后执行：

```bash
wget https://raw.githubusercontent.com/linlanniao/backup-to-oss/main/install.sh
chmod +x install.sh
sudo ./install.sh
```

### 安装脚本说明

安装脚本会自动：

1. 检测系统架构（amd64/arm64）和操作系统（Linux/macOS）
2. 从 GitHub Releases 下载最新版本
3. 安装到 `/usr/local/bin/backup-to-oss`
4. 验证安装是否成功

### 手动安装

如果一键安装脚本无法使用，也可以手动下载二进制文件：

1. 访问 [Releases 页面](https://github.com/linlanniao/backup-to-oss/releases)
2. 下载对应系统的二进制文件
3. 解压并移动到 PATH 目录：

```bash
# Linux
tar -xzf backup-to-oss_*_linux_amd64.tar.gz
sudo mv backup-to-oss /usr/local/bin/

# macOS
tar -xzf backup-to-oss_*_darwin_amd64.tar.gz
sudo mv backup-to-oss /usr/local/bin/
```

## 使用方法

### 目录备份 (dir)

```bash
# 备份单个目录
backup-to-oss dir --path /path/to/directory \
  --endpoint oss-cn-hangzhou.aliyuncs.com \
  --access-key YOUR_ACCESS_KEY \
  --secret-key YOUR_SECRET_KEY \
  --bucket your-bucket-name

# 备份多个目录
backup-to-oss dir --path /path/to/dir1,/path/to/dir2 \
  --endpoint oss-cn-hangzhou.aliyuncs.com \
  --access-key YOUR_ACCESS_KEY \
  --secret-key YOUR_SECRET_KEY \
  --bucket your-bucket-name

# 排除特定文件或目录
backup-to-oss dir --path /path/to/directory \
  --exclude "*.log,node_modules,.git" \
  --endpoint oss-cn-hangzhou.aliyuncs.com \
  --access-key YOUR_ACCESS_KEY \
  --secret-key YOUR_SECRET_KEY \
  --bucket your-bucket-name

# 指定压缩方式（zstd/gzip/none）
backup-to-oss dir --path /path/to/directory \
  --compress gzip \
  --endpoint oss-cn-hangzhou.aliyuncs.com \
  --access-key YOUR_ACCESS_KEY \
  --secret-key YOUR_SECRET_KEY \
  --bucket your-bucket-name
```

### 文件备份 (file)

```bash
# 备份单个文件
backup-to-oss file --path /path/to/file.txt \
  --endpoint oss-cn-hangzhou.aliyuncs.com \
  --access-key YOUR_ACCESS_KEY \
  --secret-key YOUR_SECRET_KEY \
  --bucket your-bucket-name

# 备份多个文件
backup-to-oss file --path /path/to/file1.txt,/path/to/file2.txt \
  --endpoint oss-cn-hangzhou.aliyuncs.com \
  --access-key YOUR_ACCESS_KEY \
  --secret-key YOUR_SECRET_KEY \
  --bucket your-bucket-name
```

### Consul 备份 (consul)

```bash
# 基本用法（使用默认地址 http://127.0.0.1:8500）
backup-to-oss consul \
  --endpoint oss-cn-hangzhou.aliyuncs.com \
  --access-key YOUR_ACCESS_KEY \
  --secret-key YOUR_SECRET_KEY \
  --bucket your-bucket-name

# 指定 Consul 地址
backup-to-oss consul --address http://localhost:8500 \
  --endpoint oss-cn-hangzhou.aliyuncs.com \
  --access-key YOUR_ACCESS_KEY \
  --secret-key YOUR_SECRET_KEY \
  --bucket your-bucket-name

# 使用 ACL Token
backup-to-oss consul --address http://localhost:8500 \
  --token your-consul-token \
  --endpoint oss-cn-hangzhou.aliyuncs.com \
  --access-key YOUR_ACCESS_KEY \
  --secret-key YOUR_SECRET_KEY \
  --bucket your-bucket-name

# 允许从非 leader 节点获取快照
backup-to-oss consul --address http://localhost:8500 \
  --stale \
  --endpoint oss-cn-hangzhou.aliyuncs.com \
  --access-key YOUR_ACCESS_KEY \
  --secret-key YOUR_SECRET_KEY \
  --bucket your-bucket-name
```

### etcd 备份 (etcd)

```bash
# 基本用法（使用默认地址 http://127.0.0.1:2379）
backup-to-oss etcd \
  --endpoint oss-cn-hangzhou.aliyuncs.com \
  --access-key YOUR_ACCESS_KEY \
  --secret-key YOUR_SECRET_KEY \
  --bucket your-bucket-name

# 指定 etcd 端点（支持多个，逗号分隔）
backup-to-oss etcd --etcd-endpoints http://127.0.0.1:2379,http://127.0.0.2:2379 \
  --endpoint oss-cn-hangzhou.aliyuncs.com \
  --access-key YOUR_ACCESS_KEY \
  --secret-key YOUR_SECRET_KEY \
  --bucket your-bucket-name

# 使用 TLS 证书
backup-to-oss etcd --etcd-endpoints https://127.0.0.1:2379 \
  --cacert /etc/etcd/ca.crt \
  --cert /etc/etcd/etcd.crt \
  --key /etc/etcd/etcd.key \
  --endpoint oss-cn-hangzhou.aliyuncs.com \
  --access-key YOUR_ACCESS_KEY \
  --secret-key YOUR_SECRET_KEY \
  --bucket your-bucket-name

# 使用用户名密码认证
backup-to-oss etcd --etcd-endpoints http://127.0.0.1:2379 \
  --user root \
  --password password123 \
  --endpoint oss-cn-hangzhou.aliyuncs.com \
  --access-key YOUR_ACCESS_KEY \
  --secret-key YOUR_SECRET_KEY \
  --bucket your-bucket-name

# 设置超时时间
backup-to-oss etcd --etcd-endpoints http://127.0.0.1:2379 \
  --dial-timeout 20s \
  --command-timeout 60s \
  --endpoint oss-cn-hangzhou.aliyuncs.com \
  --access-key YOUR_ACCESS_KEY \
  --secret-key YOUR_SECRET_KEY \
  --bucket your-bucket-name
```

### 使用配置文件

创建 `.env` 文件：

```env
# OSS 配置
OSS_ENDPOINT=oss-cn-hangzhou.aliyuncs.com
OSS_ACCESS_KEY=your_access_key
OSS_SECRET_KEY=your_secret_key
OSS_BUCKET=your-bucket-name
OSS_OBJECT_PREFIX=backups/

# 目录备份配置
DIRS_TO_BACKUP=/path/to/dir1,/path/to/dir2
EXCLUDE_PATTERNS=*.log,node_modules,.git

# 文件备份配置
FILES_TO_BACKUP=/path/to/file1.txt,/path/to/file2.txt

# Consul 配置
CONSUL_ADDRESS=http://127.0.0.1:8500
CONSUL_TOKEN=your-consul-token
CONSUL_STALE=false

# etcd 配置
ETCD_ENDPOINTS=http://127.0.0.1:2379
ETCD_CACERT=/etc/etcd/ca.crt
ETCD_CERT=/etc/etcd/etcd.crt
ETCD_KEY=/etc/etcd/etcd.key
ETCD_USER=root
ETCD_PASSWORD=password123
ETCD_DIAL_TIMEOUT=5s
ETCD_COMMAND_TIMEOUT=60s

# 压缩配置
COMPRESS_METHOD=zstd  # zstd/gzip/none，默认为 zstd

# 备份文件保留配置
KEEP_BACKUP_FILES=false  # 是否保留备份文件，默认为 false

# 日志配置（可选）
LOG_LEVEL=info  # debug/info/warn/error，默认为 info
LOG_DIR=/var/log/backup-to-oss
```

然后直接运行：

```bash
backup-to-oss dir --path /path/to/directory
```

### 使用环境变量

```bash
export OSS_ENDPOINT=oss-cn-hangzhou.aliyuncs.com
export OSS_ACCESS_KEY=your_access_key
export OSS_SECRET_KEY=your_secret_key
export OSS_BUCKET=your-bucket-name

backup-to-oss dir --path /path/to/directory
```

### 配置优先级

配置优先级从高到低：

1. 命令行参数
2. 环境变量
3. `.env` 文件

## 命令行参数

### 全局参数

- `--endpoint, -e`: OSS 端点地址（如 `oss-cn-hangzhou.aliyuncs.com`）
- `--access-key, -a`: OSS AccessKey
- `--secret-key, -s`: OSS SecretKey
- `--bucket, -b`: OSS 存储桶名称
- `--prefix`: OSS 对象前缀（可选，默认为时间戳）
- `--compress, -c`: 压缩方式（zstd/gzip/none，默认: zstd）
- `--keep-backup-files`: 保留备份文件（打包压缩后的文件），不上传到 OSS 后删除
- `--log-level, -l`: 日志级别（debug/info/warn/error，默认: info）
- `--log-dir`: 日志文件输出目录（可选）
- `--env-file`: `.env` 配置文件路径（默认: 当前目录下的 `.env`）

### dir 命令参数

- `--path, -p`: 要备份的目录路径，支持多个目录用逗号分隔
- `--exclude, -x`: 排除模式，支持多个模式用逗号分隔，支持 glob 模式

### file 命令参数

- `--path, -p`: 要备份的文件路径，支持多个文件用逗号分隔

### consul 命令参数

- `--address`: Consul 服务器地址（默认: http://127.0.0.1:8500）
- `--token`: Consul ACL Token（可选）
- `--stale`: 允许从非 leader 节点获取快照（设置为 true 时允许）

### etcd 命令参数

- `--etcd-endpoints`: etcd 服务器地址列表，多个地址用逗号分隔（默认: http://127.0.0.1:2379）
- `--cacert`: CA 证书文件路径（可选，用于 TLS）
- `--cert`: 客户端证书文件路径（可选，用于 TLS）
- `--key`: 客户端私钥文件路径（可选，用于 TLS）
- `--user`: etcd 用户名（可选）
- `--password`: etcd 密码（可选）
- `--dial-timeout`: 连接超时时间（如 20s，默认: 5s）
- `--command-timeout`: 命令超时时间（如 60s，默认无超时）

## OSS 路径结构

备份文件在 OSS 中的路径结构：

### 目录和文件备份

```
{prefix}/{public_ip}/{date}/{timestamp}_{name}.{ext}
```

例如：

```
backups/123.45.67.89/20251217/20251217-143022_home_user_data.zst
backups/123.45.67.89/20251217/20251217-143022_file_txt.zst
```

### Consul 备份

```
{prefix}/{public_ip}/{date}/{timestamp}_consul.snap.{ext}
```

例如：

```
backups/123.45.67.89/20251217/20251217-143022_consul.snap.zst
```

### etcd 备份

```
{prefix}/{public_ip}/{date}/{timestamp}_etcd.snap.{ext}
```

例如：

```
backups/123.45.67.89/20251217/20251217-143022_etcd.snap.zst
```

**路径说明：**

- `prefix`: 通过 `--prefix` 或 `OSS_OBJECT_PREFIX` 设置的前缀
- `public_ip`: 自动获取的公网 IP（如果获取失败则省略）
- `date`: 备份日期（YYYYMMDD 格式）
- `timestamp`: 备份时间戳（YYYYMMDD-HHMMSS 格式）
- `name`: 目录/文件路径转换后的名称（斜杠替换为下划线）
- `ext`: 压缩文件扩展名（zst 表示 zstd，gz 表示 gzip，snap 表示无压缩）

## 示例

### 示例 1: 备份网站目录

```bash
backup-to-oss dir \
  --path /var/www/html \
  --exclude "*.log,*.tmp,node_modules" \
  --endpoint oss-cn-hangzhou.aliyuncs.com \
  --access-key YOUR_KEY \
  --secret-key YOUR_SECRET \
  --bucket website-backups
```

### 示例 2: 使用配置文件备份多个目录

`.env` 文件：

```env
OSS_ENDPOINT=oss-cn-hangzhou.aliyuncs.com
OSS_ACCESS_KEY=your_key
OSS_SECRET_KEY=your_secret
OSS_BUCKET=backups
OSS_OBJECT_PREFIX=server-backups/
DIRS_TO_BACKUP=/etc/nginx,/var/log,/home/user/data
EXCLUDE_PATTERNS=*.log,*.tmp
```

执行：

```bash
backup-to-oss dir
```

### 示例 3: 备份 Consul 数据

```bash
backup-to-oss consul \
  --address http://consul-server:8500 \
  --token your-token \
  --endpoint oss-cn-hangzhou.aliyuncs.com \
  --access-key YOUR_KEY \
  --secret-key YOUR_SECRET \
  --bucket consul-backups
```

### 示例 4: 备份 etcd 数据（使用 TLS）

```bash
backup-to-oss etcd \
  --etcd-endpoints https://etcd1:2379,https://etcd2:2379,https://etcd3:2379 \
  --cacert /etc/etcd/ca.crt \
  --cert /etc/etcd/etcd.crt \
  --key /etc/etcd/etcd.key \
  --endpoint oss-cn-hangzhou.aliyuncs.com \
  --access-key YOUR_KEY \
  --secret-key YOUR_SECRET \
  --bucket etcd-backups
```

### 示例 5: 定时备份（crontab）

```bash
# 每天凌晨 2 点备份目录
0 2 * * * /usr/local/bin/backup-to-oss dir --path /important/data --env-file /etc/backup.env --log-dir /var/log/

# 每小时备份 Consul
0 * * * * /usr/local/bin/backup-to-oss consul --env-file /etc/backup.env --log-dir /var/log/

# 每天凌晨 3 点备份 etcd
0 3 * * * /usr/local/bin/backup-to-oss etcd --env-file /etc/backup.env --log-dir /var/log/
```

## 版本信息

查看版本信息：

```bash
backup-to-oss version
# 或
backup-to-oss v
```

## 压缩方式说明

工具支持三种压缩方式：

- **zstd**（默认）：压缩率高，速度快，推荐使用
- **gzip**：兼容性好，压缩率中等
- **none**：不压缩，直接上传原始文件

可以通过 `--compress` 参数或 `COMPRESS_METHOD` 环境变量指定压缩方式。

## 开发

### 构建

```bash
go build -o backup-to-oss .
```

### 运行测试

```bash
go test ./...
```

## 许可证

见 [LICENSE](LICENSE) 文件

## 贡献

欢迎提交 Issue 和 Pull Request！
