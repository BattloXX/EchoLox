#!/bin/bash
LBHOMEDIR="${LBHOMEDIR:-/opt/loxberry}"
BINARY="$LBHOMEDIR/bin/plugins/EchoLox/EchoLox"
CFGFILE="$LBHOMEDIR/config/plugins/EchoLox/EchoLox.cfg"
exec "$BINARY" --config "$CFGFILE"
