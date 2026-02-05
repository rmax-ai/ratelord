package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestLoadConfig_Defaults(t *testing.T) {
	// Setup: Create temp dir and cd into it
	tmpDir := t.TempDir()
	oldWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working directory: %v", err)
	}
	defer os.Chdir(oldWd)
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to change directory: %v", err)
	}

	config, err := LoadConfig([]string{})
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}

	// Verify defaults
	if config.Addr != defaultAddr {
		t.Errorf("expected default addr %s, got %s", defaultAddr, config.Addr)
	}
	if config.PollInterval != defaultPollInterval {
		t.Errorf("expected default poll interval %v, got %v", defaultPollInterval, config.PollInterval)
	}
	if config.WebAssetsMode != defaultWebAssetsMode {
		t.Errorf("expected default web assets mode %s, got %s", defaultWebAssetsMode, config.WebAssetsMode)
	}
	if !strings.HasSuffix(config.DBPath, "ratelord.db") {
		t.Errorf("expected DBPath to end with ratelord.db, got %s", config.DBPath)
	}
	if !strings.HasSuffix(config.PolicyPath, "policy.json") {
		t.Errorf("expected PolicyPath to end with policy.json, got %s", config.PolicyPath)
	}
}

func TestLoadConfig_EnvVars(t *testing.T) {
	tmpDir := t.TempDir()
	oldWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working directory: %v", err)
	}
	defer os.Chdir(oldWd)
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to change directory: %v", err)
	}

	// Set env vars
	os.Setenv("RATELORD_DB_PATH", "/custom/db.db")
	os.Setenv("RATELORD_POLICY_PATH", "/custom/policy.json")
	os.Setenv("RATELORD_ADDR", "0.0.0.0:9000")
	os.Setenv("RATELORD_POLL_INTERVAL", "30s")
	os.Setenv("RATELORD_WEB_ASSETS_MODE", "off")
	defer func() {
		os.Unsetenv("RATELORD_DB_PATH")
		os.Unsetenv("RATELORD_POLICY_PATH")
		os.Unsetenv("RATELORD_ADDR")
		os.Unsetenv("RATELORD_POLL_INTERVAL")
		os.Unsetenv("RATELORD_WEB_ASSETS_MODE")
	}()

	config, err := LoadConfig([]string{})
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}

	if config.DBPath != "/custom/db.db" {
		t.Errorf("expected DBPath from env /custom/db.db, got %s", config.DBPath)
	}
	if config.PolicyPath != "/custom/policy.json" {
		t.Errorf("expected PolicyPath from env /custom/policy.json, got %s", config.PolicyPath)
	}
	if config.Addr != "0.0.0.0:9000" {
		t.Errorf("expected Addr from env 0.0.0.0:9000, got %s", config.Addr)
	}
	if config.PollInterval != 30*time.Second {
		t.Errorf("expected PollInterval from env 30s, got %v", config.PollInterval)
	}
	if config.WebAssetsMode != "off" {
		t.Errorf("expected WebAssetsMode from env off, got %s", config.WebAssetsMode)
	}
}

func TestLoadConfig_FlagOverridesEnv(t *testing.T) {
	tmpDir := t.TempDir()
	oldWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working directory: %v", err)
	}
	defer os.Chdir(oldWd)
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to change directory: %v", err)
	}

	// Set env vars
	os.Setenv("RATELORD_ADDR", "env-addr:8080")
	os.Setenv("RATELORD_POLL_INTERVAL", "15s")
	defer func() {
		os.Unsetenv("RATELORD_ADDR")
		os.Unsetenv("RATELORD_POLL_INTERVAL")
	}()

	// Flags should override env
	config, err := LoadConfig([]string{
		"-addr", "flag-addr:9090",
		"-poll-interval", "25s",
	})
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}

	if config.Addr != "flag-addr:9090" {
		t.Errorf("expected flag to override env for addr, got %s", config.Addr)
	}
	if config.PollInterval != 25*time.Second {
		t.Errorf("expected flag to override env for poll-interval, got %v", config.PollInterval)
	}
}

