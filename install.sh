#!/bin/bash

set -e

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# 配置变量
REPO_OWNER="linlanniao"
REPO_NAME="backup-to-oss"
BINARY_NAME="backup-to-oss"
INSTALL_DIR="/usr/local/bin"

# 检测系统信息
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)

# 架构映射
case "$ARCH" in
    x86_64)
        ARCH="amd64"
        ;;
    arm64|aarch64)
        ARCH="arm64"
        ;;
    *)
        echo -e "${RED}错误: 不支持的架构 $ARCH${NC}"
        exit 1
        ;;
esac

# 检查操作系统
if [ "$OS" != "linux" ] && [ "$OS" != "darwin" ]; then
    echo -e "${RED}错误: 不支持的操作系统 $OS${NC}"
    exit 1
fi

echo -e "${GREEN}========================================${NC}"
echo -e "${GREEN}backup-to-oss 一键安装/升级脚本${NC}"
echo -e "${GREEN}========================================${NC}"
echo ""
echo "系统信息: $OS/$ARCH"
echo ""

# 获取最新版本号
echo "正在获取最新版本..."
GITHUB_REPO="${REPO_OWNER}/${REPO_NAME}"
if command -v curl &> /dev/null; then
    LATEST_TAG=$(curl -s "https://api.github.com/repos/${GITHUB_REPO}/releases/latest" | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/')
else
    LATEST_TAG=$(wget -qO- "https://api.github.com/repos/${GITHUB_REPO}/releases/latest" | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/')
fi

if [ -z "$LATEST_TAG" ]; then
    echo -e "${RED}错误: 无法获取最新版本信息${NC}"
    exit 1
fi

echo "最新版本: $LATEST_TAG"
echo ""

# 检查是否已安装
if command -v "$BINARY_NAME" &> /dev/null; then
    CURRENT_VERSION=$($BINARY_NAME version 2>/dev/null | grep "version" | head -1 | awk '{print $2}' || echo "")
    if [ -n "$CURRENT_VERSION" ]; then
        echo "当前已安装版本: $CURRENT_VERSION"
        if [ "$CURRENT_VERSION" = "$LATEST_TAG" ]; then
            echo -e "${GREEN}已是最新版本，无需升级${NC}"
            exit 0
        fi
        echo "将升级到版本: $LATEST_TAG"
    else
        echo "检测到已安装的 $BINARY_NAME，将进行升级"
    fi
    echo ""
fi

# 构建下载 URL
VERSION="${LATEST_TAG#v}"  # 移除 'v' 前缀（如果有）
DOWNLOAD_URL="https://github.com/${GITHUB_REPO}/releases/download/${LATEST_TAG}/${BINARY_NAME}_${VERSION}_${OS}_${ARCH}.tar.gz"

echo "下载地址: $DOWNLOAD_URL"
echo ""

# 创建临时目录
TMP_DIR=$(mktemp -d)
trap "rm -rf $TMP_DIR" EXIT

# 下载并解压
echo "正在下载..."
cd "$TMP_DIR"
if command -v curl &> /dev/null; then
    curl -L -o "${BINARY_NAME}.tar.gz" "$DOWNLOAD_URL"
else
    wget -O "${BINARY_NAME}.tar.gz" "$DOWNLOAD_URL"
fi

echo "正在解压..."
tar -xzf "${BINARY_NAME}.tar.gz"

# 查找二进制文件
BINARY_PATH=""
if [ -f "$BINARY_NAME" ]; then
    BINARY_PATH="$BINARY_NAME"
elif [ -f "${BINARY_NAME}_${VERSION}_${OS}_${ARCH}/${BINARY_NAME}" ]; then
    BINARY_PATH="${BINARY_NAME}_${VERSION}_${OS}_${ARCH}/${BINARY_NAME}"
elif [ -f "${BINARY_NAME}_${OS}_${ARCH}" ]; then
    BINARY_PATH="${BINARY_NAME}_${OS}_${ARCH}"
fi

if [ -z "$BINARY_PATH" ] || [ ! -f "$BINARY_PATH" ]; then
    echo -e "${RED}错误: 无法找到二进制文件${NC}"
    echo "临时目录内容:"
    ls -la "$TMP_DIR"
    exit 1
fi

# 安装二进制文件
echo "正在安装到 $INSTALL_DIR..."
sudo mkdir -p "$INSTALL_DIR"
sudo cp "$BINARY_PATH" "$INSTALL_DIR/$BINARY_NAME"
sudo chmod +x "$INSTALL_DIR/$BINARY_NAME"

# 验证安装
echo ""
echo -e "${GREEN}========================================${NC}"
if command -v "$BINARY_NAME" &> /dev/null; then
    echo -e "${GREEN}安装成功！${NC}"
    echo ""
    echo "版本信息:"
    $BINARY_NAME version
    echo ""
    echo "使用帮助:"
    $BINARY_NAME --help | head -20
else
    echo -e "${RED}安装失败，请检查错误信息${NC}"
    exit 1
fi

echo -e "${GREEN}========================================${NC}"
