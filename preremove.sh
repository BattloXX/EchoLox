#!/bin/bash
# Stop and disable systemd service before plugin removal
systemctl stop echolox.service 2>/dev/null
systemctl disable echolox.service 2>/dev/null
rm -f /etc/systemd/system/echolox.service
systemctl daemon-reload
exit 0
