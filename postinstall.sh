#!/bin/bash
set -e

CONF=/etc/apache2/conf-available/echolox-hue.conf

cat > "$CONF" <<'EOF'
# EchoLox — Hue Bridge emulator proxy
# Alexa requires the Hue bridge to be reachable on port 80.
# EchoLox itself listens on 8079; Apache proxies the relevant paths.
<IfModule mod_proxy.c>
    ProxyPass        /description.xml http://127.0.0.1:8079/description.xml
    ProxyPassReverse /description.xml http://127.0.0.1:8079/description.xml
    ProxyPass        /api             http://127.0.0.1:8079/api
    ProxyPassReverse /api             http://127.0.0.1:8079/api
</IfModule>
EOF

a2enmod proxy proxy_http
a2enconf echolox-hue
apache2ctl graceful || systemctl reload apache2 || true

exit 0
