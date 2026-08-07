#!/bin/bash
set -eu

LBHOMEDIR="${LBHOMEDIR:-/opt/loxberry}"
LBPBINDIR="${LBPBINDIR:-$LBHOMEDIR/bin/plugins/EchoLox}"
LBPDATADIR="${LBPDATADIR:-$LBHOMEDIR/data/plugins/EchoLox}"
VHOST="$LBHOMEDIR/system/apache2/sites-available/000-default.conf"
CFGFILE="$LBPDATADIR/EchoLox.cfg"
BEGIN_MARKER="# BEGIN EchoLox reverse-proxy"
END_MARKER="# END EchoLox reverse-proxy"

usage() {
    echo "Usage: $0 --setup|--repair" >&2
    exit 2
}

repair_proxy() {
    [ -f "$VHOST" ] || { echo "Apache vhost not found: $VHOST" >&2; exit 1; }
    tmpfile=$(mktemp "${VHOST}.echolox.XXXXXX")
    trap 'rm -f "$tmpfile"' EXIT
    awk -v begin="$BEGIN_MARKER" -v end="$END_MARKER" '
      $0 ~ begin { marked=1; next }
      marked && $0 ~ end { marked=0; next }
      marked { next }
      !inserted && /<\/VirtualHost>/ {
        print "    " begin
        print "    ProxyPass        /description.xml http://127.0.0.1:8098/description.xml"
        print "    ProxyPassReverse /description.xml http://127.0.0.1:8098/description.xml"
        print "    ProxyPass        /api/ http://127.0.0.1:8098/api/"
        print "    ProxyPassReverse /api/ http://127.0.0.1:8098/api/"
        print "    ProxyPass        /api http://127.0.0.1:8098/api"
        print "    ProxyPassReverse /api http://127.0.0.1:8098/api"
        print "    ProxyPass        /hue_logo_0.png http://127.0.0.1:8098/hue_logo_0.png"
        print "    ProxyPassReverse /hue_logo_0.png http://127.0.0.1:8098/hue_logo_0.png"
        print "    ProxyPass        /hue_logo_3.png http://127.0.0.1:8098/hue_logo_3.png"
        print "    ProxyPassReverse /hue_logo_3.png http://127.0.0.1:8098/hue_logo_3.png"
        print "    " end
        inserted=1
      }
      { print }
      END { if (!inserted) exit 3 }
    ' "$VHOST" > "$tmpfile" || { echo "No <VirtualHost> block found in $VHOST" >&2; exit 1; }
    chown --reference="$VHOST" "$tmpfile" 2>/dev/null || true
    chmod --reference="$VHOST" "$tmpfile" 2>/dev/null || true
    mv "$tmpfile" "$VHOST"
    trap - EXIT

    a2enmod proxy >/dev/null
    a2enmod proxy_http >/dev/null
    apache2ctl graceful
    echo "EchoLox reverse-proxy block installed"
}

setup_mode() {
    [ -f "$CFGFILE" ] || { echo "EchoLox config not found: $CFGFILE" >&2; exit 1; }
    tmpfile=$(mktemp "${CFGFILE}.echolox.XXXXXX")
    trap 'rm -f "$tmpfile"' EXIT
    awk '
      /^server:[[:space:]]*$/ { inserver=1; print; next }
      inserver && /^[^[:space:]]/ {
        if (!seen_discovery) print "  discovery_port: 80"
        inserver=0
      }
      inserver && /^[[:space:]]+port:[[:space:]]*/ { print "  port: 8098"; seen_port=1; next }
      inserver && /^[[:space:]]+discovery_port:[[:space:]]*/ { print "  discovery_port: 80"; seen_discovery=1; next }
      { print }
      END {
        if (inserver && !seen_discovery) print "  discovery_port: 80"
        if (!seen_port) exit 3
      }
    ' "$CFGFILE" > "$tmpfile" || { echo "server.port not found in $CFGFILE" >&2; exit 1; }
    chown --reference="$CFGFILE" "$tmpfile" 2>/dev/null || true
    chmod --reference="$CFGFILE" "$tmpfile" 2>/dev/null || true
    mv "$tmpfile" "$CFGFILE"
    trap - EXIT
    setcap -r "$LBPBINDIR/EchoLox" 2>/dev/null || true
    repair_proxy
}

case "${1:-}" in
    --repair) repair_proxy ;;
    --setup) setup_mode ;;
    *) usage ;;
esac
