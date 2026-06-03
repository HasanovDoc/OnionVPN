package singbox

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

type Config struct {
	Route string `json:"route"`
}

func GenerateConfig(baseDir string, routedDomains []string, torSocksPort int) (string, error) {
	configPath := filepath.Join(baseDir, "sing-box-config.json")

	var dnsRoutedDomains []string
	for _, d := range routedDomains {
		if d != "" {
			dnsRoutedDomains = append(dnsRoutedDomains, d)
		}
	}

	singBoxConfig := map[string]interface{}{
		"log": map[string]interface{}{
			"level":  "info",
			"output": filepath.Join(baseDir, "sing-box.log"),
		},
		"dns": map[string]interface{}{
			"servers": []map[string]interface{}{
				{
					"tag":     "dns-direct",
					"address": "1.1.1.1",
					"detour":  "direct",
				},
				{
					"tag":     "dns-tor",
					"address": "https://1.1.1",
					"detour":  "tor-socks",
				},
			},
			"rules": []map[string]interface{}{
				{
					"domain_suffix": []string{".onion"},
					"server":        "dns-tor",
				},
				{
					"domain_keyword": dnsRoutedDomains,
					"server":         "dns-tor",
				},
			},
			"final": "dns-direct",
		},
		"inbounds": []map[string]interface{}{
			{
				"type":           "tun",
				"tag":            "tun-in",
				"interface_name": "OnionVPNTun",
				"address": []string{
					"172.19.0.1/30",
				},
				"auto_route":   true,
				"strict_route": false,
				"stack":        "gvisor",
			},
		},
		"outbounds": []map[string]interface{}{
			{
				"type": "direct",
				"tag":  "direct",
			},
			{
				"type":        "socks",
				"tag":         "tor-socks",
				"server":      "127.0.0.1",
				"server_port": torSocksPort,
				"version":     "5",
			},
		},
		"route": map[string]interface{}{
			"rules": []map[string]interface{}{
				{
					"action":   "hijack-dns",
					"protocol": []string{"dns"},
				},
				{
					"action":  "sniff",
					"inbound": []string{"tun-in"},
				},
				{
					"domain_suffix": []string{".onion"},
					"outbound":      "tor-socks",
				},
				{
					"domain_keyword": dnsRoutedDomains,
					"outbound":       "tor-socks",
				},
				{
					"ip_is_private": true,
					"outbound":      "direct",
				},
			},
			"final":                   "direct",
			"auto_detect_interface":   true,
			"default_domain_resolver": "dns-direct",
		},
	}

	data, err := json.MarshalIndent(singBoxConfig, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal sing-box config: %w", err)
	}

	err = os.WriteFile(configPath, data, 0644)
	if err != nil {
		return "", fmt.Errorf("failed to write sing-box config: %w", err)
	}

	return configPath, nil
}
