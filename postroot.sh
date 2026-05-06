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
PIDFILE="/var/run/EchoLox.pid"
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
if [ ! -f "$CFGFILE" ]; then
  # Migrate from old location (config dir) if present
  if [ -f "$LBPCFGDIR/EchoLox.cfg" ]; then
    cp "$LBPCFGDIR/EchoLox.cfg" "$CFGFILE"
    echo "<OK> EchoLox config migrated from config to data dir"
  else
    cat > "$CFGFILE" << 'CFGEOF'
server:
  port: 8079
  discovery_port: 0

loxone:
  miniserver: "1"
  transport: "http"
  udp_port: 7777

data_dir: ""
CFGEOF
    echo "<OK> EchoLox default config created"
  fi
fi

# ── Try to configure nginx proxy for Alexa port-80 discovery ──────────────
# Alexa (post-2019 firmware) requires the Hue Bridge API on port 80.
# EchoLox runs on port 8079; nginx proxies specific Hue paths to it.
NGINX_PROXY_OK=1
if command -v python3 >/dev/null 2>&1; then
python3 - <<'PYEOF'
import os, subprocess, sys

SITES = [
    '/etc/nginx/sites-enabled/loxberry.conf',
    '/etc/nginx/sites-enabled/loxberry',
    '/etc/nginx/sites-enabled/default',
    '/etc/nginx/conf.d/default.conf',
]

BLOCK = (
    '\n    # echolox-hue-proxy\n'
    '    location ~ ^/(api/|description\\.xml$|hue_logo|favicon\\.ico$) {\n'
    '        proxy_pass http://127.0.0.1:8079;\n'
    '        proxy_set_header Host $host;\n'
    '        proxy_set_header X-Real-IP $remote_addr;\n'
    '        proxy_read_timeout 10s;\n'
    '    }\n'
    '    # /echolox-hue-proxy\n'
)

site = next((s for s in SITES if os.path.exists(s)), None)
if not site:
    print('<WARN> EchoLox: nginx site config not found — configure proxy manually for Alexa (see logs page)')
    sys.exit(1)

content = open(site).read()
if 'echolox-hue-proxy' in content:
    print('<OK> EchoLox nginx proxy already configured')
    sys.exit(0)

idx = content.rfind('}')
if idx < 0:
    print('<WARN> EchoLox: cannot parse nginx config — configure proxy manually')
    sys.exit(1)

open(site + '.echolox.bak', 'w').write(content)
open(site, 'w').write(content[:idx] + BLOCK + content[idx:])

r = subprocess.run(['nginx', '-t'], capture_output=True)
if r.returncode == 0:
    subprocess.run(['nginx', '-s', 'reload'])
    print('<OK> EchoLox nginx proxy configured (port 80 -> 8079 for Hue API)')
    sys.exit(0)
else:
    open(site, 'w').write(content)
    if os.path.exists(site + '.echolox.bak'):
        os.unlink(site + '.echolox.bak')
    print('<WARN> EchoLox: nginx config test failed — reverted. Configure proxy manually.')
    sys.exit(1)
PYEOF
NGINX_PROXY_OK=$?
else
  echo "<WARN> EchoLox: python3 not found — configure nginx proxy manually for Alexa discovery"
fi

# If nginx proxy was configured, set discovery_port: 80 in the config
if [ "$NGINX_PROXY_OK" -eq 0 ] && command -v python3 >/dev/null 2>&1; then
  python3 - "$CFGFILE" <<'PYCFG'
import sys, yaml
path = sys.argv[1]
try:
    with open(path) as f:
        cfg = yaml.safe_load(f) or {}
    if 'server' not in cfg or not isinstance(cfg['server'], dict):
        cfg['server'] = {}
    if cfg['server'].get('discovery_port', 0) == 0:
        cfg['server']['discovery_port'] = 80
        with open(path, 'w') as f:
            yaml.dump(cfg, f, default_flow_style=False, allow_unicode=True)
        print('<OK> EchoLox discovery_port set to 80')
except Exception as e:
    print('<WARN> EchoLox: could not update discovery_port:', e)
PYCFG
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
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
ExecStart=$LBPBINDIR/EchoLox --config $CFGFILE
Environment=LBHOMEDIR=$LBHOMEDIR
Environment=LBPBINDIR=$LBPBINDIR
Environment=LBPCFGDIR=$LBPCFGDIR
Environment=LBPDATADIR=$LBPDATADIR
Environment=LBPLOGDIR=$LBPLOGDIR
PIDFile=$PIDFILE
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

echo "<OK> EchoLox installed — autostart via systemd enabled"
exit 0
