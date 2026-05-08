#!/bin/bash
set -e

CONF=/etc/apache2/conf-available/echolox-hue.conf

cat > "$CONF" <<'EOF'
# EchoLox — Hue Bridge emulator proxy
# Alexa requires the Hue bridge to be reachable on port 80.
# EchoLox itself listens on 8079; Apache proxies the relevant paths.
ProxyPreserveHost On
# EchoLox Web-UI und Management-API
ProxyPass /echoloxui/ http://127.0.0.1:8079/echoloxui/
ProxyPassReverse /echoloxui/ http://127.0.0.1:8079/echoloxui/
ProxyPass /echolox/ http://127.0.0.1:8079/echolox/
ProxyPassReverse /echolox/ http://127.0.0.1:8079/echolox/
# Philips Hue API-Pfade fuer Alexa
ProxyPassMatch ^(/api(/.*)?|/description\.xml|/hue_logo[^/]*)$ http://127.0.0.1:8079$1
EOF

a2enmod proxy proxy_http
a2enconf echolox-hue
apache2ctl graceful || systemctl reload apache2 || true

echo "<OK> EchoLox Apache2 proxy configured (port 80 -> 8079 for Hue API + UI)"

exit 0
