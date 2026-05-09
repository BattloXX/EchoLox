#!/bin/bash
# EchoLox postinstall: grant the binary permission to bind port 80 without root.
# Requires: LoxBerry admin port moved to 88, lbssdpd disabled (see README).

BIN="${LBPBINDIR:-/opt/loxberry/webfrontend/htmlauth/plugins/EchoLox/bin}/EchoLox"

if [ -x "$BIN" ]; then
    if command -v setcap >/dev/null 2>&1; then
        if setcap 'cap_net_bind_service=+ep' "$BIN"; then
            echo "EchoLox: CAP_NET_BIND_SERVICE set on $BIN — port 80 binding enabled"
        else
            echo "EchoLox WARNING: setcap failed — EchoLox may not bind port 80 without root"
        fi
    else
        echo "EchoLox WARNING: setcap not found — install libcap2-bin, then run: setcap 'cap_net_bind_service=+ep' $BIN"
    fi
else
    echo "EchoLox WARNING: binary not found at $BIN — skipping setcap"
fi

# Non-fatal sanity checks
if ss -tlnp 2>/dev/null | grep -qE ':80\s'; then
    echo "EchoLox WARNING: something is already listening on port 80."
    echo "  -> Move LoxBerry admin port to 88 (LoxBerry Settings -> System) before starting EchoLox."
fi

if systemctl is-active --quiet lbssdpd 2>/dev/null; then
    echo "EchoLox WARNING: lbssdpd is still running and holds port 1900."
    echo "  -> Disable it: systemctl disable --now lbssdpd"
fi

exit 0
