#!/bin/bash
# Stop EchoLox before installation or update

# Stop via systemd if available
if systemctl is-active --quiet echolox.service 2>/dev/null; then
    systemctl stop echolox.service
    echo "<OK> EchoLox stopped (systemd)"
fi

# Stop via daemon script if available
DAEMONSCRIPT="${LBHOMEDIR:-/opt/loxberry}/daemon/plugins/EchoLox/EchoLox"
if [ -x "$DAEMONSCRIPT" ]; then
    "$DAEMONSCRIPT" stop 2>/dev/null
fi

# Kill by PID file as fallback
PIDFILE="/var/run/EchoLox.pid"
if [ -f "$PIDFILE" ]; then
    kill "$(cat "$PIDFILE")" 2>/dev/null
    rm -f "$PIDFILE"
fi

# Kill any remaining process by name
pkill -f "bin/plugins/EchoLox/EchoLox" 2>/dev/null

sleep 1
echo "<OK> EchoLox stopped"

# Ensure icon destination exists so the installer can copy the icon there
mkdir -p "${LBHOMEDIR:-/opt/loxberry}/webfrontend/html/system/images/icons/EchoLox" 2>/dev/null

# Remove nginx proxy blocks added by a previous install (postroot.sh will re-add them)
if command -v python3 >/dev/null 2>&1; then
python3 - <<'PYEOF'
import os, subprocess

SITES = [
    '/etc/nginx/sites-enabled/loxberry.conf',
    '/etc/nginx/sites-enabled/loxberry',
    '/etc/nginx/sites-enabled/default',
    '/etc/nginx/conf.d/default.conf',
]

for site in SITES:
    if not os.path.exists(site):
        continue
    content = open(site).read()
    if 'echolox-hue-proxy' not in content:
        continue
    lines = content.splitlines(keepends=True)
    out = []
    skip = False
    for line in lines:
        if '# echolox-hue-proxy' in line and not '# /echolox-hue-proxy' in line:
            skip = True
        if not skip:
            out.append(line)
        if '# /echolox-hue-proxy' in line:
            skip = False
    open(site, 'w').writelines(out)
    r = subprocess.run(['nginx', '-t'], capture_output=True)
    if r.returncode == 0:
        subprocess.run(['nginx', '-s', 'reload'])
        print('<OK> EchoLox nginx proxy removed (will be re-added by postroot.sh)')
    else:
        open(site, 'w').write(content)
        print('<WARN> EchoLox: nginx proxy removal failed — keeping original config')
    break
PYEOF
fi

exit 0