func TestLoadConfig_InvalidPollInterval_Env(t *testing.T) {
	tmpDir := t.TempDir()
	oldWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working directory: %v", err)
	}
	defer os.Chdir(oldWd)
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to change directory: %v", err)
	}

	os.Setenv("RATELORD_POLL_INTERVAL", "invalid")
	defer os.Unsetenv("RATELORD_POLL_INTERVAL")

	_, err = LoadConfig([]string{})
	if err == nil {
		t.Fatal("expected error for invalid RATELORD_POLL_INTERVAL, got nil")
	}
	if !strings.Contains(err.Error(), "invalid RATELORD_POLL_INTERVAL") {
		t.Errorf("expected error message to contain 'invalid RATELORD_POLL_INTERVAL', got: %v", err)
	}
}

func TestLoadConfig_InvalidPollInterval_Flag(t *testing.T) {
	tmpDir := t.TempDir()
	oldWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working directory: %v", err)
	}
	defer os.Chdir(oldWd)
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to change directory: %v", err)
	}

	_, err = LoadConfig([]string{"-poll-interval", "not-a-duration"})
	if err == nil {
		t.Fatal("expected error for invalid poll-interval flag, got nil")
	}
	if !strings.Contains(err.Error(), "invalid poll interval") {
		t.Errorf("expected error message to contain 'invalid poll interval', got: %v", err)
	}
}

func TestLoadConfig_EmptyAddr(t *testing.T) {
	tmpDir := t.TempDir()
	oldWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working directory: %v", err)
	}
	defer os.Chdir(oldWd)
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to change directory: %v", err)
	}

	_, err = LoadConfig([]string{"-addr", ""})
	if err == nil {
		t.Fatal("expected error for empty addr, got nil")
	}
	if !strings.Contains(err.Error(), "addr cannot be empty") {
		t.Errorf("expected error about empty addr, got: %v", err)
	}
}

func TestLoadConfig_EmptyAddr_Whitespace(t *testing.T) {
	tmpDir := t.TempDir()
	oldWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working directory: %v", err)
	}
	defer os.Chdir(oldWd)
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to change directory: %v", err)
	}

	_, err = LoadConfig([]string{"-addr", "   "})
	if err == nil {
		t.Fatal("expected error for whitespace-only addr, got nil")
	}
	if !strings.Contains(err.Error(), "addr cannot be empty") {
		t.Errorf("expected error about empty addr, got: %v", err)
	}
}

func TestLoadConfig_WebAssets_FS_MissingWebDir(t *testing.T) {
	tmpDir := t.TempDir()
	oldWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working directory: %v", err)
	}
	defer os.Chdir(oldWd)
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to change directory: %v", err)
	}

	_, err = LoadConfig([]string{"-web-assets", "fs"})
	if err == nil {
		t.Fatal("expected error when web-assets=fs but web-dir is missing, got nil")
	}
	if !strings.Contains(err.Error(), "web-assets=fs requires web-dir") {
		t.Errorf("expected error about missing web-dir, got: %v", err)
	}
}

func TestLoadConfig_WebAssets_FS_WithWebDir(t *testing.T) {
	tmpDir := t.TempDir()
	oldWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working directory: %v", err)
	}
	defer os.Chdir(oldWd)
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to change directory: %v", err)
	}

	webDir := filepath.Join(tmpDir, "web")
	config, err := LoadConfig([]string{"-web-assets", "fs", "-web-dir", webDir})
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}

	if config.WebAssetsMode != "fs" {
		t.Errorf("expected WebAssetsMode fs, got %s", config.WebAssetsMode)
	}
	if config.WebDir != webDir {
		t.Errorf("expected WebDir %s, got %s", webDir, config.WebDir)
	}
}

