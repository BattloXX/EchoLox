<?php
// ─────────────────────────────────────────────────────────────────────────────
// EchoLox – LoxBerry plugin entry page
// Opens EchoLox's own web UI (running on port 80) inside the LoxBerry wrapper.
// ─────────────────────────────────────────────────────────────────────────────
$lbhome = getenv('LBHOMEDIR') ?: '/opt/loxberry';

// Load LoxBerry PHP libraries (graceful: skip if not present)
foreach (["$lbhome/libs/phplib/loxberry_system.php",
          "$lbhome/libs/phplib/loxberry_web.php"] as $lib) {
    if (file_exists($lib)) require_once $lib;
}

// Read server.port from EchoLox YAML config (default: 80)
$port = 80;
$cfgfile = "$lbhome/data/plugins/EchoLox/EchoLox.cfg";
if (!file_exists($cfgfile)) {
    $cfgfile = "$lbhome/config/plugins/EchoLox/EchoLox.cfg";
}
if (file_exists($cfgfile)) {
    $inServer = false;
    foreach (file($cfgfile, FILE_IGNORE_NEW_LINES | FILE_SKIP_EMPTY_LINES) as $line) {
        if (preg_match('/^server:\s*(?:#.*)?$/', $line)) {
            $inServer = true;
            continue;
        }
        if ($inServer && preg_match('/^[^\s#]/', $line)) {
            break;
        }
        if ($inServer && preg_match('/^\s+port:\s*(\d+)\s*(?:#.*)?$/', $line, $m)) {
            $candidate = (int)$m[1];
            if ($candidate >= 1 && $candidate <= 65535) {
                $port = $candidate;
            }
            break;
        }
    }
}

// Build the EchoLox UI URL – always on EchoLox's own port, not the LoxBerry port
$host     = preg_replace('/:\d+$/', '', $_SERVER['HTTP_HOST'] ?? 'localhost');
$portPart = ($port === 80) ? '' : ':' . $port;
$uiUrl    = 'http://' . htmlspecialchars($host, ENT_QUOTES) . $portPart . '/echoloxui/';

// ── Render ───────────────────────────────────────────────────────────────────

if (!class_exists('LBWeb')) {
    // LoxBerry library not available → direct redirect to EchoLox UI
    header("Location: $uiUrl");
    exit;
}

if ($_SERVER['REQUEST_METHOD'] === 'POST' && ($_POST['action'] ?? '') === 'restart') {
    exec('sudo -n /usr/bin/systemctl restart echolox.service 2>&1');
    sleep(2);
    header('Location: ' . ($_SERVER['PHP_SELF'] ?? '/'));
    exit;
}

$socket = @fsockopen('127.0.0.1', $port, $errno, $errstr, 1.0);
$reachable = is_resource($socket);
if ($reachable) {
    fclose($socket);
}

// LoxBerry-integrated wrapper: LoxBerry header/nav + EchoLox as iframe
echo LBWeb::lbheader('EchoLox', '', '');
?>
<?php if (!$reachable):
    $serviceStatus = trim((string)shell_exec('systemctl is-active echolox.service 2>&1'));
    $portUsers = trim((string)shell_exec('ss -tlnp 2>/dev/null | grep -F -- ' . escapeshellarg(':' . $port . ' ')));
    $logfile = "$lbhome/log/plugins/EchoLox/EchoLox.log";
    $logOutput = file_exists($logfile)
        ? trim((string)shell_exec('tail -n 30 ' . escapeshellarg($logfile) . ' 2>&1'))
        : 'Logdatei nicht vorhanden: ' . $logfile;
?>
<div style="margin:20px;padding:16px;border:1px solid #c00;background:#fff4f4;color:#222">
  <h3 style="margin-top:0">EchoLox ist auf Port <?= htmlspecialchars((string)$port, ENT_QUOTES, 'UTF-8') ?> nicht erreichbar</h3>
  <form method="post" style="margin:0 0 14px"><button type="submit" name="action" value="restart">Dienst neu starten</button></form>
  <p>Prüfen Sie bitte:</p>
  <ul>
    <li>LoxBerry-Webport muss ungleich Port <?= htmlspecialchars((string)$port, ENT_QUOTES, 'UTF-8') ?> sein.</li>
    <li>SSDP (lbssdpd) aus.</li>
    <li>Logs im Log Manager.</li>
  </ul>
  <p><strong>systemctl is-active echolox.service</strong></p>
  <pre style="white-space:pre-wrap"><?= htmlspecialchars($serviceStatus, ENT_QUOTES, 'UTF-8') ?></pre>
  <p><strong>Belegung von Port <?= htmlspecialchars((string)$port, ENT_QUOTES, 'UTF-8') ?></strong></p>
  <pre style="white-space:pre-wrap"><?= htmlspecialchars($portUsers !== '' ? $portUsers : 'Keine passende ss-Ausgabe.', ENT_QUOTES, 'UTF-8') ?></pre>
  <p><strong>Letzte 30 Zeilen des EchoLox-Logs</strong></p>
  <pre style="white-space:pre-wrap"><?= htmlspecialchars($logOutput, ENT_QUOTES, 'UTF-8') ?></pre>
</div>
<?php else: ?>
<style>
  /* iframe fills everything below the LoxBerry navbar */
  html, body { margin: 0; overflow: hidden; }
  #echolox-frame { display: block; width: 100%; border: none; background: #f5f5f5; }
</style>
<iframe id="echolox-frame" src="<?= $uiUrl ?>" title="EchoLox UI" scrolling="auto"></iframe>
<script>
(function () {
  var f = document.getElementById('echolox-frame');
  function bottomChromeHeight() {
    var tallest = 0;
    var elements = document.body.querySelectorAll('*');
    for (var i = 0; i < elements.length; i++) {
      var el = elements[i];
      if (el === f || el.contains(f)) continue;
      var style = window.getComputedStyle(el);
      if ((style.position !== 'fixed' && style.position !== 'sticky') ||
          style.display === 'none' || style.visibility === 'hidden') continue;
      var rect = el.getBoundingClientRect();
      if (rect.height <= 0 || rect.width <= 0 || rect.height > window.innerHeight / 2) continue;
      if (Math.abs(rect.bottom - window.innerHeight) <= 6) {
        tallest = Math.max(tallest, rect.height);
      }
    }
    return tallest;
  }
  function fit() {
    var available = window.innerHeight - f.getBoundingClientRect().top - bottomChromeHeight();
    f.style.height = Math.max(0, available) + 'px';
  }
  window.addEventListener('resize', fit);
  if (window.visualViewport) window.visualViewport.addEventListener('resize', fit);
  // Run after layout; retry once to catch deferred nav rendering
  setTimeout(fit, 0);
  setTimeout(fit, 250);
  setTimeout(fit, 1000);
})();
</script>
<?php endif; ?>
<?php echo LBWeb::lbfooter(); ?>
