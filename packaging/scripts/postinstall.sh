#!/bin/sh
set -e

# ---------------------------------------------------------------------------
# smtp2discord — post-install script
# Runs after the package files have been laid down.
# Handles both fresh installs and upgrades.
# ---------------------------------------------------------------------------

SERVICE_USER="smtp2discord"
SERVICE_GROUP="smtp2discord"
CONFIG_FILE="/etc/default/smtp2discord"

# ---- Create service user/group (skip if already present) -----------------
if ! getent group "${SERVICE_GROUP}" >/dev/null 2>&1; then
    if command -v groupadd >/dev/null 2>&1; then
        groupadd --system "${SERVICE_GROUP}"
    elif command -v addgroup >/dev/null 2>&1; then
        addgroup --system "${SERVICE_GROUP}"
    fi
fi

if ! getent passwd "${SERVICE_USER}" >/dev/null 2>&1; then
    if command -v useradd >/dev/null 2>&1; then
        useradd \
            --system \
            --gid "${SERVICE_GROUP}" \
            --no-create-home \
            --home-dir /nonexistent \
            --shell /usr/sbin/nologin \
            --comment "smtp2discord service account" \
            "${SERVICE_USER}"
    elif command -v adduser >/dev/null 2>&1; then
        adduser \
            --system \
            --ingroup "${SERVICE_GROUP}" \
            --no-create-home \
            --home /nonexistent \
            --shell /usr/sbin/nologin \
            --disabled-password \
            --gecos "smtp2discord service account" \
            "${SERVICE_USER}"
    fi
fi

# ---- Init system detection -----------------------------------------------
_has_systemd() {
    command -v systemctl >/dev/null 2>&1 && systemctl is-system-running >/dev/null 2>&1
}

_has_openrc() {
    command -v rc-update >/dev/null 2>&1
}

# ---- Register and enable service -----------------------------------------
if _has_systemd; then
    systemctl daemon-reload
    # Enable on boot (idempotent for upgrades).
    systemctl enable smtp2discord.service 2>/dev/null || true
elif _has_openrc; then
    rc-update add smtp2discord default 2>/dev/null || true
fi

# ---- Post-install notice --------------------------------------------------
if grep -q '^SMTP2DISCORD_WEBHOOK=$' "${CONFIG_FILE}" 2>/dev/null ||
   ! grep -q 'SMTP2DISCORD_WEBHOOK' "${CONFIG_FILE}" 2>/dev/null; then
    echo ""
    echo "┌──────────────────────────────────────────────────────────────┐"
    echo "│  smtp2discord installed — action required before first start  │"
    echo "├──────────────────────────────────────────────────────────────┤"
    echo "│  1. Set your Discord webhook URL:                             │"
    echo "│       \$EDITOR ${CONFIG_FILE}  │"
    echo "│     → set SMTP2DISCORD_WEBHOOK=https://discord.com/...        │"
    echo "│                                                               │"
    echo "│  2. Start the service:                                        │"
    echo "│     systemd:  systemctl start smtp2discord                    │"
    echo "│     OpenRC:   rc-service smtp2discord start                   │"
    echo "└──────────────────────────────────────────────────────────────┘"
    echo ""
fi
