#!/bin/bash
BINDIR="$LBHOMEDIR/bin/plugins/EchoLox"
DAEMONDIR="$LBHOMEDIR/daemon/plugins/EchoLox"
ARCH=$(uname -m)

case "$ARCH" in
  aarch64) cp "$BINDIR/EchoLox-arm64" "$BINDIR/EchoLox" ;;
  armv7l)  cp "$BINDIR/EchoLox-armv7" "$BINDIR/EchoLox" ;;
  x86_64)  cp "$BINDIR/EchoLox-amd64" "$BINDIR/EchoLox" ;;
  *) echo "<FAIL> Unsupported architecture: $ARCH"; exit 2 ;;
esac
chmod +x "$BINDIR/EchoLox"

# Resolve actual home dir at install time so generated scripts work without env
LBHOME="${LBHOMEDIR:-/opt/loxberry}"

# ── LoxBerry daemon init script ────────────────────────────────────────────
mkdir -p "$DAEMONDIR"
cat > "$DAEMONDIR/EchoLox" << DAEMONEOF
#!/bin/bash
LBHOMEDIR="\${LBHOMEDIR:-$LBHOME}"
BINARY="\$LBHOMEDIR/bin/plugins/EchoLox/EchoLox"
CFGFILE="\$LBHOMEDIR/config/plugins/EchoLox/EchoLox.cfg"
PIDFILE="/var/run/EchoLox.pid"

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
cat > /etc/systemd/system/echolox.service << SVCEOF
[Unit]
Description=EchoLox Hue Bridge Emulator for Loxone
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
ExecStart=$LBHOME/bin/plugins/EchoLox/EchoLox --config $LBHOME/config/plugins/EchoLox/EchoLox.cfg
Environment=LBHOMEDIR=$LBHOME
Restart=on-failure
RestartSec=5
PIDFile=/var/run/EchoLox.pid

[Install]
WantedBy=multi-user.target
SVCEOF

systemctl daemon-reload
systemctl enable echolox.service
systemctl start echolox.service

echo "<OK> EchoLox installed and autostart enabled"
exit 0
