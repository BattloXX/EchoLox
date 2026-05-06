package bridge

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"

	"github.com/BattloXX/EchoLox/internal/api"
	"github.com/BattloXX/EchoLox/internal/device"
	"github.com/BattloXX/EchoLox/internal/hue"
	"github.com/BattloXX/EchoLox/internal/identity"
	"github.com/BattloXX/EchoLox/internal/logbuf"
	"github.com/BattloXX/EchoLox/internal/loxone"
	"github.com/BattloXX/EchoLox/internal/migrate"
	"github.com/BattloXX/EchoLox/internal/upnp"
	"github.com/BattloXX/EchoLox/internal/web"
)

func Run(cfg *Config, cfgPath string) error {
	// Wire standard log package to ring-buffer logger
	log.SetOutput(logbuf.Global.Writer())
	log.SetFlags(0) // timestamps handled by logbuf

	// Open log file in LoxBerry log directory
	logDir := lbPath("LBPLOGDIR", "./log")
	if err := os.MkdirAll(logDir, 0755); err == nil {
		logFile := filepath.Join(logDir, "EchoLox.log")
		if err := logbuf.Global.SetFile(logFile); err != nil {
			logbuf.Global.Info("WARNING: cannot open log file %s: %v", logFile, err)
		} else {
			logbuf.Global.Info("Log file: %s", logFile)
		}
	}

	dbPath := filepath.Join(cfg.DataDir, "devices.json")
	mgr, err := device.NewManager(dbPath)
	if err != nil {
		return fmt.Errorf("device manager: %w", err)
	}

	lbs, err := loxone.ReadLoxBerryMiniservers()
	if err != nil {
		logbuf.Global.Info("WARNING: %v — Miniserver commands will be disabled", err)
	}
	var ms *loxone.LBMiniserver
	if lbs != nil {
		if entry, ok := lbs[cfg.Loxone.Miniserver]; ok {
			ms = &entry
		} else {
			for _, v := range lbs {
				cp := v
				ms = &cp
				break
			}
		}
	}

	var loxClient *loxone.Client
	if ms != nil {
		loxClient = loxone.NewClient(ms, cfg.Loxone.Transport, cfg.Loxone.UDPPort)
	}

	verifier := loxone.NewVerifier(loxClient)

	bridgeIP := cfg.Server.IP
	if bridgeIP == "" {
		bridgeIP = autoDetectIP()
	}

	discoveryPort := cfg.Server.DiscoveryPort
	info := identity.New(bridgeIP, cfg.Server.Port, discoveryPort)
	logbuf.Global.Info("Bridge identity: IP=%s  port=%d  discovery-port=%d  bridgeid=%s",
		info.IP, info.Port, info.DiscoveryPort, info.BridgeID)

	mux := http.NewServeMux()

	upnp.RegisterDescription(mux, info)

	hueAPI := hue.NewAPI(mgr, loxClient, verifier, info)
	hueAPI.Register(mux)

	apiHandler := api.NewHandler(mgr, loxClient, verifier, lbs, cfgPath, cfg.DataDir)
	apiHandler.Register(mux)

	webHandler := web.NewHandler(mgr, verifier, lbs, web.WebConfig{
		ServerPort: cfg.Server.Port,
		ServerIP:   bridgeIP,
	})
	webHandler.Register(mux)

	migrateHandler := migrate.NewHandler(mgr)
	migrateHandler.Register(mux)

	go func() {
		upnp.NewListener(info).Listen()
	}()

	// Try to also listen on discovery port when it differs from the main port.
	// This enables direct port-80 access on standalone (non-LoxBerry) installations.
	// On LoxBerry, nginx proxies port 80 -> 8079 so this bind will fail — that's expected.
	if discoveryPort > 0 && discoveryPort != cfg.Server.Port {
		go func() {
			addr80 := fmt.Sprintf(":%d", discoveryPort)
			logbuf.Global.Info("Trying secondary HTTP listener on port %d…", discoveryPort)
			if err := http.ListenAndServe(addr80, mux); err != nil {
				logbuf.Global.Info(
					"Port %d not available (%v). "+
						"Configure nginx proxy: location ~ ^/(api/|description\\.xml$|hue_logo) { proxy_pass http://127.0.0.1:%d; }",
					discoveryPort, err, cfg.Server.Port)
			}
		}()
	}

	addr := fmt.Sprintf(":%d", cfg.Server.Port)
	logbuf.Global.Info("EchoLox HTTP server listening on %s", addr)
	return http.ListenAndServe(addr, mux)
}

func RunCLIImport(from, out string, cfg *Config) {
	if out == "" {
		out = filepath.Join(cfg.DataDir, "devices.json")
	}
	if err := migrate.ImportCLI(from, out); err != nil {
		log.Fatalf("import error: %v", err)
	}
	log.Printf("Import complete → %s", out)
}

func autoDetectIP() string {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		return "0.0.0.0"
	}
	defer conn.Close()
	return conn.LocalAddr().(*net.UDPAddr).IP.String()
}
