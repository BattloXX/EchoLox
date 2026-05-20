#!/bin/bash
# EchoLox preroot.sh — runs as root before installation/update.
# Stops the running service and backs up devices.json.

LBHOMEDIR="${LBHOMEDIR:-/opt/loxberry}"
LBPDATADIR="${LBPDATADIR:-$LBHOMEDIR/data/plugins/EchoLox}"

# Stop via systemd
if systemctl is-active --quiet echolox.service 2>/dev/null; then
    systemctl stop echolox.service
    echo "<OK> EchoLox stopped (systemd)"
fi

# Stop via daemon script as fallback
DAEMONSCRIPT="$LBHOMEDIR/daemon/plugins/EchoLox/EchoLox"
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

# Backup devices.json to /var/tmp — outside the plugin directory tree so it
# survives LoxBerry's purge_installation (which wipes $LBPDATADIR and
# $LBPCFGDIR between preroot.sh and postroot.sh), and persists across reboots
# (unlike /tmp which is a tmpfs on most LoxBerry systems).
DEVICES_JSON="$LBPDATADIR/devices.json"
BACKUP_PATH="/var/tmp/EchoLox_devices.bak"
if [ -f "$DEVICES_JSON" ]; then
    cp "$DEVICES_JSON" "$BACKUP_PATH"
    echo "<OK> EchoLox devices backed up to $BACKUP_PATH"
fi

# Ensure icon destination exists for the installer
mkdir -p "${LBHOMEDIR:-/opt/loxberry}/webfrontend/html/system/images/icons/EchoLox" 2>/dev/null

exit 0
