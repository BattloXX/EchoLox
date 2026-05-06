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

# Install daemon init script
mkdir -p "$DAEMONDIR"
cat > "$DAEMONDIR/EchoLox" << 'EOF'
#!/bin/bash
BINARY="$LBHOMEDIR/bin/plugins/EchoLox/EchoLox"
CFGFILE="$LBHOMEDIR/config/plugins/EchoLox/EchoLox.cfg"
PIDFILE="/var/run/EchoLox.pid"

case "$1" in
  start)
    "$BINARY" --config "$CFGFILE" &
    echo $! > "$PIDFILE"
    echo "<OK> EchoLox started (PID $(cat $PIDFILE))"
    ;;
  stop)
    if [ -f "$PIDFILE" ]; then
      kill $(cat "$PIDFILE") 2>/dev/null && rm -f "$PIDFILE"
      echo "<OK> EchoLox stopped"
    else
      echo "<INFO> EchoLox not running"
    fi
    ;;
  status)
    if [ -f "$PIDFILE" ] && kill -0 $(cat "$PIDFILE") 2>/dev/null; then
      echo "running (PID $(cat $PIDFILE))"
    else
      echo "stopped"
    fi
    ;;
  restart)
    $0 stop
    sleep 1
    $0 start
    ;;
  *)
    echo "Usage: $0 {start|stop|status|restart}"
    exit 1
    ;;
esac
exit 0
EOF
chmod +x "$DAEMONDIR/EchoLox"

exit 0
