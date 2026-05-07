#!/bin/bash
# Stop and disable systemd service before plugin removal
systemctl stop echolox.service 2>/dev/null
systemctl disable echolox.service 2>/dev/null
rm -f /etc/systemd/system/echolox.service
systemctl daemon-reload

# Remove Apache2 proxy configuration
if command -v a2disconf >/dev/null 2>&1; then
    a2disconf echolox-hue >/dev/null 2>&1
    rm -f /etc/apache2/conf-available/echolox-hue.conf
    apache2ctl graceful >/dev/null 2>&1
fi

exit 0