func TestLoadConfig_InvalidWebAssetsMode(t *testing.T) {
	tmpDir := t.TempDir()
	oldWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working directory: %v", err)
	}
	defer os.Chdir(oldWd)
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to change directory: %v", err)
	}

	_, err = LoadConfig([]string{"-web-assets", "invalid-mode"})
	if err == nil {
		t.Fatal("expected error for invalid web-assets mode, got nil")
	}
	if !strings.Contains(err.Error(), "unsupported web-assets mode") {
		t.Errorf("expected error about unsupported web-assets mode, got: %v", err)
	}
}

func TestLoadConfig_PathResolution_Relative(t *testing.T) {
	tmpDir := t.TempDir()
	oldWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working directory: %v", err)
	}
	defer os.Chdir(oldWd)
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to change directory: %v", err)
	}

	config, err := LoadConfig([]string{
		"-db", "data/ratelord.db",
		"-policy", "config/policy.json",
	})
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}

	expectedDB := filepath.Join(tmpDir, "data/ratelord.db")
	expectedPolicy := filepath.Join(tmpDir, "config/policy.json")

	if config.DBPath != expectedDB {
		t.Errorf("expected DBPath %s, got %s", expectedDB, config.DBPath)
	}
	if config.PolicyPath != expectedPolicy {
		t.Errorf("expected PolicyPath %s, got %s", expectedPolicy, config.PolicyPath)
	}
}

func TestLoadConfig_PathResolution_Absolute(t *testing.T) {
	tmpDir := t.TempDir()
	oldWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working directory: %v", err)
	}
	defer os.Chdir(oldWd)
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to change directory: %v", err)
	}

	absDBPath := "/absolute/path/to/db.db"
	absPolicyPath := "/absolute/path/to/policy.json"

	config, err := LoadConfig([]string{
		"-db", absDBPath,
		"-policy", absPolicyPath,
	})
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}

	if config.DBPath != absDBPath {
		t.Errorf("expected DBPath %s, got %s", absDBPath, config.DBPath)
	}
	if config.PolicyPath != absPolicyPath {
		t.Errorf("expected PolicyPath %s, got %s", absPolicyPath, config.PolicyPath)
	}
}

func TestLoadConfig_EnvPortOnly(t *testing.T) {
	tmpDir := t.TempDir()
	oldWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working directory: %v", err)
	}
	defer os.Chdir(oldWd)
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to change directory: %v", err)
	}

	os.Setenv("RATELORD_PORT", "3000")
	defer os.Unsetenv("RATELORD_PORT")

	config, err := LoadConfig([]string{})
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}

	expected := "127.0.0.1:3000"
	if config.Addr != expected {
		t.Errorf("expected addr %s from RATELORD_PORT, got %s", expected, config.Addr)
	}
}

func TestLoadConfig_EnvAddrOverridesPort(t *testing.T) {
	tmpDir := t.TempDir()
	oldWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working directory: %v", err)
	}
	defer os.Chdir(oldWd)
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to change directory: %v", err)
	}

	os.Setenv("RATELORD_ADDR", "0.0.0.0:8080")
	os.Setenv("RATELORD_PORT", "3000")
	defer func() {
		os.Unsetenv("RATELORD_ADDR")
		os.Unsetenv("RATELORD_PORT")
	}()

	config, err := LoadConfig([]string{})
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}

	// RATELORD_ADDR should take precedence over RATELORD_PORT
	if config.Addr != "0.0.0.0:8080" {
		t.Errorf("expected RATELORD_ADDR to override RATELORD_PORT, got %s", config.Addr)
	}
}

func TestLoadConfig_PolicyPathFallback(t *testing.T) {
	tmpDir := t.TempDir()
	oldWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working directory: %v", err)
	}
	defer os.Chdir(oldWd)
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to change directory: %v", err)
	}

	// Test that RATELORD_CONFIG_PATH is used as fallback for RATELORD_POLICY_PATH
	os.Setenv("RATELORD_CONFIG_PATH", "/fallback/config.json")
	defer os.Unsetenv("RATELORD_CONFIG_PATH")

	config, err := LoadConfig([]string{})
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}

	if config.PolicyPath != "/fallback/config.json" {
		t.Errorf("expected PolicyPath from RATELORD_CONFIG_PATH fallback, got %s", config.PolicyPath)
	}
}

