<?php
// Determine LoxBerry home dir — getenv() is reliable regardless of loxberry_web.php
$lbhome = getenv('LBHOMEDIR');
if (!$lbhome) $lbhome = '/opt/loxberry';

// Read port from config (default 8083)
$port = 8083;
$cfgfile = "$lbhome/config/plugins/EchoLox/EchoLox.cfg";
if (file_exists($cfgfile)) {
    foreach (file($cfgfile) as $line) {
        if (preg_match('/^\s*port:\s*(\d+)/', $line, $m)) {
            $port = (int)$m[1];
            break;
        }
    }
}

// Build EchoLox URL: same host, different port
$host = preg_replace('/:\d+$/', '', $_SERVER['HTTP_HOST'] ?? 'localhost');
$uiUrl = 'http://' . htmlspecialchars($host, ENT_QUOTES) . ':' . $port . '/ui/';
?>
<!DOCTYPE html>
<html lang="de">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>EchoLox</title>
  <style>
    * { margin: 0; padding: 0; box-sizing: border-box; }
    body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', sans-serif; background: #f0f0f0; }
    .lb-header {
      background: #1a73a7;
      color: white;
      padding: 10px 20px;
      display: flex;
      align-items: center;
      gap: 12px;
      font-size: 1.1em;
      font-weight: bold;
      box-shadow: 0 2px 4px rgba(0,0,0,0.2);
    }
    .lb-header a { color: #cce; font-size: 0.85em; font-weight: normal; margin-left: auto; text-decoration: none; }
    iframe {
      display: block;
      width: 100%;
      height: calc(100vh - 46px);
      border: none;
      background: white;
    }
  </style>
</head>
<body>
  <div class="lb-header">
    EchoLox
    <a href="<?= $uiUrl ?>settings.html">Einstellungen</a>
  </div>
  <iframe src="<?= $uiUrl ?>" title="EchoLox UI"></iframe>
</body>
</html>
