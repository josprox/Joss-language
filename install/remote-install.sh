#!/bin/bash
# -----------------------------------------------------------
# JosSecurity Remote Installer, Updater, and Uninstaller (Linux/macOS)
# Usos:
#   Instalación: curl -fsSL <URL_DE_ESTE_SCRIPT> | bash
#   Ejecución manual: ./remote-install.sh
# -----------------------------------------------------------

set -e

# --- COLORS & CONFIGURACIÓN ---
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

# Configuración que DEBE ser actualizada manualmente en el script
JOSS_VERSION="3.6.1"
REPO_OWNER="josprox"
REPO_NAME="Joss-language"

# Rutas
INSTALL_DIR="/usr/local/bin"
SDK_INSTALL_DIR="/usr/local/share/joss/sdk"
LOG_FILE="/tmp/jossecurity-action.log"
TEMP_DIR="/tmp/jossecurity-temp-action"
REPO_URL="https://api.github.com/repos/$REPO_OWNER/$REPO_NAME/releases/latest"
# --------------------

# --- FUNCIONES DE LOGGING/UTILIDADES ---
log() {
    local level=$1
    shift
    local message="$@"
    local timestamp=$(date '+%Y-%m-%d %H:%M:%S')
    echo "[$timestamp] [$level] $message" >> "$LOG_FILE"
    case $level in
        ERROR) echo -e "${RED}$message${NC}";;
        SUCCESS) echo -e "${GREEN}$message${NC}";;
        WARNING) echo -e "${YELLOW}$message${NC}";;
        *) echo -e "$message";;
    esac
}

detect_vscode() {
    command -v code &> /dev/null
}

run_privileged() {
    if [ "$(id -u)" -eq 0 ]; then
        "$@"
        return $?
    fi
    if command -v sudo &> /dev/null; then
        sudo "$@"
        return $?
    fi
    log ERROR "[X] Administrator privileges are required, but sudo is not available."
    return 1
}

# Detecta el nombre del binario a buscar en el ZIP
get_binary_name() {
    OS=$(uname -s)
    ARCH=$(uname -m)

    case "$OS" in
        Linux*)
            case "$ARCH" in
                x86_64) echo "joss-linux-amd64";;
                aarch64) echo "joss-linux-arm64";;
                armv7l) echo "joss-linux-armv7";;
                *) log ERROR "Linux Architecture $ARCH not supported."; exit 1;;
            esac
            ;;
        Darwin*)
            case "$ARCH" in
                x86_64) echo "joss-macos-amd64";;
                arm64) echo "joss-macos-arm64";;
                *) log ERROR "macOS Architecture $ARCH not supported."; exit 1;;
            esac
            ;;
        *)
            log ERROR "Unsupported OS: $OS"
            exit 1
            ;;
    esac
}
# --------------------

# --- FUNCIONES DE ACCIÓN ---

# 1. Instalación
install_jossecurity() {
    log INFO "[1/3] Installing JosSecurity..."
    
    BINARY_FILE=$(get_binary_name)
    BINARY_PATH=$(find "$TEMP_DIR" -name "$BINARY_FILE" -type f | head -n 1)

    if [ -z "$BINARY_PATH" ]; then
        log ERROR "[X] Binary $BINARY_FILE not found in ZIP."
        return 1
    fi
    
    log INFO "Found binary: $BINARY_FILE. Installing to $INSTALL_DIR..."

    if ! run_privileged mkdir -p "$INSTALL_DIR"; then
        log ERROR "[X] Could not create installation directory $INSTALL_DIR."
        return 1
    fi

    STAGED_BINARY="$INSTALL_DIR/.joss-install-$$"
    if ! run_privileged cp "$BINARY_PATH" "$STAGED_BINARY" ||
       ! run_privileged chmod +x "$STAGED_BINARY" ||
       ! run_privileged mv -f "$STAGED_BINARY" "$INSTALL_DIR/joss"; then
        run_privileged rm -f "$STAGED_BINARY" || true
        log ERROR "[X] Failed to install binary in $INSTALL_DIR."
        return 1
    fi

    log SUCCESS "[OK] Binary installed and executable."
    if ! command -v joss &> /dev/null; then
        log WARNING "$INSTALL_DIR is not currently in PATH. Restart the terminal or add it to your shell profile."
    fi
    return 0
}

