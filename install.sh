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

# 操作系统映射
case "$OS" in
    linux)
        OS_NAME="linux"
        ;;
    darwin)
        OS_NAME="darwin"
        ;;
    *)
        echo -e "${RED}错误: 不支持的操作系统 $OS${NC}"
        exit 1
        ;;
esac

echo -e "${GREEN}========================================${NC}"
echo -e "${GREEN}backup-to-oss 一键安装/升级脚本${NC}"
echo -e "${GREEN}========================================${NC}"
echo ""
echo "系统信息: $OS_NAME/$ARCH"
echo ""

# 检查必要的命令
check_command() {
    if ! command -v "$1" &> /dev/null; then
        echo -e "${YELLOW}警告: 未找到 $1，正在尝试安装...${NC}"
        return 1
    fi
    return 0
}

# 安装依赖（Linux）
install_deps_linux() {
    if command -v apt-get &> /dev/null; then
        echo "安装必要的依赖..."
        sudo apt-get update -qq
        sudo apt-get install -y wget curl
    elif command -v yum &> /dev/null; then
        echo "安装必要的依赖..."
        sudo yum install -y wget curl
    elif command -v apk &> /dev/null; then
        echo "安装必要的依赖..."
        sudo apk add --no-cache wget curl
    fi
}

# 安装依赖（macOS）
install_deps_darwin() {
    if ! command -v wget &> /dev/null; then
        if command -v brew &> /dev/null; then
            echo "安装 wget..."
            brew install wget
        else
            echo -e "${YELLOW}警告: 未找到 wget，将使用 curl 替代${NC}"
        fi
    fi
}

# 安装依赖
if [ "$OS_NAME" = "linux" ]; then
    if ! check_command wget && ! check_command curl; then
        install_deps_linux
    fi
elif [ "$OS_NAME" = "darwin" ]; then
    install_deps_darwin
fi

