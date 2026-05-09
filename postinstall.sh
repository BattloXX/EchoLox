#!/bin/bash
# EchoLox postinstall (runs as plugin user, non-root).
# Binary selection, setcap, and systemd setup are handled by postroot.sh (root).
# This script migrates old configs as a non-root safety-net pass.

DATADIR="${LBPDATADIR:-/opt/loxberry/data/plugins/EchoLox}"
CFG="$DATADIR/EchoLox.cfg"

if [ -f "$CFG" ]; then
    if grep -q 'port: 8079' "$CFG"; then
        sed -i 's/port: 8079/port: 80/' "$CFG" 2>/dev/null
        echo "EchoLox: migrated config: port 8079 → 80"
    fi
    if grep -q 'discovery_port' "$CFG"; then
        sed -i '/discovery_port/d' "$CFG" 2>/dev/null
        echo "EchoLox: migrated config: discovery_port removed"
    fi
fi

exit 0