install_sdk() {
    log INFO "[2/3] Installing Joss plugin SDK..."

    SDK_SOURCE="$TEMP_DIR/sdk-package/sdk"
    if [ ! -d "$SDK_SOURCE" ]; then
        log ERROR "[X] SDK directory not found in downloaded package."
        return 1
    fi

    SDK_PARENT=$(dirname "$SDK_INSTALL_DIR")
    STAGED_SDK="$SDK_PARENT/.sdk-install-$$"
    if ! run_privileged mkdir -p "$SDK_PARENT" ||
       ! run_privileged rm -rf "$STAGED_SDK" ||
       ! run_privileged mkdir -p "$STAGED_SDK" ||
       ! run_privileged cp -R "$SDK_SOURCE"/. "$STAGED_SDK"/; then
        run_privileged rm -rf "$STAGED_SDK" || true
        log ERROR "[X] SDK staging failed."
        return 1
    fi

    if ! run_privileged rm -rf "$SDK_INSTALL_DIR" ||
       ! run_privileged mv "$STAGED_SDK" "$SDK_INSTALL_DIR"; then
        run_privileged rm -rf "$STAGED_SDK" || true
        log ERROR "[X] SDK installation failed."
        return 1
    fi

    log SUCCESS "[OK] SDK installed at $SDK_INSTALL_DIR"
    return 0
}

install_extension() {
    log INFO "[3/3] Installing VS Code extension..."
    
    if ! detect_vscode; then
        log WARNING "[SKIP] VS Code not detected. Extension installation skipped."
        return 0
    fi

    # Buscar CUALQUIER archivo VSIX
    VSIX_FILE=$(find "$TEMP_DIR" -name "*.vsix" -type f | head -n 1)
    
    if [ -z "$VSIX_FILE" ]; then
        log ERROR "[X] VSIX file not found in ZIP."
        return 1
    fi
    
    log INFO "Found VSIX: $(basename "$VSIX_FILE"). Installing..."
    
    # El comando --install-extension --force maneja la actualización/instalación
    if code --install-extension "$VSIX_FILE" --force &> /dev/null; then
        log SUCCESS "[OK] Extension installed successfully."
        return 0
    else
        log ERROR "[X] Extension installation failed. Is VS Code in your PATH?"
        return 1
    fi
}

# 2. Desinstalación
uninstall_jossecurity() {
    log INFO "Uninstalling JosSecurity..."
    if [ -f "$INSTALL_DIR/joss" ]; then
        if ! run_privileged rm -f "$INSTALL_DIR/joss"; then
            log ERROR "[X] Failed to remove $INSTALL_DIR/joss"
            return 1
        fi
        log SUCCESS "[OK] Binary removed from $INSTALL_DIR"
    else
        log INFO "[OK] Binary not found."
    fi
    if [ -d "$SDK_INSTALL_DIR" ]; then
        if ! run_privileged rm -rf "$SDK_INSTALL_DIR"; then
            log ERROR "[X] Failed to remove $SDK_INSTALL_DIR"
            return 1
        fi
        log SUCCESS "[OK] SDK removed from $SDK_INSTALL_DIR"
    fi
}

uninstall_extension() {
    log INFO "Uninstalling VS Code extension..."
    if detect_vscode && code --list-extensions | grep -q "joss-language"; then
        code --uninstall-extension jossecurity.joss-language &> /dev/null
        log SUCCESS "[OK] Extension uninstalled."
    else
        log INFO "[OK] Extension not found or already removed."
    fi
}

