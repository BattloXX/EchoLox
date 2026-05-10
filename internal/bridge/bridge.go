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

func Run(cfg *Config, cfgPath string, version string) error {
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

	apiHandler := api.NewHandler(mgr, loxClient, verifier, lbs, cfgPath, cfg.DataDir, version)
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
	// Fast path: UDP probe learns the preferred outbound IP without sending packets.
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err == nil {
		ip := conn.LocalAddr().(*net.UDPAddr).IP.String()
		conn.Close()
		if ip != "" && ip != "0.0.0.0" {
			return ip
		}
	}
	// Fallback: iterate interfaces and pick the first non-loopback LAN IPv4.
	ifaces, _ := net.Interfaces()
	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 || iface.Flags&net.FlagPointToPoint != 0 {
			continue
		}
		addrs, _ := iface.Addrs()
		for _, addr := range addrs {
			var ip net.IP
			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}
			if ip4 := ip.To4(); ip4 != nil {
				s := ip4.String()
				// Skip link-local (169.254.x.x)
				if !ip.IsLinkLocalUnicast() {
					logbuf.Global.Info("autoDetectIP fallback: using %s from interface %s", s, iface.Name)
					return s
				}
			}
		}
	}
	logbuf.Global.Info("WARNING: could not determine local IP — set server.ip in config")
	return "0.0.0.0"
}
