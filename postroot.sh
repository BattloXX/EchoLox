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

# ── Select architecture-specific binary ────────────────────────────────────
ARCH=$(uname -m)
case "$ARCH" in
  aarch64) cp "$LBPBINDIR/EchoLox-arm64" "$LBPBINDIR/EchoLox" ;;
  armv7l)  cp "$LBPBINDIR/EchoLox-armv7" "$LBPBINDIR/EchoLox" ;;
  x86_64)  cp "$LBPBINDIR/EchoLox-amd64" "$LBPBINDIR/EchoLox" ;;
  *) echo "<FAIL> Unsupported architecture: $ARCH"; exit 2 ;;
esac
chmod +x "$LBPBINDIR/EchoLox"

# ── LoxBerry daemon init script ────────────────────────────────────────────
mkdir -p "$DAEMONDIR"
cat > "$DAEMONDIR/EchoLox" << DAEMONEOF
#!/bin/bash
# Paths are baked in at install time from LoxBerry variables
BINARY="$LBPBINDIR/EchoLox"
CFGFILE="$LBPCFGDIR/EchoLox.cfg"
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
ExecStart=$LBPBINDIR/EchoLox --config $LBPCFGDIR/EchoLox.cfg
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

echo "<OK> EchoLox installed — autostart via systemd enabled"
exit 0
