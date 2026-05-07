#!/bin/bash
# LoxBerry sets the following variables before calling postroot.sh:
#   LBHOMEDIR  -- LoxBerry home      (e.g. /opt/loxberry)
#   LBPBINDIR  -- Plugin binary dir  (e.g. /opt/loxberry/bin/plugins/EchoLox)
#   LBPCFGDIR  -- Plugin config dir  (e.g. /opt/loxberry/config/plugins/EchoLox)
#   LBPDATADIR -- Plugin data dir    (e.g. /opt/loxberry/data/plugins/EchoLox)
#   LBPLOGDIR  -- Plugin log dir     (e.g. /opt/loxberry/log/plugins/EchoLox)

# Fallbacks so the script also works when run manually
LBHOMEDIR="${LBHOMEDIR:-/opt/loxberry}"
LBPBINDIR="${LBPBINDIR:-$LBHOMEDIR/bin/plugins/EchoLox}"
LBPCFGDIR="${LBPCFGDIR:-$LBHOMEDIR/config/plugins/EchoLox}"
LBPDATADIR="${LBPDATADIR:-$LBHOMEDIR/data/plugins/EchoLox}"
LBPLOGDIR="${LBPLOGDIR:-$LBHOMEDIR/log/plugins/EchoLox}"

DAEMONDIR="$LBHOMEDIR/daemon/plugins/EchoLox"
PIDFILE="/tmp/EchoLox.pid"
# User config lives in the data dir so it survives plugin updates
CFGFILE="$LBPDATADIR/EchoLox.cfg"

# ── Select architecture-specific binary ────────────────────────────────────
ARCH=$(uname -m)
case "$ARCH" in
  aarch64) cp "$LBPBINDIR/EchoLox-arm64" "$LBPBINDIR/EchoLox" ;;
  armv7l)  cp "$LBPBINDIR/EchoLox-armv7" "$LBPBINDIR/EchoLox" ;;
  x86_64)  cp "$LBPBINDIR/EchoLox-amd64" "$LBPBINDIR/EchoLox" ;;
  *) echo "<FAIL> Unsupported architecture: $ARCH"; exit 2 ;;
esac
chmod +x "$LBPBINDIR/EchoLox"

# ── Create/migrate user config into data dir (survives updates) ─────────────
mkdir -p "$LBPDATADIR"
mkdir -p "$LBPLOGDIR"
chown -R loxberry:loxberry "$LBPDATADIR" "$LBPLOGDIR" 2>/dev/null || true

# Restore devices.json if it was lost during the update
if [ ! -f "$LBPDATADIR/devices.json" ] && [ -f "/tmp/EchoLox_devices.bak" ]; then
    cp "/tmp/EchoLox_devices.bak" "$LBPDATADIR/devices.json"
    chown loxberry:loxberry "$LBPDATADIR/devices.json" 2>/dev/null || true
    echo "<OK> EchoLox devices restored from backup"
fi
rm -f "/tmp/EchoLox_devices.bak"

if [ ! -f "$CFGFILE" ]; then
  # Migrate from old location (config dir) if present
  if [ -f "$LBPCFGDIR/EchoLox.cfg" ]; then
    cp "$LBPCFGDIR/EchoLox.cfg" "$CFGFILE"
    echo "<OK> EchoLox config migrated from config to data dir"
  else
    cat > "$CFGFILE" << CFGEOF
server:
  port: 8079
  discovery_port: 0

loxone:
  miniserver: "1"
  transport: "http"
  udp_port: 7777

data_dir: "$LBPDATADIR"
CFGEOF
    echo "<OK> EchoLox default config created"
  fi
fi

# ── Configure Apache2 proxy for Alexa port-80 discovery ───────────────────
# Alexa (post-2019 firmware) requires the Hue Bridge API on port 80.
# LoxBerry runs Apache2 on port 80; we add a conf that proxies Hue-specific
# paths from port 80 to EchoLox on port 8079 without touching existing config.
APACHE_CONF="/etc/apache2/conf-available/echolox-hue.conf"
PROXY_OK=0

