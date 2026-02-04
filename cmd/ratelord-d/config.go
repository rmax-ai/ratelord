package main

import (
	"flag"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	defaultPollInterval = 10 * time.Second
	defaultListenAddr   = "127.0.0.1:8090"
	defaultWebMode      = "embedded"
)

type Config struct {
	DBPath       string
	PolicyPath   string
	ListenAddr   string
	PollInterval time.Duration
	WebMode      string
	WebDir       string
}

func LoadConfig() (Config, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return Config{}, fmt.Errorf("failed to get cwd: %w", err)
	}

	defaultDBPath := filepath.Join(cwd, "ratelord.db")
	defaultPolicyPath := filepath.Join(cwd, "policy.json")

	dbDefault := envOrDefault("RATELORD_DB_PATH", defaultDBPath)
	policyDefault := envOrDefault("RATELORD_POLICY_PATH", defaultPolicyPath)
	listenDefault := defaultListenAddr
	if portOverride := envOrDefault("RATELORD_PORT", ""); portOverride != "" {
		listenDefault = net.JoinHostPort("127.0.0.1", portOverride)
	}
	listenDefault = envOrDefault("RATELORD_LISTEN_ADDR", listenDefault)
	webModeDefault := envOrDefault("RATELORD_WEB_MODE", defaultWebMode)
	webDirDefault := envOrDefault("RATELORD_WEB_DIR", "")

	pollDefault, err := envDuration("RATELORD_POLL_INTERVAL", defaultPollInterval)
	if err != nil {
		return Config{}, err
	}

	dbPath := flag.String("db-path", dbDefault, "Path to the SQLite database")
	policyPath := flag.String("policy-path", policyDefault, "Path to the policy JSON file")
	listenAddr := flag.String("listen-addr", listenDefault, "HTTP listen address")
	pollInterval := flag.Duration("poll-interval", pollDefault, "Provider poll interval (e.g. 10s)")
	webMode := flag.String("web-mode", webModeDefault, "Web assets mode: embedded, dir, off")
	webDir := flag.String("web-dir", webDirDefault, "Web assets directory when web-mode=dir")

	flag.Parse()

	cfg := Config{
		DBPath:       strings.TrimSpace(*dbPath),
		PolicyPath:   strings.TrimSpace(*policyPath),
		ListenAddr:   strings.TrimSpace(*listenAddr),
		PollInterval: *pollInterval,
		WebMode:      strings.TrimSpace(*webMode),
		WebDir:       strings.TrimSpace(*webDir),
	}

	return normalizeConfig(cfg)
}

func normalizeConfig(cfg Config) (Config, error) {
	var err error
	if cfg.DBPath == "" {
		return Config{}, fmt.Errorf("db path is required")
	}
	if cfg.PolicyPath == "" {
		return Config{}, fmt.Errorf("policy path is required")
	}
	if cfg.ListenAddr == "" {
		return Config{}, fmt.Errorf("listen address is required")
	}
	if cfg.PollInterval <= 0 {
		return Config{}, fmt.Errorf("poll interval must be positive")
	}

	cfg.DBPath, err = filepath.Abs(cfg.DBPath)
	if err != nil {
		return Config{}, fmt.Errorf("resolve db path: %w", err)
	}
	cfg.PolicyPath, err = filepath.Abs(cfg.PolicyPath)
	if err != nil {
		return Config{}, fmt.Errorf("resolve policy path: %w", err)
	}

	cfg.WebMode = strings.ToLower(cfg.WebMode)
	switch cfg.WebMode {
	case "embedded", "dir", "off":
	default:
		return Config{}, fmt.Errorf("invalid web mode %q (expected embedded, dir, or off)", cfg.WebMode)
	}

	if cfg.WebMode == "dir" {
		if cfg.WebDir == "" {
			return Config{}, fmt.Errorf("web dir is required when web mode is dir")
		}
		cfg.WebDir, err = filepath.Abs(cfg.WebDir)
		if err != nil {
			return Config{}, fmt.Errorf("resolve web dir: %w", err)
		}
		if info, err := os.Stat(cfg.WebDir); err != nil {
			return Config{}, fmt.Errorf("stat web dir: %w", err)
		} else if !info.IsDir() {
			return Config{}, fmt.Errorf("web dir is not a directory: %s", cfg.WebDir)
		}
	}

	return cfg, nil
}

func envOrDefault(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return strings.TrimSpace(value)
	}
	return fallback
}

func envDuration(key string, fallback time.Duration) (time.Duration, error) {
	if value, ok := os.LookupEnv(key); ok {
		value = strings.TrimSpace(value)
		if value == "" {
			return fallback, nil
		}
		parsed, err := time.ParseDuration(value)
		if err != nil {
			return 0, fmt.Errorf("invalid %s duration: %w", key, err)
		}
		return parsed, nil
	}
	return fallback, nil
}
