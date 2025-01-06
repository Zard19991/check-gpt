#!/bin/bash

# 版本和仓库信息
REPO="go-coders/check-gpt"
DEFAULT_VERSION="v0.1.7"

# 设置颜色
if [ -t 1 ]; then
    # 终端支持颜色
    GREEN='\033[0;32m'
    RED='\033[0;31m'
    YELLOW='\033[1;33m'
    NC='\033[0m' # No Color
else
    # 非终端或不支持颜色
    GREEN=''
    RED=''
    YELLOW=''
    NC=''
fi

# 安装目录
INSTALL_DIR=""

# 打印消息
print_message() {
    echo -e "${GREEN}$1${NC}"
    # 打印空行
    echo ""
}

print_error() {
    echo -e "${RED}错误: $1${NC}" >&2
}

print_warning() {
    echo -e "${YELLOW}警告: $1${NC}" >&2
}

# 获取最新版本号
get_latest_version() {

    VERSION=$(curl -s "https://api.github.com/repos/${REPO}/releases/latest" | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/')
    if [ -z "$VERSION" ]; then
        print_warning "无法获取最新版本，使用默认版本 ${DEFAULT_VERSION}"
        VERSION=${DEFAULT_VERSION}
    fi
    print_message "获取最新版本: $VERSION"
}

# 获取系统信息
get_system_info() {
    # 检测操作系统
    case "$(uname -s)" in
        Darwin*)
            OS="darwin"
            ;;
        Linux*)
            OS="linux"
            ;;
        MINGW*|MSYS*|CYGWIN*)
            OS="windows"
            ;;
        *)
            print_error "不支持的操作系统: $(uname -s)"
            exit 1
            ;;
    esac
    
    # 检测架构
    case "$(uname -m)" in
        x86_64|amd64)
            ARCH="amd64"
            ;;
        arm64|aarch64)
            ARCH="arm64"
            ;;
        *)
            print_error "不支持的架构: $(uname -m)"
            exit 1
            ;;
    esac

}

# 检查程序是否已安装并处理更新
check_and_install() {
    get_system_info
    
    if [ "$OS" = "windows" ]; then
        BINARY_NAME="check-gpt.exe"
    else
        BINARY_NAME="check-gpt"
    fi

    # 检查常见安装位置
    COMMON_PATHS=(
        "/usr/local/bin/$BINARY_NAME"
        "$HOME/bin/$BINARY_NAME"
        "/usr/bin/$BINARY_NAME"
    )

    INSTALLED_PATH=""
    for path in "${COMMON_PATHS[@]}"; do
        if [ -x "$path" ]; then
            INSTALLED_PATH="$path"
            INSTALL_DIR=$(dirname "$path")  # 设置全局变量
            break
        fi
    done

    # 获取最新版本
    get_latest_version

    if [ -n "$INSTALLED_PATH" ]; then
        # 获取已安装版本
        CURRENT_VERSION=$("$INSTALLED_PATH" -version 2>/dev/null)
        if [ $? -ne 0 ]; then
            CURRENT_VERSION="v0.0.0"
        fi
        
        # 标准化版本号格式（确保都有 v 前缀）
        if [[ ! "$CURRENT_VERSION" =~ ^v ]]; then
            CURRENT_VERSION="v${CURRENT_VERSION#check-gpt }"
        fi
        
        # 比较版本号（去除v前缀）
        CURRENT_VERSION_NUM=${CURRENT_VERSION#v}
        LATEST_VERSION_NUM=${VERSION#v}
        
        # 去除可能的空格
        CURRENT_VERSION_NUM=$(echo "$CURRENT_VERSION_NUM" | tr -d '[:space:]')
        LATEST_VERSION_NUM=$(echo "$LATEST_VERSION_NUM" | tr -d '[:space:]')
        
        if [ "$CURRENT_VERSION_NUM" != "$LATEST_VERSION_NUM" ]; then
            print_message "发现新版本 ${VERSION}，当前版本 ${CURRENT_VERSION}"
            print_message "正在自动升级...\n"
            
            # 备份当前程序
            BACKUP_PATH="${INSTALLED_PATH}.bak"
            if mv "$INSTALLED_PATH" "$BACKUP_PATH"; then
                # 安装新版本
                INSTALL_DIR=$(dirname "$INSTALLED_PATH")
                if install_tool; then
                    rm -f "$BACKUP_PATH"
                    print_message "升级完成！"
                else
                    mv "$BACKUP_PATH" "$INSTALLED_PATH"
                    print_error "升级失败，已恢复原版本"
                    return 1
                fi
            else
                print_error "无法创建备份，取消升级"
                return 1
            fi
        else
            print_message "已安装最新版本"
        fi
    else
        print_message "未检测到已安装的程序，开始安装..."
        install_tool || return 1
    fi

    return 0
}

# 获取安装目录
get_install_dir() {
    if [ "$OS" = "windows" ]; then
        # Windows 下安装到用户目录
        INSTALL_DIR="$HOME/bin"
        mkdir -p "$INSTALL_DIR"
    else
        # Linux/macOS 下尝试安装到 /usr/local/bin，如果没权限则安装到 ~/bin
        if [ -w "/usr/local/bin" ]; then
            INSTALL_DIR="/usr/local/bin"
        else
            INSTALL_DIR="$HOME/bin"
            mkdir -p "$INSTALL_DIR"
            # 检查 PATH 中是否包含 ~/bin
            if [[ ":$PATH:" != *":$HOME/bin:"* ]]; then
                print_warning "请将 $INSTALL_DIR 添加到您的 PATH 环境变量中"
                case "$SHELL" in
                    */bash)
                        print_message "echo 'export PATH=\"\$HOME/bin:\$PATH\"' >> ~/.bashrc"
                        print_message "source ~/.bashrc"
                        ;;
                    */zsh)
                        print_message "echo 'export PATH=\"\$HOME/bin:\$PATH\"' >> ~/.zshrc"
                        print_message "source ~/.zshrc"
                        ;;
                esac
            fi
        fi
    fi
}

