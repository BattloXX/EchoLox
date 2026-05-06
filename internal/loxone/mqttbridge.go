package loxone

import (
	"encoding/json"
	"os"
	"path/filepath"
)

type MQTTSubscription struct {
	Topic  string `json:"topic"`
	Target string `json:"target"`
}

func RegisterMQTTSubscriptions(subscriptions []MQTTSubscription) error {
	lbpcfg := os.Getenv("LBPCFG")
	if lbpcfg == "" {
		lbpcfg = "/opt/loxberry/config/plugins"
	}
	cfgPath := filepath.Join(lbpcfg, "mqttgateway", "echolox_subscriptions.json")
	if err := os.MkdirAll(filepath.Dir(cfgPath), 0755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(subscriptions, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(cfgPath, data, 0644)
}
