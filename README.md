# backup-to-oss

一个简单高效的目录备份工具，支持将本地目录压缩后自动上传到阿里云 OSS（对象存储服务）。

## 项目作用与目的

`backup-to-oss` 是一个命令行工具，旨在简化目录备份流程，特别适用于：

- **自动化备份**: 通过命令行或脚本实现自动化备份，无需手动操作
- **多目录备份**: 支持同时备份多个目录，提高备份效率
- **灵活配置**: 支持通过环境变量、配置文件或命令行参数进行配置
- **智能压缩**: 自动将目录压缩为 gzip 格式（.tgz），节省存储空间和传输时间
- **文件过滤**: 支持 glob 模式排除不需要备份的文件或目录（如日志文件、node_modules 等）
- **云端存储**: 自动上传到阿里云 OSS，实现异地备份，保障数据安全
- **IP 标识**: 自动获取公网 IP 并在 OSS 路径中使用，方便区分不同服务器的备份

### 典型使用场景

- 服务器数据定期备份
- 开发环境配置备份
- 数据库备份文件上传
- 日志归档存储
- CI/CD 流程中的文件备份

## 功能特性

- ✅ 支持单个或多个目录备份
- ✅ 支持 glob 模式文件排除（如 `*.log`, `node_modules`, `.git`）
- ✅ 自动压缩为 gzip 格式
- ✅ 自动上传到阿里云 OSS
- ✅ 支持通过 `.env` 文件、环境变量或命令行参数配置
- ✅ 自动获取公网 IP 用于路径标识
- ✅ 详细的日志输出
- ✅ 跨平台支持（Linux、macOS）

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

### 基本用法

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

# 备份配置
DIRS_TO_BACKUP=/path/to/dir1,/path/to/dir2
EXCLUDE_PATTERNS=*.log,node_modules,.git

# 日志配置（可选）
LOG_LEVEL=info
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
- `--log-level, -l`: 日志级别（debug/info/warn/error，默认: info）
- `--log-dir`: 日志文件输出目录（可选）
- `--env-file`: `.env` 配置文件路径（默认: 当前目录下的 `.env`）

### dir 命令参数

- `--path, -p`: 要备份的目录路径，支持多个目录用逗号分隔
- `--exclude, -x`: 排除模式，支持多个模式用逗号分隔，支持 glob 模式

## OSS 路径结构

备份文件在 OSS 中的路径结构：

```
{prefix}/{public_ip}/{date}/{timestamp}_{dir_name}.tgz
```

例如：

```
backups/123.45.67.89/20251217/20251217-143022_home_user_data.tgz
```

- `prefix`: 通过 `--prefix` 或 `OSS_OBJECT_PREFIX` 设置的前缀
- `public_ip`: 自动获取的公网 IP（如果获取失败则省略）
- `date`: 备份日期（YYYYMMDD 格式）
- `timestamp`: 备份时间戳（YYYYMMDD-HHMMSS 格式）
- `dir_name`: 目录路径转换后的名称（斜杠替换为下划线）

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

### 示例 3: 定时备份（crontab）

```bash
# 每天凌晨 2 点备份
0 2 * * * /usr/local/bin/backup-to-oss dir --path /important/data --env-file /etc/backup.env >> /var/log/backup.log 2>&1
```

## 版本信息

查看版本信息：

```bash
backup-to-oss version
```

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
