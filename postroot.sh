#!/bin/bash
BINDIR="$LBHOMEDIR/bin/plugins/EchoLox"
ARCH=$(uname -m)

case "$ARCH" in
  aarch64) cp "$BINDIR/EchoLox-arm64" "$BINDIR/EchoLox" ;;
  armv7l)  cp "$BINDIR/EchoLox-armv7" "$BINDIR/EchoLox" ;;
  x86_64)  cp "$BINDIR/EchoLox-amd64" "$BINDIR/EchoLox" ;;
  *) echo "<FAIL> Unsupported architecture: $ARCH"; exit 2 ;;
esac

chmod +x "$BINDIR/EchoLox"
exit 0
