#!/bin/bash
# Stop EchoLox before installation or update

# Stop via systemd if available
if systemctl is-active --quiet echolox.service 2>/dev/null; then
    systemctl stop echolox.service
    echo "<OK> EchoLox stopped (systemd)"
fi

# Stop via daemon script if available
DAEMONSCRIPT="${LBHOMEDIR:-/opt/loxberry}/daemon/plugins/EchoLox/EchoLox"
if [ -x "$DAEMONSCRIPT" ]; then
    "$DAEMONSCRIPT" stop 2>/dev/null
fi

# Kill by PID file as fallback
PIDFILE="/var/run/EchoLox.pid"
if [ -f "$PIDFILE" ]; then
    kill "$(cat "$PIDFILE")" 2>/dev/null
    rm -f "$PIDFILE"
fi

# Kill any remaining process by name
pkill -f "bin/plugins/EchoLox/EchoLox" 2>/dev/null

sleep 1
echo "<OK> EchoLox stopped"

# Ensure icon destination exists so the installer can copy the icon there
mkdir -p "${LBHOMEDIR:-/opt/loxberry}/webfrontend/html/system/images/icons/EchoLox" 2>/dev/null

exit 0
