package config

import (
	"os"
	"strconv"
	"strings"
)

// Config holds all runtime configuration loaded from environment variables.
type Config struct {
	NodeRPCURL  string
	NodeRESTURL string
	APIURL      string
	AdminAPIURL string
	Participant string

	Port            int
	RefreshInterval int
	HistoryFile     string
	MaxHistory      int
	StateFile       string

	BlockHeightNodes []string
}

var defaultBlockNodes = []string{
	"http://node1.gonka.ai:8000",
	"http://node2.gonka.ai:8000",
	"https://node3.gonka.ai",
	"http://36.189.234.237:17241",
	"http://47.236.26.199:8000",
	"http://47.236.19.22:18000",
	"http://gonka.spv.re:8000",
}

// Load reads all configuration from environment variables.
func Load() Config {
	c := Config{
		NodeRPCURL:      envOr("NODE_RPC_URL", "http://node:26657"),
		NodeRESTURL:     envOr("NODE_REST_URL", "http://node:1317"),
		APIURL:          envOr("API_URL", "http://api:9000"),
		AdminAPIURL:     envOr("ADMIN_API_URL", "http://api:9200"),
		Participant:     strings.TrimSpace(os.Getenv("PARTICIPANT_ADDRESS")),
		Port:            envInt("EXPORTER_PORT", 9404),
		RefreshInterval: envInt("REFRESH_INTERVAL", 30),
		HistoryFile:     envOr("EPOCH_HISTORY_FILE", "/data/epoch_history.json"),
		MaxHistory:      envInt("MAX_EPOCH_HISTORY", 500),
		StateFile:       envOr("STATE_FILE", "/data/exporter_state.json"),
	}

	c.NodeRPCURL  = strings.TrimRight(c.NodeRPCURL, "/")
	c.NodeRESTURL = strings.TrimRight(c.NodeRESTURL, "/")
	c.APIURL      = strings.TrimRight(c.APIURL, "/")
	c.AdminAPIURL = strings.TrimRight(c.AdminAPIURL, "/")

	raw := os.Getenv("BLOCK_HEIGHT_NODES")
	if raw != "" {
		for _, u := range strings.Split(raw, ",") {
			if u = strings.TrimSpace(u); u != "" {
				c.BlockHeightNodes = append(c.BlockHeightNodes, u)
			}
		}
	} else {
		c.BlockHeightNodes = defaultBlockNodes
	}

	return c
}

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func envInt(key string, def int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return def
}
