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
$cfgfile = "$lbhome/config/plugins/EchoLox/EchoLox.cfg";
if (file_exists($cfgfile)) {
    foreach (file($cfgfile, FILE_IGNORE_NEW_LINES | FILE_SKIP_EMPTY_LINES) as $line) {
        // Match "  port: 80" inside the server: block (indented key)
        if (preg_match('/^\s+port:\s*(\d+)\s*$/', $line, $m)) {
            $port = (int)$m[1];
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

// LoxBerry-integrated wrapper: LoxBerry header/nav + EchoLox as iframe
echo LBWeb::lbheader('EchoLox', '', '');
?>
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
<?php echo LBWeb::lbfooter(); ?>