# 安装工具
install_tool() {
    print_message "开始安装 check-gpt 工具..."
    
    # 获取系统信息
    get_system_info
    
    # 获取安装目录
    get_install_dir
    
    # 去除版本号中的 'v' 前缀
    VERSION_NUM=${VERSION#v}
    
    # 构建下载 URL 和文件名
    if [ "$OS" = "windows" ]; then
        ARCHIVE_NAME="check-gpt_${VERSION_NUM}_${OS}_${ARCH}.zip"
        BINARY_NAME="check-gpt.exe"
    else
        ARCHIVE_NAME="check-gpt_${VERSION_NUM}_${OS}_${ARCH}.tar.gz"
        BINARY_NAME="check-gpt"
    fi
    DOWNLOAD_URL="https://github.com/${REPO}/releases/download/${VERSION}/${ARCHIVE_NAME}"
    
    print_message "下载链接: ${DOWNLOAD_URL}"
    
    # 创建临时目录
    TMP_DIR=$(mktemp -d)
    cd "$TMP_DIR" || exit 1
    
    # 下载文件
    print_message "下载发布包..."
    if ! curl -L -f -o "$ARCHIVE_NAME" "$DOWNLOAD_URL"; then
        print_error "下载失败: 未找到发布包 ${ARCHIVE_NAME}"
        print_error "请确认版本 ${VERSION} 已发布: https://github.com/${REPO}/releases"
        cd ..
        rm -rf "$TMP_DIR"
        exit 1
    fi
    
    # 检查文件大小
    if [ ! -s "$ARCHIVE_NAME" ]; then
        print_error "下载的文件为空"
        cd ..
        rm -rf "$TMP_DIR"
        exit 1
    fi
    
    if [ "$OS" = "windows" ]; then
        if ! unzip -q "$ARCHIVE_NAME"; then
            print_error "解压失败: 文件可能已损坏"
            cd ..
            rm -rf "$TMP_DIR"
            exit 1
        fi
    else
        if ! tar xzf "$ARCHIVE_NAME" 2>/dev/null; then
            print_error "解压失败: 文件可能已损坏"
            cd ..
            rm -rf "$TMP_DIR"
            exit 1
        fi
    fi
    
    # 安装文件
    print_message "安装程序到 $INSTALL_DIR"
    if [ -f "$BINARY_NAME" ]; then
        if mv "$BINARY_NAME" "$INSTALL_DIR/"; then
            chmod +x "$INSTALL_DIR/$BINARY_NAME"
        else
            print_error "安装失败，请尝试手动安装"
            print_message "您可以手动将 $BINARY_NAME 文件移动到 $INSTALL_DIR 目录"
            cd ..
            rm -rf "$TMP_DIR"
            exit 1
        fi
    else
        print_error "安装文件不存在"
        cd ..
        rm -rf "$TMP_DIR"
        exit 1
    fi
    
    # 清理临时文件
    cd ..
    rm -rf "$TMP_DIR"
}

# 主流程
main() {
    print_message "=== 安装 check-gpt 工具 ==="
    
    # 检查已安装版本并处理安装/升级
    check_and_install || exit 1
    
    print_message "安装完成！使用以下命令启动程序："
    # 检查是否在 PATH 中
    if [ "$INSTALL_DIR" = "/usr/local/bin" ] || [[ ":$PATH:" == *":$INSTALL_DIR:"* ]]; then
        print_message "check-gpt"
    else
        print_message "$INSTALL_DIR/check-gpt"
    fi
}

# 运行主流程
main 