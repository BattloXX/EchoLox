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

# Backup devices.json before update so it can be restored if needed
DEVICES_JSON="${LBPDATADIR:-/opt/loxberry/data/plugins/EchoLox}/devices.json"
if [ -f "$DEVICES_JSON" ]; then
    cp "$DEVICES_JSON" /tmp/EchoLox_devices.bak
    echo "<OK> EchoLox devices backed up"
fi

# Ensure icon destination exists so the installer can copy the icon there
mkdir -p "${LBHOMEDIR:-/opt/loxberry}/webfrontend/html/system/images/icons/EchoLox" 2>/dev/null

# Remove Apache2 proxy config before update (postroot.sh will re-add it cleanly)
if command -v a2disconf >/dev/null 2>&1; then
    a2disconf echolox-hue >/dev/null 2>&1
    if [ -f /etc/apache2/conf-available/echolox-hue.conf ]; then
        rm -f /etc/apache2/conf-available/echolox-hue.conf
        apache2ctl graceful >/dev/null 2>&1
        echo "<OK> EchoLox Apache2 proxy removed (will be re-added by postroot.sh)"
    fi
fi

exit 0
