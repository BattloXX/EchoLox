<?php
require_once "loxberry_web.php";

// $lbpcfg is set by loxberry_web.php to the plugin config directory
$port = 8083;
$cfgfile = $lbpcfg . "/EchoLox.cfg";
if (file_exists($cfgfile)) {
    foreach (file($cfgfile) as $line) {
        if (preg_match('/^\s*port:\s*(\d+)/', $line, $m)) {
            $port = (int)$m[1];
            break;
        }
    }
}

// Strip port from HTTP_HOST if present — we want the bare IP/hostname
$host = preg_replace('/:\d+$/', '', $_SERVER['HTTP_HOST']);

lbheader("EchoLox", "EchoLox", "");
?>
<iframe
  src="http://<?= htmlspecialchars($host, ENT_QUOTES) ?>:<?= $port ?>/ui/"
  style="width:100%;height:calc(100vh - 120px);border:none;"
  title="EchoLox">
</iframe>
<?php lbfooter(); ?>
