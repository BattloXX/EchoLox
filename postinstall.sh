#!/bin/bash
# EchoLox postinstall:
#  1. Select the correct binary for this CPU architecture
#  2. Grant CAP_NET_BIND_SERVICE so EchoLox can bind port 80 without root
#  3. Sanity-check that port 80 and port 1900 are free

BINDIR="${LBPBINDIR:-/opt/loxberry/webfrontend/htmlauth/plugins/EchoLox/bin}"
BIN="$BINDIR/EchoLox"

# ── 1. Architecture selection ────────────────────────────────────────────────
ARCH=$(uname -m)
case "$ARCH" in
    aarch64)        SRC="$BINDIR/EchoLox-arm64" ;;
    armv7l|armv6l)  SRC="$BINDIR/EchoLox-armv7" ;;
    x86_64)         SRC="$BINDIR/EchoLox-amd64"  ;;
    *)              SRC="$BINDIR/EchoLox-arm64"  ;;   # best-effort fallback
esac

if [ -f "$SRC" ]; then
    cp "$SRC" "$BIN"
    chmod +x "$BIN"
    echo "EchoLox: installed $SRC as $BIN (arch: $ARCH)"
else
    echo "EchoLox ERROR: binary for arch '$ARCH' not found at $SRC — aborting"
    exit 1
fi

# ── 2. Privileged-port binding (port 80) ────────────────────────────────────
if command -v setcap >/dev/null 2>&1; then
    if setcap 'cap_net_bind_service=+ep' "$BIN"; then
        echo "EchoLox: CAP_NET_BIND_SERVICE set on $BIN — port 80 binding enabled"
    else
        echo "EchoLox WARNING: setcap failed — EchoLox may not bind port 80 without root"
    fi
else
    echo "EchoLox WARNING: setcap not found — install libcap2-bin, then run:"
    echo "  setcap 'cap_net_bind_service=+ep' $BIN"
fi

# ── 3. Sanity checks (non-fatal) ─────────────────────────────────────────────
if ss -tlnp 2>/dev/null | grep -qE ':80\s'; then
    echo "EchoLox WARNING: something is already listening on port 80."
    echo "  -> Move LoxBerry admin port to 88 (LoxBerry Settings -> System) before starting EchoLox."
fi

if systemctl is-active --quiet lbssdpd 2>/dev/null; then
    echo "EchoLox WARNING: lbssdpd is still running and holds port 1900."
    echo "  -> Disable it: systemctl disable --now lbssdpd"
fi

exit 0
