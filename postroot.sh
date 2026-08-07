#!/bin/bash
# EchoLox postroot.sh — runs as root after files are installed.
# LoxBerry sets these variables before calling this script:
#   LBHOMEDIR  -- LoxBerry home      (e.g. /opt/loxberry)
#   LBPBINDIR  -- Plugin binary dir  (e.g. /opt/loxberry/bin/plugins/EchoLox)
#   LBPCFGDIR  -- Plugin config dir  (e.g. /opt/loxberry/config/plugins/EchoLox)
#   LBPDATADIR -- Plugin data dir    (e.g. /opt/loxberry/data/plugins/EchoLox)
#   LBPLOGDIR  -- Plugin log dir     (e.g. /opt/loxberry/log/plugins/EchoLox)

LBHOMEDIR="${LBHOMEDIR:-/opt/loxberry}"
LBPBINDIR="${LBPBINDIR:-$LBHOMEDIR/bin/plugins/EchoLox}"
LBPCFGDIR="${LBPCFGDIR:-$LBHOMEDIR/config/plugins/EchoLox}"
LBPDATADIR="${LBPDATADIR:-$LBHOMEDIR/data/plugins/EchoLox}"
LBPLOGDIR="${LBPLOGDIR:-$LBHOMEDIR/log/plugins/EchoLox}"
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"

DAEMONDIR="$LBHOMEDIR/daemon/plugins/EchoLox"
PIDFILE="/tmp/EchoLox.pid"
CFGFILE="$LBPDATADIR/EchoLox.cfg"

# ── 1. Architecture-specific binary selection ────────────────────────────────
ARCH=$(uname -m)
case "$ARCH" in
  aarch64)       cp "$LBPBINDIR/EchoLox-arm64" "$LBPBINDIR/EchoLox" ;;
  armv7l|armv6l) cp "$LBPBINDIR/EchoLox-armv7" "$LBPBINDIR/EchoLox" ;;
  x86_64)        cp "$LBPBINDIR/EchoLox-amd64"  "$LBPBINDIR/EchoLox" ;;
  *) echo "<FAIL> EchoLox: unsupported architecture: $ARCH"; exit 2 ;;
esac
chmod +x "$LBPBINDIR/EchoLox"
echo "<OK> EchoLox binary selected for $ARCH"

# ── 2. CAP_NET_BIND_SERVICE — allow binding port 80 without root ─────────────
if command -v setcap >/dev/null 2>&1; then
    if setcap 'cap_net_bind_service=+ep' "$LBPBINDIR/EchoLox"; then
        echo "<OK> EchoLox: CAP_NET_BIND_SERVICE set — port 80 binding enabled"
    else
        echo "<WARN> EchoLox: setcap failed — EchoLox may not bind port 80 without root"
    fi
else
    echo "<WARN> EchoLox: setcap not found — install libcap2-bin for port 80 binding"
fi

# ── 3. Data and log directories ──────────────────────────────────────────────
mkdir -p "$LBPDATADIR" "$LBPLOGDIR"
chown -R loxberry:loxberry "$LBPDATADIR" "$LBPLOGDIR" 2>/dev/null || true

# ── 4. Restore devices.json after update ────────────────────────────────────
BACKUP_PATH="/var/tmp/EchoLox_devices.bak"
if [ ! -f "$LBPDATADIR/devices.json" ] && [ -f "$BACKUP_PATH" ]; then
    cp "$BACKUP_PATH" "$LBPDATADIR/devices.json"
    chown loxberry:loxberry "$LBPDATADIR/devices.json" 2>/dev/null || true
    echo "<OK> EchoLox devices restored from backup"
fi
rm -f "$BACKUP_PATH"

# ── 5. Create or migrate user config ─────────────────────────────────────────
if [ -f "$CFGFILE" ]; then
    # Migrate: port 8079 → 80 (old proxy architecture → direct binding)
    if grep -q 'port: 8079' "$CFGFILE"; then
        sed -i 's/port: 8079/port: 80/' "$CFGFILE"
        echo "<OK> EchoLox config migrated: port 8079 → 80"
    fi
else
    # Migrate from old config-dir location if present
    if [ -f "$LBPCFGDIR/EchoLox.cfg" ]; then
        cp "$LBPCFGDIR/EchoLox.cfg" "$CFGFILE"
        sed -i 's/port: 8079/port: 80/' "$CFGFILE" 2>/dev/null || true
        echo "<OK> EchoLox config migrated from config dir"
    else
        cat > "$CFGFILE" << CFGEOF
server:
  port: 80

loxone:
  miniserver: "1"
  transport: "http"
  udp_port: 7777

data_dir: "$LBPDATADIR"
CFGEOF
        echo "<OK> EchoLox default config created (port 80)"
    fi
fi
chown loxberry:loxberry "$CFGFILE" 2>/dev/null || true

# Reverse-proxy mode listens on an unprivileged internal port.
if grep -qE '^[[:space:]]*discovery_port:[[:space:]]*[1-9][0-9]*' "$CFGFILE"; then
    setcap -r "$LBPBINDIR/EchoLox" 2>/dev/null || true
fi

# ── 6. Sanity checks (non-fatal) ─────────────────────────────────────────────
if ss -tlnp 2>/dev/null | grep -qE ':80\s'; then
    echo "<WARN> EchoLox: port 80 is already in use — move LoxBerry admin port to 88 first"
