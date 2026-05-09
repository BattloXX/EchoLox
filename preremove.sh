#!/bin/bash
# EchoLox preremove.sh — runs as root before plugin removal.

systemctl stop echolox.service 2>/dev/null
systemctl disable echolox.service 2>/dev/null
rm -f /etc/systemd/system/echolox.service
systemctl daemon-reload

rm -f /etc/sudoers.d/echolox-restart

exit 0