if command -v a2enconf >/dev/null 2>&1; then
    cat > "$APACHE_CONF" << 'APACHEEOF'
# EchoLox Hue Bridge proxy — Alexa (post-2019 firmware) requires port 80.
# Managed by EchoLox postroot.sh — do not edit manually.
ProxyPreserveHost On
# EchoLox web UI and internal management API
ProxyPass /ui/ http://127.0.0.1:8079/ui/
ProxyPassReverse /ui/ http://127.0.0.1:8079/ui/
ProxyPass /echolox/ http://127.0.0.1:8079/echolox/
ProxyPassReverse /echolox/ http://127.0.0.1:8079/echolox/
# Philips Hue API paths required for Alexa discovery
ProxyPassMatch ^(/api(/.*)?|/description\.xml|/hue_logo[^/]*)$ http://127.0.0.1:8079$1
APACHEEOF

    a2enmod proxy proxy_http >/dev/null 2>&1
    a2enconf echolox-hue >/dev/null 2>&1

    if apache2ctl configtest >/dev/null 2>&1; then
        apache2ctl graceful >/dev/null 2>&1
        echo "<OK> EchoLox Apache2 proxy configured (port 80 -> 8079 for Hue API + UI)"
        PROXY_OK=1
    else
        a2disconf echolox-hue >/dev/null 2>&1
        rm -f "$APACHE_CONF"
        apache2ctl graceful >/dev/null 2>&1
        echo "<WARN> EchoLox: Apache2 config test failed — configure proxy manually (see /ui/logs.html)"
    fi
else
    echo "<WARN> EchoLox: apache2 not found — configure proxy manually for Alexa discovery"
fi

# If proxy was configured and discovery_port is not yet set, add it.
# Never overwrite an existing value — the user may have changed it intentionally.
if [ "$PROXY_OK" -eq 1 ] && command -v python3 >/dev/null 2>&1; then
    python3 -c "
import re, sys
with open('$CFGFILE') as f:
    content = f.read()
if 'discovery_port:' not in content:
    content = re.sub(r'(  port:[^\n]*\n)', r'\1  discovery_port: 80\n', content, count=1)
    with open('$CFGFILE', 'w') as f:
        f.write(content)
    print('<OK> EchoLox discovery_port set to 80')
else:
    print('<OK> EchoLox discovery_port already set, not changed')
" 2>/dev/null || echo "<WARN> EchoLox: could not update discovery_port in config"
fi

# ── LoxBerry daemon init script ────────────────────────────────────────────
mkdir -p "$DAEMONDIR"
cat > "$DAEMONDIR/EchoLox" << DAEMONEOF
#!/bin/bash
# Paths are baked in at install time from LoxBerry variables
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

# ── systemd service for reliable autostart ─────────────────────────────────
# Paths are expanded from LoxBerry variables at install time
cat > /etc/systemd/system/echolox.service << SVCEOF
[Unit]
Description=EchoLox Hue Bridge Emulator for Loxone
After=network-online.target apache2.service
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

[Install]
WantedBy=multi-user.target
SVCEOF

systemctl daemon-reload
systemctl enable echolox.service
systemctl start echolox.service

# ── Install plugin icons into LoxBerry web frontend ────────────────────────
# LoxBerry expects icon_64/128/256/512.png in a subdirectory named after the plugin
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

# ── LoxBerry plugin admin redirect ────────────────────────────────────────
# Creates a landing page so the EchoLox UI is reachable from LoxBerry's
# admin panel at /admin/plugins/EchoLox/
PLUGINWEBDIR="$LBHOMEDIR/webfrontend/htmlauth/plugins/EchoLox"
mkdir -p "$PLUGINWEBDIR"
cat > "$PLUGINWEBDIR/index.php" << 'PHPEOF'
<?php header("Location: /ui/"); exit;
PHPEOF
echo "<OK> EchoLox plugin admin page created"

echo "<OK> EchoLox installed — autostart via systemd enabled"
exit 0