func TestLoadConfig_PolicyPathPrimaryOverFallback(t *testing.T) {
	tmpDir := t.TempDir()
	oldWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working directory: %v", err)
	}
	defer os.Chdir(oldWd)
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to change directory: %v", err)
	}

	// Test that RATELORD_POLICY_PATH takes precedence over RATELORD_CONFIG_PATH
	os.Setenv("RATELORD_POLICY_PATH", "/primary/policy.json")
	os.Setenv("RATELORD_CONFIG_PATH", "/fallback/config.json")
	defer func() {
		os.Unsetenv("RATELORD_POLICY_PATH")
		os.Unsetenv("RATELORD_CONFIG_PATH")
	}()

	config, err := LoadConfig([]string{})
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}

	if config.PolicyPath != "/primary/policy.json" {
		t.Errorf("expected PolicyPath from RATELORD_POLICY_PATH, got %s", config.PolicyPath)
	}
}

func TestNormalizeWebAssetsMode(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"", "embedded"},
		{"embedded", "embedded"},
		{"EMBEDDED", "embedded"},
		{"  embedded  ", "embedded"},
		{"fs", "fs"},
		{"FS", "fs"},
		{"dir", "fs"},
		{"directory", "fs"},
		{"off", "off"},
		{"OFF", "off"},
		{"disabled", "off"},
		{"none", "off"},
		{"unknown", "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := normalizeWebAssetsMode(tt.input)
			if result != tt.expected {
				t.Errorf("normalizeWebAssetsMode(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestResolvePath(t *testing.T) {
	cwd := "/test/cwd"

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"empty", "", ""},
		{"whitespace", "   ", ""},
		{"absolute", "/absolute/path", "/absolute/path"},
		{"relative", "relative/path", "/test/cwd/relative/path"},
		{"with_whitespace", "  relative/path  ", "/test/cwd/relative/path"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := resolvePath(tt.input, cwd)
			if result != tt.expected {
				t.Errorf("resolvePath(%q, %q) = %q, want %q", tt.input, cwd, result, tt.expected)
			}
		})
	}
}

func TestLoadConfig_WebDirResolution(t *testing.T) {
	tmpDir := t.TempDir()
	oldWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working directory: %v", err)
	}
	defer os.Chdir(oldWd)
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to change directory: %v", err)
	}

	// Test that web-dir gets resolved when web-assets=fs
	config, err := LoadConfig([]string{
		"-web-assets", "fs",
		"-web-dir", "relative/web",
	})
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}

	expectedWebDir := filepath.Join(tmpDir, "relative/web")
	if config.WebDir != expectedWebDir {
		t.Errorf("expected WebDir to be resolved to %s, got %s", expectedWebDir, config.WebDir)
	}
}

func TestLoadConfig_WebAssets_Embedded(t *testing.T) {
	tmpDir := t.TempDir()
	oldWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working directory: %v", err)
	}
	defer os.Chdir(oldWd)
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to change directory: %v", err)
	}

	config, err := LoadConfig([]string{"-web-assets", "embedded"})
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}

	if config.WebAssetsMode != "embedded" {
		t.Errorf("expected WebAssetsMode embedded, got %s", config.WebAssetsMode)
	}
	// WebDir can be empty for embedded mode
}

func TestLoadConfig_WebAssets_Off(t *testing.T) {
	tmpDir := t.TempDir()
	oldWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working directory: %v", err)
	}
	defer os.Chdir(oldWd)
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to change directory: %v", err)
	}

	config, err := LoadConfig([]string{"-web-assets", "off"})
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}

	if config.WebAssetsMode != "off" {
		t.Errorf("expected WebAssetsMode off, got %s", config.WebAssetsMode)
	}
}
