package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	defaultAddr          = "127.0.0.1:8090"
	defaultPollInterval  = 10 * time.Second
	defaultWebAssetsMode = "embedded"
)

type Config struct {
	DBPath        string
	PolicyPath    string
	Addr          string
	PollInterval  time.Duration
	WebAssetsMode string
	WebDir        string
}

func LoadConfig(args []string) (Config, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return Config{}, fmt.Errorf("failed to get cwd: %w", err)
	}

	defaultDBPath := filepath.Join(cwd, "ratelord.db")
	defaultPolicyPath := filepath.Join(cwd, "policy.json")

	dbPath := envOrDefault("RATELORD_DB_PATH", defaultDBPath)
	policyPath := envOrDefaultWithFallback([]string{"RATELORD_POLICY_PATH", "RATELORD_CONFIG_PATH"}, defaultPolicyPath)
	addr := addrFromEnv(defaultAddr)
	pollInterval := defaultPollInterval
	if pollIntervalEnv := os.Getenv("RATELORD_POLL_INTERVAL"); pollIntervalEnv != "" {
		parsed, err := time.ParseDuration(pollIntervalEnv)
		if err != nil {
			return Config{}, fmt.Errorf("invalid RATELORD_POLL_INTERVAL: %w", err)
		}
		pollInterval = parsed
	}
	webAssetsMode := envOrDefault("RATELORD_WEB_ASSETS_MODE", defaultWebAssetsMode)
	webDir := os.Getenv("RATELORD_WEB_DIR")

	flagSet := flag.NewFlagSet("ratelord-d", flag.ContinueOnError)
	flagSet.SetOutput(io.Discard)
	flagDB := flagSet.String("db", dbPath, "path to SQLite database")
	flagPolicy := flagSet.String("policy", policyPath, "path to policy JSON")
	flagAddr := flagSet.String("addr", addr, "HTTP listen address")
	flagPollInterval := flagSet.String("poll-interval", pollInterval.String(), "provider poll interval")
	flagWebAssets := flagSet.String("web-assets", webAssetsMode, "web assets mode: embedded|fs|off")
	flagWebDir := flagSet.String("web-dir", webDir, "web assets directory when web-assets=fs")

	if err := flagSet.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			flagSet.SetOutput(os.Stdout)
			flagSet.PrintDefaults()
			return Config{}, err
		}
		return Config{}, err
	}

	pollIntervalParsed, err := time.ParseDuration(*flagPollInterval)
	if err != nil {
		return Config{}, fmt.Errorf("invalid poll interval: %w", err)
	}

	resolvedDBPath := resolvePath(*flagDB, cwd)
	resolvedPolicyPath := resolvePath(*flagPolicy, cwd)
	mode := normalizeWebAssetsMode(*flagWebAssets)

	config := Config{
		DBPath:        resolvedDBPath,
		PolicyPath:    resolvedPolicyPath,
		Addr:          strings.TrimSpace(*flagAddr),
		PollInterval:  pollIntervalParsed,
		WebAssetsMode: mode,
		WebDir:        strings.TrimSpace(*flagWebDir),
	}

	if config.Addr == "" {
		return Config{}, errors.New("addr cannot be empty")
	}

	if config.WebAssetsMode == "fs" {
		if config.WebDir == "" {
			return Config{}, errors.New("web-assets=fs requires web-dir")
		}
		config.WebDir = resolvePath(config.WebDir, cwd)
	}

	if config.WebAssetsMode != "embedded" && config.WebAssetsMode != "fs" && config.WebAssetsMode != "off" {
		return Config{}, fmt.Errorf("unsupported web-assets mode: %s", config.WebAssetsMode)
	}

	return config, nil
}

func envOrDefault(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func envOrDefaultWithFallback(keys []string, fallback string) string {
	for _, key := range keys {
		if value := os.Getenv(key); value != "" {
			return value
		}
	}
	return fallback
}

func addrFromEnv(fallback string) string {
	if value := os.Getenv("RATELORD_ADDR"); value != "" {
		return value
	}
	if port := os.Getenv("RATELORD_PORT"); port != "" {
		return fmt.Sprintf("127.0.0.1:%s", port)
	}
	return fallback
}

func resolvePath(path string, cwd string) string {
	trimmed := strings.TrimSpace(path)
	if trimmed == "" {
		return trimmed
	}
	if filepath.IsAbs(trimmed) {
		return trimmed
	}
	return filepath.Join(cwd, trimmed)
}

func normalizeWebAssetsMode(mode string) string {
	switch strings.ToLower(strings.TrimSpace(mode)) {
	case "", "embedded":
		return "embedded"
	case "fs", "dir", "directory":
		return "fs"
	case "off", "disabled", "none":
		return "off"
	default:
		return strings.ToLower(strings.TrimSpace(mode))
	}
}
