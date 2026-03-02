#!/bin/sh
set -e

# ---------------------------------------------------------------------------
# smtp2discord — pre-remove script
# Runs before package files are removed (uninstall, not upgrade).
# ---------------------------------------------------------------------------

_has_systemd() {
    command -v systemctl >/dev/null 2>&1 && systemctl is-system-running >/dev/null 2>&1
}

_has_openrc() {
    command -v rc-update >/dev/null 2>&1
}

# Stop and disable the service before files are removed.
if _has_systemd; then
    systemctl stop  smtp2discord.service 2>/dev/null || true
    systemctl disable smtp2discord.service 2>/dev/null || true
elif _has_openrc; then
    rc-service smtp2discord stop 2>/dev/null || true
    rc-update del smtp2discord default 2>/dev/null || true
fi
