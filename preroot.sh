#!/bin/bash
# EchoLox preroot.sh — runs as root before installation/update.
# Stops the running service and backs up devices.json.

# Stop via systemd
if systemctl is-active --quiet echolox.service 2>/dev/null; then
    systemctl stop echolox.service
    echo "<OK> EchoLox stopped (systemd)"
fi

# Stop via daemon script as fallback
DAEMONSCRIPT="${LBHOMEDIR:-/opt/loxberry}/daemon/plugins/EchoLox/EchoLox"
if [ -x "$DAEMONSCRIPT" ]; then
    "$DAEMONSCRIPT" stop 2>/dev/null
fi

# Kill by PID file as last resort
PIDFILE="/tmp/EchoLox.pid"
if [ -f "$PIDFILE" ]; then
    kill "$(cat "$PIDFILE")" 2>/dev/null
    rm -f "$PIDFILE"
fi

pkill -f "bin/plugins/EchoLox/EchoLox" 2>/dev/null
sleep 1
echo "<OK> EchoLox stopped"

# Backup devices.json so it survives the update
DEVICES_JSON="${LBPDATADIR:-/opt/loxberry/data/plugins/EchoLox}/devices.json"
if [ -f "$DEVICES_JSON" ]; then
    cp "$DEVICES_JSON" /tmp/EchoLox_devices.bak
    echo "<OK> EchoLox devices backed up"
fi

# Ensure icon destination exists for the installer
mkdir -p "${LBHOMEDIR:-/opt/loxberry}/webfrontend/html/system/images/icons/EchoLox" 2>/dev/null

exit 0