# 3. Actualización
check_update() {
    log INFO "Checking for updates..."
    
    LOCAL_VERSION="0.0.0"
    if command -v joss &> /dev/null; then
        VERSION_OUT=$(joss version 2>&1 || true)
        if [[ $VERSION_OUT =~ v([0-9]+\.[0-9]+\.[0-9]+) ]]; then
            LOCAL_VERSION="${BASH_REMATCH[1]}"
        fi
    fi
    
    log INFO "Current version: $LOCAL_VERSION"
    
    RELEASE_INFO=$(curl -s "$REPO_URL")
    LATEST_TAG=$(echo "$RELEASE_INFO" | grep '"tag_name"' | head -n 1 | sed -E 's/.*"tag_name"[[:space:]]*:[[:space:]]*"([^"]+)".*/\1/' || echo "")
    LATEST_VERSION="${LATEST_TAG#[vV]}"
    if [ -z "$LATEST_VERSION" ]; then LATEST_VERSION="0.0.0"; fi
    
    log INFO "Latest version: $LATEST_VERSION"
    
    if [ "$LATEST_VERSION" != "0.0.0" ] && [ "$(printf '%s\n' "$LATEST_VERSION" "$LOCAL_VERSION" | sort -V | head -n1)" != "$LATEST_VERSION" ]; then
        log WARNING "[!] Update available: $LATEST_VERSION"
        return 0 # Update available
    fi
    log SUCCESS "[OK] You have the latest version ($LOCAL_VERSION)."
    return 1 # No update
}

run_update() {
    log INFO "Running update: Download and reinstalling."
    if install_jossecurity && install_sdk && install_extension; then
        log SUCCESS "Update completed successfully."
        return 0
    fi
    log ERROR "Update failed."
    return 1
}

# --- FLUJO DE TRABAJO PRINCIPAL ---

ensure_vscode() {
    if detect_vscode; then
        return 0
    fi
    
    echo ""
    log WARNING "Visual Studio Code is not installed."
    read -p "Do you want to install VS Code? (y/n) " ans < /dev/tty
    if [ "$ans" = "y" ]; then
        if [[ "$(uname -s)" == "Darwin" ]]; then
             log INFO "Please install via Homebrew: brew install --cask visual-studio-code"
             # Or try to run it? safer to just instruct.
        else
             # Linux attempt
             if command -v snap &> /dev/null; then
                 run_privileged snap install code --classic
                 return 0
             elif command -v apt-get &> /dev/null; then
                 # Complex on debian/ubuntu without repo added.
                 log INFO "Please install manually: https://code.visualstudio.com/download"
             else
                 log INFO "Please install manually: https://code.visualstudio.com/download"
             fi
        fi
    fi
}

# 1. Descarga y Extracción
resolve_release_tag() {
    local requested="${1:-}"
    local payload=""
    if [ -z "$requested" ]; then
        payload=$(curl -fsSL "$REPO_URL") || return 1
        echo "$payload" | grep '"tag_name"' | head -n 1 | sed -E 's/.*"tag_name"[[:space:]]*:[[:space:]]*"([^"]+)".*/\1/'
        return 0
    fi

    local normalized="${requested#[vV]}"
    if ! [[ "$normalized" =~ ^[0-9]+\.[0-9]+\.[0-9]+([.-][0-9A-Za-z.-]+)?$ ]]; then
        return 1
    fi

    local tag
    for tag in "$requested" "v$normalized" "V$normalized"; do
        payload=$(curl -fsSL "https://api.github.com/repos/$REPO_OWNER/$REPO_NAME/releases/tags/$tag" 2>/dev/null) || continue
        echo "$payload" | grep '"tag_name"' | head -n 1 | sed -E 's/.*"tag_name"[[:space:]]*:[[:space:]]*"([^"]+)".*/\1/'
        return 0
    done
    return 1
}

