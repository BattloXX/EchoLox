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
	"github.com/BattloXX/EchoLox/internal/identity"
	"github.com/BattloXX/EchoLox/internal/loxone"
	"github.com/BattloXX/EchoLox/internal/migrate"
	"github.com/BattloXX/EchoLox/internal/upnp"
	"github.com/BattloXX/EchoLox/internal/web"
)

func Run(cfg *Config, cfgPath string) error {
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

	bridgeIP := cfg.Server.IP
	if bridgeIP == "" {
		bridgeIP = autoDetectIP()
	}

	// BridgeInfo is derived deterministically from IP — UUID/bridgeid stay
	// consistent across restarts so Alexa doesn't lose the bridge.
	info := identity.New(bridgeIP, cfg.Server.Port)
	log.Printf("Bridge identity: IP=%s  bridgeid=%s  UUID=%s", info.IP, info.BridgeID, info.UUID)

	mux := http.NewServeMux()

	upnp.RegisterDescription(mux, info)

	hueAPI := hue.NewAPI(mgr, loxClient, verifier, info)
	hueAPI.Register(mux)

	apiHandler := api.NewHandler(mgr, loxClient, verifier, lbs, cfgPath)
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
	log.Printf("HTTP server listening on %s", addr)
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