fi
if ss -tlnp 2>/dev/null | grep -q 'tcp6.*:80' || ss -tlnp6 2>/dev/null | grep -qE ':80\s'; then
    echo "<WARN> EchoLox: tcp6 port 80 is still occupied — known Apache IPv6 issue (LoxBerry #1481); this may prevent EchoLox's optional IPv6 listener from binding, while IPv4 remains available"
fi
if systemctl is-active --quiet lbssdpd 2>/dev/null; then
    echo "<WARN> EchoLox: lbssdpd is active — disable it: systemctl disable --now lbssdpd"
fi

# ── 7. LoxBerry daemon init script ───────────────────────────────────────────
mkdir -p "$DAEMONDIR"
cat > "$DAEMONDIR/EchoLox" << DAEMONEOF
#!/bin/bash
BINARY="$LBPBINDIR/EchoLox"
CFGFILE="$CFGFILE"
PIDFILE="$PIDFILE"
export LBHOMEDIR="$LBHOMEDIR"
export LBPBINDIR="$LBPBINDIR"
export LBPCFGDIR="$LBPCFGDIR"
export LBPDATADIR="$LBPDATADIR"
export LBPLOGDIR="$LBPLOGDIR"

case "\$1" in
  start)
    "\$BINARY" --config "\$CFGFILE" &
    echo \$! > "\$PIDFILE"
    echo "<OK> EchoLox started (PID \$(cat \$PIDFILE))"
    ;;
  stop)
    if [ -f "\$PIDFILE" ]; then
      kill \$(cat "\$PIDFILE") 2>/dev/null && rm -f "\$PIDFILE"
      echo "<OK> EchoLox stopped"
    else
      echo "<INFO> EchoLox not running"
    fi
    ;;
  status)
    if [ -f "\$PIDFILE" ] && kill -0 \$(cat "\$PIDFILE") 2>/dev/null; then
      echo "running (PID \$(cat \$PIDFILE))"
    else
      echo "stopped"
    fi
    ;;
  restart)
    \$0 stop
    sleep 1
    \$0 start
    ;;
  *)
    echo "Usage: \$0 {start|stop|status|restart}"
    exit 1
    ;;
esac
exit 0
DAEMONEOF
chmod +x "$DAEMONDIR/EchoLox"

# ── 8. systemd service ────────────────────────────────────────────────────────
cat > /etc/systemd/system/echolox.service << SVCEOF
[Unit]
Description=EchoLox Hue Bridge Emulator for Loxone
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
User=loxberry
Group=loxberry
ExecStart=$LBPBINDIR/EchoLox --config $CFGFILE
Environment=LBHOMEDIR=$LBHOMEDIR
Environment=LBPBINDIR=$LBPBINDIR
Environment=LBPCFGDIR=$LBPCFGDIR
Environment=LBPDATADIR=$LBPDATADIR
Environment=LBPLOGDIR=$LBPLOGDIR
Restart=on-failure
RestartSec=5
AmbientCapabilities=CAP_NET_BIND_SERVICE

[Install]
WantedBy=multi-user.target
SVCEOF

systemctl daemon-reload
systemctl enable echolox.service
systemctl start echolox.service
echo "<OK> EchoLox service started via systemd"

# ── 9. Icons ──────────────────────────────────────────────────────────────────
ICONDIR="$LBHOMEDIR/webfrontend/html/system/images/icons/EchoLox"
mkdir -p "$ICONDIR"
ICONS_OK=0
for SIZE in 64 128 256 512; do
    if [ -f "$LBPBINDIR/icon_${SIZE}.png" ]; then
        cp "$LBPBINDIR/icon_${SIZE}.png" "$ICONDIR/icon_${SIZE}.png"
        ICONS_OK=$((ICONS_OK + 1))
    fi
done
[ "$ICONS_OK" -gt 0 ] && echo "<OK> EchoLox icons installed ($ICONS_OK sizes)"

# Install the opt-in reverse-proxy helper on every install/update.
cp "$SCRIPT_DIR/scripts/reverse-proxy-setup.sh" "$LBPBINDIR/reverse-proxy-setup.sh"
chmod +x "$LBPBINDIR/reverse-proxy-setup.sh"

# ── 10. sudoers: allow loxberry to restart the service ───────────────────────
SUDOERS_FILE="/etc/sudoers.d/echolox-restart"
cat > "$SUDOERS_FILE" << 'SUDOEOF'
loxberry ALL=(ALL) NOPASSWD: /usr/bin/systemctl restart echolox.service
loxberry ALL=(ALL) NOPASSWD: /bin/systemctl restart echolox.service
SUDOEOF
chmod 0440 "$SUDOERS_FILE"
echo "<OK> EchoLox sudoers rule installed"

REVERSEPROXY_SUDOERS_FILE="/etc/sudoers.d/echolox-reverseproxy"
cat > "$REVERSEPROXY_SUDOERS_FILE" << SUDOEOF
loxberry ALL=(ALL) NOPASSWD: $LBPBINDIR/reverse-proxy-setup.sh --repair
SUDOEOF
chmod 0440 "$REVERSEPROXY_SUDOERS_FILE"
echo "<OK> EchoLox reverse-proxy repair sudoers rule installed"

echo "<OK> EchoLox installed — autostart via systemd enabled"
exit 0