download_and_extract() {
    RELEASE_TAG="${1:-}"
    if [ -z "$RELEASE_TAG" ]; then
        RELEASE_TAG=$(resolve_release_tag) || {
            log ERROR "[X] Could not resolve the latest release."
            return 1
        }
    fi

    log INFO "[INIT] Preparing temp directory..."
    rm -rf "$TEMP_DIR"
    mkdir -p "$TEMP_DIR"

    # Determine correct ZIP
    OS=$(uname -s)
    if [[ "$OS" == "Darwin" ]]; then
        OS_ZIP="jossecurity-macos.zip"
    else
        OS_ZIP="jossecurity-linux.zip"
    fi
    EXT_ZIP="jossecurity-vscode.zip"
    SDK_ZIP="joss-plugin-sdk.zip"

    BINARY_URL="https://github.com/$REPO_OWNER/$REPO_NAME/releases/download/$RELEASE_TAG/$OS_ZIP"
    EXT_URL="https://github.com/$REPO_OWNER/$REPO_NAME/releases/download/$RELEASE_TAG/$EXT_ZIP"
    SDK_URL="https://github.com/$REPO_OWNER/$REPO_NAME/releases/download/$RELEASE_TAG/$SDK_ZIP"
    log INFO "Selected release: $RELEASE_TAG"
    
    # 1. Binaries
    log INFO "[INIT] Downloading Binaries ($OS_ZIP)..."
    if ! curl -fsSL "$BINARY_URL" -o "$TEMP_DIR/$OS_ZIP"; then
        log ERROR "[X] Failed to download $OS_ZIP. Check internet or release version."
        return 1
    fi
    
    # 2. Extension
    log INFO "[INIT] Downloading Extension ($EXT_ZIP)..."
    if ! curl -fsSL "$EXT_URL" -o "$TEMP_DIR/$EXT_ZIP"; then
        log ERROR "[X] Failed to download extension." 
        # Non-fatal? Maybe we still proceed with binary. But let's fail safe.
        return 1
    fi

    # 3. SDK
    log INFO "[INIT] Downloading Plugin SDK ($SDK_ZIP)..."
    if ! curl -fsSL "$SDK_URL" -o "$TEMP_DIR/$SDK_ZIP"; then
        log ERROR "[X] Failed to download plugin SDK."
        return 1
    fi
    
    log INFO "[INIT] Extracting files..."
    unzip -o "$TEMP_DIR/$OS_ZIP" -d "$TEMP_DIR/runtime"
    unzip -o "$TEMP_DIR/$EXT_ZIP" -d "$TEMP_DIR/extension"
    unzip -o "$TEMP_DIR/$SDK_ZIP" -d "$TEMP_DIR/sdk-package"
    
    # Check VS Code
    ensure_vscode
    
    return 0
}

main_menu() {
    echo -e "${BLUE}"
    echo "======================================="
    echo "  JosSecurity Action Menu"
    echo "======================================="
    echo -e "${NC}"
    echo ""
    echo "  [1] Install (Joss Binary + SDK + Extension)"
    echo "  [2] Update (Only When a New Version Exists)"
    echo "  [3] Reinstall (Latest or Specific Version)"
    echo "  [4] Uninstall (Remove Binary + SDK + Extension)"
    echo "  [0] Exit"
    echo ""
    read -p "Option: " option < /dev/tty

    case $option in
        1)
            if download_and_extract; then run_update; fi
            ;;
        2)
            if check_update; then
                log SUCCESS "Updating to v$LATEST_VERSION"
                if download_and_extract; then run_update; fi
            fi
            ;;
        3)
            read -p "Version to reinstall (leave blank for latest): " requested_version < /dev/tty
            release_tag=$(resolve_release_tag "$requested_version") || {
                log ERROR "[X] Requested release was not found."
                return 1
            }
            log INFO "Reinstalling $release_tag..."
            if download_and_extract "$release_tag"; then run_update; fi
            ;;
        4)
            uninstall_jossecurity
            uninstall_extension
            ;;
        0) log INFO "Operation cancelled."; return 0;;
        *) log ERROR "Invalid option";;
    esac

    log INFO "Cleaning up temp directory..."
    rm -rf "$TEMP_DIR"
    log SUCCESS "Operation finished. Log: $LOG_FILE"
}

# Execute main logic
if [ "${JOSS_INSTALLER_SKIP_MENU:-0}" != "1" ]; then
    main_menu
fi
