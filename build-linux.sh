#!/bin/bash

set -e

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# 配置变量
BINARY_NAME="backup-to-oss"
BUILD_DIR="build"
VERSION="${VERSION:-dev}"
REVISION="${REVISION:-$(git rev-parse --short HEAD 2>/dev/null || echo "unknown")}"
BRANCH="${BRANCH:-$(git rev-parse --abbrev-ref HEAD 2>/dev/null || echo "unknown")}"
BUILD_USER="${BUILD_USER:-$(whoami)}"
BUILD_DATE="${BUILD_DATE:-$(date -u +"%Y-%m-%dT%H:%M:%SZ")}"

# 支持的架构
ARCHITECTURES=("amd64" "arm64")

# 显示帮助信息
show_help() {
    cat << EOF
用法: $0 [选项]

选项:
    -a, --arch ARCH     指定架构 (amd64, arm64)，默认构建所有架构
    -v, --version VER   指定版本号，默认从 git tag 或 "dev"
    -o, --output DIR    输出目录，默认为 ./build
    -p, --package       打包为 tar.gz
    -c, --clean         构建前清理输出目录
    -h, --help          显示此帮助信息

示例:
    $0                          # 构建所有架构
    $0 -a amd64                 # 只构建 amd64
    $0 -a amd64 -p              # 构建 amd64 并打包
    $0 -v v1.0.0 -p             # 指定版本并打包
    $0 -c -p                    # 清理后构建并打包

环境变量:
    VERSION      版本号（默认: dev）
    REVISION     Git commit hash（默认: 自动检测）
    BRANCH       Git 分支（默认: 自动检测）
    BUILD_USER   构建用户（默认: 当前用户）
    BUILD_DATE   构建日期（默认: 当前时间）

EOF
}

# 解析命令行参数
ARCH=""
PACKAGE=false
CLEAN=false
OUTPUT_DIR="$BUILD_DIR"

while [[ $# -gt 0 ]]; do
    case $1 in
        -a|--arch)
            ARCH="$2"
            shift 2
            ;;
        -v|--version)
            VERSION="$2"
            shift 2
            ;;
        -o|--output)
            OUTPUT_DIR="$2"
            shift 2
            ;;
        -p|--package)
            PACKAGE=true
            shift
            ;;
        -c|--clean)
            CLEAN=true
            shift
            ;;
        -h|--help)
            show_help
            exit 0
            ;;
        *)
            echo -e "${RED}错误: 未知参数 $1${NC}"
            show_help
            exit 1
            ;;
    esac
done

# 验证架构
if [ -n "$ARCH" ]; then
    if [[ ! " ${ARCHITECTURES[@]} " =~ " ${ARCH} " ]]; then
        echo -e "${RED}错误: 不支持的架构 $ARCH${NC}"
        echo "支持的架构: ${ARCHITECTURES[*]}"
        exit 1
    fi
    ARCHITECTURES=("$ARCH")
fi

# 清理输出目录
if [ "$CLEAN" = true ]; then
    echo -e "${YELLOW}正在清理输出目录...${NC}"
    rm -rf "$OUTPUT_DIR"
fi

# 创建输出目录
mkdir -p "$OUTPUT_DIR"

# 显示构建信息
echo -e "${GREEN}========================================${NC}"
echo -e "${GREEN}构建 Linux 版本${NC}"
echo -e "${GREEN}========================================${NC}"
echo ""
echo "版本信息:"
echo "  Version:   $VERSION"
echo "  Revision:  $REVISION"
echo "  Branch:    $BRANCH"
echo "  BuildUser: $BUILD_USER"
echo "  BuildDate: $BUILD_DATE"
echo ""
echo "目标架构: ${ARCHITECTURES[*]}"
echo "输出目录: $OUTPUT_DIR"
echo ""

# 检查 Go 环境
if ! command -v go &> /dev/null; then
    echo -e "${RED}错误: 未找到 Go 编译器${NC}"
    exit 1
fi

GO_VERSION=$(go version | awk '{print $3}')
echo -e "${BLUE}Go 版本: $GO_VERSION${NC}"
echo ""

# 构建函数
build_for_arch() {
    local arch=$1
    local output_file="${OUTPUT_DIR}/${BINARY_NAME}_linux_${arch}"
    
    echo -e "${GREEN}正在构建 Linux/${arch}...${NC}"
    
    # 构建 ldflags
    local ldflags="-X 'backup-to-oss/pkg/version.Version=${VERSION}'"
    ldflags="${ldflags} -X 'backup-to-oss/pkg/version.Revision=${REVISION}'"
    ldflags="${ldflags} -X 'backup-to-oss/pkg/version.Branch=${BRANCH}'"
    ldflags="${ldflags} -X 'backup-to-oss/pkg/version.BuildUser=${BUILD_USER}'"
    ldflags="${ldflags} -X 'backup-to-oss/pkg/version.BuildDate=${BUILD_DATE}'"
    
    # 交叉编译
    CGO_ENABLED=0 GOOS=linux GOARCH=$arch go build \
        -ldflags "$ldflags" \
        -trimpath \
        -o "$output_file" \
        .
    
    if [ $? -eq 0 ]; then
        # 获取文件大小
        local size=$(du -h "$output_file" | cut -f1)
        echo -e "${GREEN}✓ 构建成功: $output_file (${size})${NC}"
        
        # 打包
        if [ "$PACKAGE" = true ]; then
            package_binary "$arch" "$output_file"
        fi
    else
        echo -e "${RED}✗ 构建失败: Linux/${arch}${NC}"
        return 1
    fi
    echo ""
}

# 打包函数
package_binary() {
    local arch=$1
    local binary_file=$2
    local package_name="${BINARY_NAME}_${VERSION}_linux_${arch}.tar.gz"
    local package_path="${OUTPUT_DIR}/${package_name}"
    
    echo -e "${BLUE}正在打包...${NC}"
    
    # 创建临时目录
    local temp_dir=$(mktemp -d)
    local temp_binary="${temp_dir}/${BINARY_NAME}"
    
    # 复制二进制文件
    cp "$binary_file" "$temp_binary"
    chmod +x "$temp_binary"
    
    # 创建 tar.gz
    cd "$temp_dir"
    tar -czf "$package_path" "$(basename "$temp_binary")"
    cd - > /dev/null
    
    # 清理临时目录
    rm -rf "$temp_dir"
    
    # 获取包大小
    local package_size=$(du -h "$package_path" | cut -f1)
    echo -e "${GREEN}✓ 打包成功: $package_path (${package_size})${NC}"
}

# 构建所有架构
SUCCESS_COUNT=0
FAIL_COUNT=0

for arch in "${ARCHITECTURES[@]}"; do
    if build_for_arch "$arch"; then
        ((SUCCESS_COUNT++))
    else
        ((FAIL_COUNT++))
    fi
done

# 显示构建摘要
echo -e "${GREEN}========================================${NC}"
echo -e "${GREEN}构建完成${NC}"
echo -e "${GREEN}========================================${NC}"
echo ""
echo "成功: $SUCCESS_COUNT"
if [ $FAIL_COUNT -gt 0 ]; then
    echo -e "${RED}失败: $FAIL_COUNT${NC}"
fi
echo ""
echo "输出文件:"
ls -lh "$OUTPUT_DIR" | grep -E "${BINARY_NAME}|total" || true
echo ""

if [ $FAIL_COUNT -eq 0 ]; then
    echo -e "${GREEN}所有构建任务完成！${NC}"
    exit 0
else
    echo -e "${RED}部分构建任务失败${NC}"
    exit 1
fi

