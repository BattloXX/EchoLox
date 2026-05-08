package bridge

import (
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Server struct {
		Port          int    `yaml:"port"`
		IP            string `yaml:"ip"`
		DiscoveryPort int    `yaml:"discovery_port"` // port advertised in SSDP (80 for Alexa; 0 = same as Port)
	} `yaml:"server"`
	UPNP struct {
		Name string `yaml:"name"`
		UUID string `yaml:"uuid"`
	} `yaml:"upnp"`
	Loxone struct {
		Miniserver string `yaml:"miniserver"`
		Transport  string `yaml:"transport"`
		UDPPort    int    `yaml:"udp_port"`
	} `yaml:"loxone"`
	MQTT struct {
		Broker   string `yaml:"broker"`
		Username string `yaml:"username"`
		Password string `yaml:"password"`
	} `yaml:"mqtt"`
	DataDir string `yaml:"data_dir"`
}

func DefaultConfig() *Config {
	c := &Config{}
	c.Server.Port = 8079
	c.Server.DiscoveryPort = 80 // Apache proxy forwards :80 → :8079 (set by postinstall.sh)
	c.UPNP.Name = "EchoLox"
	c.Loxone.Miniserver = "1"
	c.Loxone.Transport = "http"
	c.Loxone.UDPPort = 7777
	c.MQTT.Broker = "tcp://localhost:1883"
	c.DataDir = lbPath("LBPDATADIR", "./data")
	return c
}

func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	cfg := DefaultConfig()
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, err
	}
	if cfg.DataDir == "" {
		cfg.DataDir = lbPath("LBPDATADIR", "./data")
	}
	return cfg, nil
}

func lbPath(envKey, fallback string) string {
	if v := os.Getenv(envKey); v != "" {
		return v
	}
	return fallback
}

var (
	DataDir   = lbPath("LBPDATADIR", "./data")
	ConfigDir = lbPath("LBPCFGDIR", "./config")
	LogDir    = lbPath("LBPLOGDIR", "./log")
	BinDir    = lbPath("LBPBINDIR", "./bin")

	_ = filepath.Join // keep import used
)
