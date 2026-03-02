#!/bin/sh
# install.sh — smtp2discord installer
#
# Downloads and installs the latest smtp2discord release as a system service.
# Supports Ubuntu, Debian, Alpine, Fedora, and Amazon Linux.
#
# Usage:
#   # Interactive (prompts for webhook URL):
#   curl -fsSL https://raw.githubusercontent.com/MrZoidberg/smtp2discord/master/install.sh | sudo sh        # Ubuntu/Debian/Fedora/Amazon Linux
#   curl -fsSL https://raw.githubusercontent.com/MrZoidberg/smtp2discord/master/install.sh | doas sh         # Alpine (uses doas, not sudo)
#
#   # Non-interactive (webhook URL supplied via flag):
#   curl -fsSL https://raw.githubusercontent.com/MrZoidberg/smtp2discord/master/install.sh | sudo sh -s -- --webhook https://discord.com/api/webhooks/<ID>/<TOKEN>
#   curl -fsSL https://raw.githubusercontent.com/MrZoidberg/smtp2discord/master/install.sh | doas sh -s -- --webhook https://discord.com/api/webhooks/<ID>/<TOKEN>   # Alpine
#
#   # Upgrade an existing installation (preserves /etc/default/smtp2discord):
#   curl -fsSL https://raw.githubusercontent.com/MrZoidberg/smtp2discord/master/install.sh | sudo sh -s -- --upgrade
#   curl -fsSL https://raw.githubusercontent.com/MrZoidberg/smtp2discord/master/install.sh | doas sh -s -- --upgrade   # Alpine
#
set -e

# ---------------------------------------------------------------------------
# Constants
# ---------------------------------------------------------------------------
REPO="MrZoidberg/smtp2discord"
GITHUB_API="https://api.github.com/repos/${REPO}/releases/latest"
CONFIG_FILE="/etc/default/smtp2discord"
BINARY_PATH="/usr/bin/smtp2discord"
SERVICE_NAME="smtp2discord"

# ---------------------------------------------------------------------------
# Helpers
# ---------------------------------------------------------------------------
info()  { printf '\033[1;32m  =>\033[0m  %s\n' "$*"; }
warn()  { printf '\033[1;33m warn:\033[0m %s\n' "$*"; }
error() { printf '\033[1;31m err: \033[0m %s\n' "$*" >&2; exit 1; }
step()  { printf '\n\033[1;34m[%s]\033[0m %s\n' "$1" "$2"; }

# Require root.
require_root() {
    if [ "$(id -u)" -ne 0 ]; then
        error "This installer must be run as root (try: sudo sh install.sh  or  doas sh install.sh on Alpine)"
    fi
}

# Pick a download tool.
downloader() {
    if command -v curl >/dev/null 2>&1; then
        curl --fail --silent --location "$1"
    elif command -v wget >/dev/null 2>&1; then
        wget --quiet --output-document=- "$1"
    else
        error "Neither curl nor wget is available. Install one and retry."
    fi
}

# ---------------------------------------------------------------------------
# Argument parsing
# ---------------------------------------------------------------------------
WEBHOOK_URL=""
UPGRADE=0

while [ "$#" -gt 0 ]; do
    case "$1" in
        --webhook)
            shift
            WEBHOOK_URL="$1"
            ;;
        --webhook=*)
            WEBHOOK_URL="${1#--webhook=}"
            ;;
        --upgrade)
            UPGRADE=1
            ;;
        -h|--help)
            cat <<EOF
smtp2discord installer

Options:
  --webhook <URL>   Discord webhook URL to write to ${CONFIG_FILE}
  --upgrade         Re-install over an existing installation (preserve config)
  --help            Show this help

If --webhook is omitted and the service is not yet configured, you will be
prompted interactively.
EOF
            exit 0
            ;;
        *)
            warn "Unknown option: $1 (ignored)"
            ;;
    esac
    shift
done

# ---------------------------------------------------------------------------
# OS / arch detection
# ---------------------------------------------------------------------------
detect_os() {
    if [ -f /etc/os-release ]; then
        # shellcheck disable=SC1091
        . /etc/os-release
        OS_ID="${ID}"
        OS_ID_LIKE="${ID_LIKE:-}"
    else
        error "Cannot determine OS: /etc/os-release not found."
    fi

    ARCH="$(uname -m)"
    case "${ARCH}" in
        x86_64)   ARCH="amd64" ;;
        aarch64)  ARCH="arm64" ;;
        *)        error "Unsupported architecture: ${ARCH}" ;;
    esac
}

