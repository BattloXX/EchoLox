<?php
require_once "loxberry_web.php";

$cfgfile = $ENV['LBPCFG'] . "/EchoLox.cfg";
$port = 8083;
if (file_exists($cfgfile)) {
    $lines = file($cfgfile);
    foreach ($lines as $line) {
        if (preg_match('/^\s*port:\s*(\d+)/', $line, $m)) {
            $port = (int)$m[1];
            break;
        }
    }
}
$host = $_SERVER['HTTP_HOST'];
$host = preg_replace('/:\d+$/', '', $host);

lbheader("EchoLox", "EchoLox", "");
?>
<iframe
  src="http://<?= htmlspecialchars($host) ?>:<?= $port ?>/ui/"
  style="width:100%;height:calc(100vh - 120px);border:none;"
  title="EchoLox">
</iframe>
<?php lbfooter(); ?>
