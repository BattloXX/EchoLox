package bridge

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"path/filepath"

	"github.com/BattloXX/EchoLox/internal/api"
	"github.com/BattloXX/EchoLox/internal/device"
	"github.com/BattloXX/EchoLox/internal/hue"
	"github.com/BattloXX/EchoLox/internal/loxone"
	"github.com/BattloXX/EchoLox/internal/migrate"
	"github.com/BattloXX/EchoLox/internal/upnp"
	"github.com/BattloXX/EchoLox/internal/web"
)

func Run(cfg *Config) error {
	dbPath := filepath.Join(cfg.DataDir, "devices.json")
	mgr, err := device.NewManager(dbPath)
	if err != nil {
		return fmt.Errorf("device manager: %w", err)
	}

	lbs, _ := loxone.ReadLoxBerryMiniservers()
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

	mux := http.NewServeMux()

	// UPnP description
	bridgeIP := cfg.Server.IP
	if bridgeIP == "" {
		bridgeIP = autoDetectIP()
	}
	upnp.RegisterDescription(mux, bridgeIP, cfg.Server.Port, cfg.UPNP.UUID)

	// Hue API
	hueAPI := hue.NewAPI(mgr, loxClient, verifier)
	hueAPI.Register(mux)

	// Management REST API
	apiHandler := api.NewHandler(mgr, loxClient, verifier, lbs)
	apiHandler.Register(mux)

	// Web UI
	webHandler := web.NewHandler(mgr, verifier, lbs, web.WebConfig{
		ServerPort: cfg.Server.Port,
		ServerIP:   cfg.Server.IP,
	})
	webHandler.Register(mux)

	// Migrate API endpoint
	migrateHandler := migrate.NewHandler(mgr)
	migrateHandler.Register(mux)

	addr := fmt.Sprintf("%s:%d", cfg.Server.IP, cfg.Server.Port)
	log.Printf("HTTP server listening on %s", addr)

	// Start SSDP listener in background
	go func() {
		l, err := upnp.NewListener(bridgeIP, cfg.Server.Port, cfg.UPNP.UUID)
		if err != nil {
			log.Printf("SSDP listener error: %v", err)
			return
		}
		l.Listen()
	}()

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