# Map OS → package format + install command.
detect_package_manager() {
    case "${OS_ID}" in
        ubuntu|debian|raspbian)
            PKG_FORMAT="deb"
            PKG_INSTALL="apt-get install -y"
            PKG_UPGRADE="dpkg -i"
            ;;
        alpine)
            PKG_FORMAT="apk"
            PKG_INSTALL="apk add --allow-untrusted"
            PKG_UPGRADE="apk add --allow-untrusted"
            ;;
        fedora)
            PKG_FORMAT="rpm"
            PKG_INSTALL="dnf install -y"
            PKG_UPGRADE="dnf upgrade -y"
            ;;
        amzn)
            PKG_FORMAT="rpm"
            PKG_INSTALL="dnf install -y"
            PKG_UPGRADE="dnf upgrade -y"
            ;;
        *)
            # Check ID_LIKE for derivatives (e.g. linuxmint → ubuntu, rhel → fedora).
            case "${OS_ID_LIKE}" in
                *debian*|*ubuntu*)
                    PKG_FORMAT="deb"
                    PKG_INSTALL="apt-get install -y"
                    PKG_UPGRADE="dpkg -i"
                    ;;
                *fedora*|*rhel*)
                    PKG_FORMAT="rpm"
                    PKG_INSTALL="dnf install -y"
                    PKG_UPGRADE="dnf upgrade -y"
                    ;;
                *)
                    warn "Unrecognised OS '${OS_ID}' — falling back to raw binary install."
                    PKG_FORMAT="binary"
                    ;;
            esac
            ;;
    esac
}

# ---------------------------------------------------------------------------
# Release resolution
# ---------------------------------------------------------------------------
fetch_latest_version() {
    info "Fetching latest release from GitHub..."
    VERSION="$(downloader "${GITHUB_API}" | grep '"tag_name"' | sed 's/.*"tag_name": *"\([^"]*\)".*/\1/')"
    if [ -z "${VERSION}" ]; then
        error "Could not determine latest release version from GitHub API."
    fi
    info "Latest version: ${VERSION}"
}

build_download_url() {
    BARE_VERSION="${VERSION#v}"
    case "${PKG_FORMAT}" in
        deb)
            # goreleaser nfpm deb naming: smtp2discord_<version>_linux_<arch>.deb
            ASSET_NAME="smtp2discord_${BARE_VERSION}_linux_${ARCH}.deb"
            ;;
        rpm)
            ASSET_NAME="smtp2discord_${BARE_VERSION}_linux_${ARCH}.rpm"
            ;;
        apk)
            ASSET_NAME="smtp2discord_${BARE_VERSION}_linux_${ARCH}.apk"
            ;;
        binary)
            ASSET_NAME="smtp2discord_${BARE_VERSION}_linux_${ARCH}.tar.gz"
            ;;
    esac
    DOWNLOAD_URL="https://github.com/${REPO}/releases/download/${VERSION}/${ASSET_NAME}"
}

# ---------------------------------------------------------------------------
# Installation
# ---------------------------------------------------------------------------
install_package() {
    TMPDIR="$(mktemp -d)"
    trap 'rm -rf "${TMPDIR}"' EXIT

    DEST="${TMPDIR}/${ASSET_NAME}"
    info "Downloading ${DOWNLOAD_URL}..."
    downloader "${DOWNLOAD_URL}" > "${DEST}"

    case "${PKG_FORMAT}" in
        deb)
            if [ "${UPGRADE}" -eq 1 ]; then
                dpkg -i "${DEST}"
            else
                ${PKG_INSTALL} "${DEST}"
            fi
            ;;
        rpm)
            if [ "${UPGRADE}" -eq 1 ]; then
                rpm -Uvh "${DEST}"
            else
                ${PKG_INSTALL} "${DEST}"
            fi
            ;;
        apk)
            ${PKG_INSTALL} "${DEST}"
            ;;
        binary)
            install_binary "${DEST}"
            ;;
    esac
}

# Fallback: extract binary from tar.gz and install manually.
install_binary() {
    TARBALL="$1"
    info "Extracting binary..."
    tar -xzf "${TARBALL}" -C "${TMPDIR}" smtp2discord
    install -m 0755 "${TMPDIR}/smtp2discord" "${BINARY_PATH}"
    info "Binary installed to ${BINARY_PATH}"
    warn "Automatic service setup is not supported on this OS."
    warn "Refer to the README for manual service installation steps."
}