# 获取最新版本
echo "正在获取最新版本..."
if command -v curl &> /dev/null; then
    LATEST_TAG=$(curl -s "https://api.github.com/repos/${REPO_OWNER}/${REPO_NAME}/releases/latest" | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/')
else
    LATEST_TAG=$(wget -qO- "https://api.github.com/repos/${REPO_OWNER}/${REPO_NAME}/releases/latest" | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/')
fi

if [ -z "$LATEST_TAG" ]; then
    echo -e "${RED}错误: 无法获取最新版本信息${NC}"
    exit 1
fi

echo "最新版本: $LATEST_TAG"
echo ""

# 检查是否已安装
CURRENT_VERSION=""
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

# 从 GitHub API 获取实际的 release assets
echo "正在查找可用的发布文件..."
ASSETS_JSON=""
if command -v curl &> /dev/null; then
    ASSETS_JSON=$(curl -s "https://api.github.com/repos/${REPO_OWNER}/${REPO_NAME}/releases/tags/${LATEST_TAG}" 2>/dev/null || echo "")
else
    ASSETS_JSON=$(wget -qO- "https://api.github.com/repos/${REPO_OWNER}/${REPO_NAME}/releases/tags/${LATEST_TAG}" 2>/dev/null || echo "")
fi

# 查找匹配的 asset（优先 tar.gz，其次 zip）
FILE_NAME=""
DOWNLOAD_URL=""
if [ -n "$ASSETS_JSON" ] && echo "$ASSETS_JSON" | grep -q '"assets"'; then
    if command -v jq &> /dev/null; then
        # 如果有 jq，使用它来解析 JSON
        FILE_NAME=$(echo "$ASSETS_JSON" | jq -r ".assets[] | select(.name | test(\"${BINARY_NAME}_${VERSION}_${OS_NAME}_${ARCH}\\.(tar\\.gz|zip)\")) | .name" | head -1)
        DOWNLOAD_URL=$(echo "$ASSETS_JSON" | jq -r ".assets[] | select(.name | test(\"${BINARY_NAME}_${VERSION}_${OS_NAME}_${ARCH}\\.(tar\\.gz|zip)\")) | .browser_download_url" | head -1)
    else
        # 如果没有 jq，使用 grep 和 sed
        # 优先查找 tar.gz
        FILE_NAME=$(echo "$ASSETS_JSON" | grep -o "\"name\":\"${BINARY_NAME}_${VERSION}_${OS_NAME}_${ARCH}\\.tar\\.gz\"" | sed 's/.*"name":"\([^"]*\)".*/\1/' | head -1)
        DOWNLOAD_URL=$(echo "$ASSETS_JSON" | grep -A 1 "\"name\":\"${BINARY_NAME}_${VERSION}_${OS_NAME}_${ARCH}\\.tar\\.gz\"" | grep "\"browser_download_url\"" | sed 's/.*"browser_download_url":"\([^"]*\)".*/\1/' | head -1)
        
        # 如果 tar.gz 不存在，尝试 zip
        if [ -z "$FILE_NAME" ]; then
            FILE_NAME=$(echo "$ASSETS_JSON" | grep -o "\"name\":\"${BINARY_NAME}_${VERSION}_${OS_NAME}_${ARCH}\\.zip\"" | sed 's/.*"name":"\([^"]*\)".*/\1/' | head -1)
            DOWNLOAD_URL=$(echo "$ASSETS_JSON" | grep -A 1 "\"name\":\"${BINARY_NAME}_${VERSION}_${OS_NAME}_${ARCH}\\.zip\"" | grep "\"browser_download_url\"" | sed 's/.*"browser_download_url":"\([^"]*\)".*/\1/' | head -1)
        fi
    fi
fi

# 如果仍然找不到，尝试直接构建 URL（向后兼容）
if [ -z "$FILE_NAME" ] || [ -z "$DOWNLOAD_URL" ]; then
    FILE_NAME="${BINARY_NAME}_${VERSION}_${OS_NAME}_${ARCH}.tar.gz"
    DOWNLOAD_URL="https://github.com/${REPO_OWNER}/${REPO_NAME}/releases/download/${LATEST_TAG}/${FILE_NAME}"
    
    # 检查文件是否存在
    HTTP_CODE=$(curl -sI -o /dev/null -w "%{http_code}" "$DOWNLOAD_URL" 2>/dev/null || echo "000")
    if [ "$HTTP_CODE" != "200" ]; then
        FILE_NAME="${BINARY_NAME}_${VERSION}_${OS_NAME}_${ARCH}.zip"
        DOWNLOAD_URL="https://github.com/${REPO_OWNER}/${REPO_NAME}/releases/download/${LATEST_TAG}/${FILE_NAME}"
        HTTP_CODE=$(curl -sI -o /dev/null -w "%{http_code}" "$DOWNLOAD_URL" 2>/dev/null || echo "000")
        if [ "$HTTP_CODE" != "200" ]; then
            echo -e "${RED}错误: 无法找到适用于 $OS_NAME/$ARCH 的发布文件${NC}"
            echo "请访问 https://github.com/${REPO_OWNER}/${REPO_NAME}/releases/tag/${LATEST_TAG} 查看可用的文件"
            exit 1
        fi
    fi
fi

echo "下载地址: $DOWNLOAD_URL"
echo ""

# 创建临时目录
TMP_DIR=$(mktemp -d)
trap "rm -rf $TMP_DIR" EXIT

# 下载文件
echo "正在下载..."
if command -v curl &> /dev/null; then
    HTTP_CODE=$(curl -L -o "$TMP_DIR/$FILE_NAME" -w "%{http_code}" "$DOWNLOAD_URL" 2>/dev/null || echo "000")
    if [ "$HTTP_CODE" != "200" ]; then
        echo -e "${RED}错误: 下载失败 (HTTP $HTTP_CODE)${NC}"
        exit 1
    fi
else
    if ! wget -O "$TMP_DIR/$FILE_NAME" "$DOWNLOAD_URL" 2>/dev/null; then
        echo -e "${RED}错误: 下载失败${NC}"
        exit 1
    fi
fi

# 验证下载的文件
if [ ! -f "$TMP_DIR/$FILE_NAME" ]; then
    echo -e "${RED}错误: 下载失败，文件不存在${NC}"
    exit 1
fi

# 检查文件大小（至少应该大于 100 字节）
FILE_SIZE=$(stat -f%z "$TMP_DIR/$FILE_NAME" 2>/dev/null || stat -c%s "$TMP_DIR/$FILE_NAME" 2>/dev/null || echo "0")
if [ "$FILE_SIZE" -lt 100 ]; then
    echo -e "${RED}错误: 下载的文件大小异常 ($FILE_SIZE 字节)，可能是下载失败${NC}"
    echo "文件内容预览:"
    head -c 200 "$TMP_DIR/$FILE_NAME" 2>/dev/null || true
    echo ""
    exit 1
fi

# 解压文件
echo "正在解压..."
cd "$TMP_DIR"
if [[ "$FILE_NAME" == *.tar.gz ]]; then
    tar -xzf "$FILE_NAME"
elif [[ "$FILE_NAME" == *.zip ]]; then
    unzip -q "$FILE_NAME"
fi

# 查找二进制文件
BINARY_PATH=""
if [ -f "$BINARY_NAME" ]; then
    BINARY_PATH="$BINARY_NAME"
elif [ -f "${BINARY_NAME}_${VERSION}_${OS_NAME}_${ARCH}/${BINARY_NAME}" ]; then
    BINARY_PATH="${BINARY_NAME}_${VERSION}_${OS_NAME}_${ARCH}/${BINARY_NAME}"
elif [ -f "${BINARY_NAME}_${OS_NAME}_${ARCH}" ]; then
    BINARY_PATH="${BINARY_NAME}_${OS_NAME}_${ARCH}"
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