# ---------------------------------------------------------------------------
# Configuration
# ---------------------------------------------------------------------------
configure_webhook() {
    # If config file already has a non-empty webhook, skip prompts (upgrade case).
    if [ -f "${CONFIG_FILE}" ]; then
        EXISTING_WEBHOOK="$(grep '^SMTP2DISCORD_WEBHOOK=' "${CONFIG_FILE}" | cut -d= -f2-)"
        if [ -n "${EXISTING_WEBHOOK}" ] && [ "${WEBHOOOK_URL}" = "" ]; then
            info "Existing webhook configuration preserved."
            return 0
        fi
    fi

    # Use the supplied --webhook flag value if available.
    if [ -n "${WEBHOOK_URL}" ]; then
        write_webhook "${WEBHOOK_URL}"
        return 0
    fi

    # Interactive prompt (falls back gracefully when stdin is not a terminal).
    if [ -t 0 ]; then
        printf '\n\033[1;33mEnter your Discord webhook URL\033[0m\n'
        printf '(find it in Discord → channel settings → Integrations → Webhooks)\n'
        printf 'Webhook URL: '
        read -r WEBHOOK_URL
        if [ -n "${WEBHOOK_URL}" ]; then
            write_webhook "${WEBHOOK_URL}"
        else
            warn "No webhook URL provided. Set SMTP2DISCORD_WEBHOOK in ${CONFIG_FILE} before starting."
        fi
    else
        warn "stdin is not a terminal. Pass --webhook <URL> or set SMTP2DISCORD_WEBHOOK in ${CONFIG_FILE} manually."
    fi
}

write_webhook() {
    _url="$1"
    if [ -f "${CONFIG_FILE}" ]; then
        # Update existing config in-place.
        if grep -q '^SMTP2DISCORD_WEBHOOK=' "${CONFIG_FILE}"; then
            sed -i "s|^SMTP2DISCORD_WEBHOOK=.*|SMTP2DISCORD_WEBHOOK=${_url}|" "${CONFIG_FILE}"
        else
            printf '\nSMTP2DISCORD_WEBHOOK=%s\n' "${_url}" >> "${CONFIG_FILE}"
        fi
    else
        # Config file was not installed by the package (binary fallback).
        printf 'SMTP2DISCORD_WEBHOOK=%s\n' "${_url}" > "${CONFIG_FILE}"
    fi
    info "Webhook URL written to ${CONFIG_FILE}"
}

# ---------------------------------------------------------------------------
# Service start
# ---------------------------------------------------------------------------
start_service() {
    [ "${PKG_FORMAT}" = "binary" ] && return 0

    if command -v systemctl >/dev/null 2>&1 && systemctl is-system-running >/dev/null 2>&1; then
        systemctl daemon-reload
        if [ "${UPGRADE}" -eq 1 ]; then
            info "Restarting smtp2discord..."
            systemctl restart smtp2discord || warn "Could not restart service. Run: systemctl start smtp2discord"
        else
            info "Starting smtp2discord..."
            systemctl start smtp2discord || warn "Could not start service. Set SMTP2DISCORD_WEBHOOK in ${CONFIG_FILE}, then run: systemctl start smtp2discord"
        fi
    elif command -v rc-service >/dev/null 2>&1; then
        if [ "${UPGRADE}" -eq 1 ]; then
            info "Restarting smtp2discord..."
            rc-service smtp2discord restart || warn "Could not restart service."
        else
            info "Starting smtp2discord..."
            rc-service smtp2discord start || warn "Could not start service. Set SMTP2DISCORD_WEBHOOK in ${CONFIG_FILE}, then run: rc-service smtp2discord start"
        fi
    fi
}

# ---------------------------------------------------------------------------
# Success summary
# ---------------------------------------------------------------------------
print_summary() {
    echo ""
    echo "┌──────────────────────────────────────────────────┐"
    echo "│  smtp2discord ${VERSION} installed successfully!   "
    echo "├──────────────────────────────────────────────────┤"
    echo "│  Configuration : ${CONFIG_FILE}"
    echo "│  Binary        : ${BINARY_PATH}"
    echo "│"
    echo "│  Useful commands:"
    if command -v systemctl >/dev/null 2>&1 && systemctl is-system-running >/dev/null 2>&1; then
        echo "│    Status : systemctl status smtp2discord"
        echo "│    Logs   : journalctl -u smtp2discord -f"
        echo "│    Restart: systemctl restart smtp2discord"
    elif command -v rc-service >/dev/null 2>&1; then
        echo "│    Status : rc-service smtp2discord status"
        echo "│    Logs   : logread | grep smtp2discord"
        echo "│    Restart: rc-service smtp2discord restart"
    fi
    echo "└──────────────────────────────────────────────────┘"
    echo ""
}

# ---------------------------------------------------------------------------
# Main
# ---------------------------------------------------------------------------
main() {
    require_root

    step "1/5" "Detecting OS and architecture"
    detect_os
    detect_package_manager
    info "OS: ${OS_ID}  Arch: ${ARCH}  Package format: ${PKG_FORMAT}"

    step "2/5" "Resolving latest release"
    fetch_latest_version

    step "3/5" "Downloading and installing package"
    build_download_url
    install_package

    step "4/5" "Configuring webhook"
    configure_webhook

    step "5/5" "Starting service"
    start_service

    print_summary
}

main
