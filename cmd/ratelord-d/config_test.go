package main

import (
	"os"
	"strings"
	"testing"
	"time"
)

func TestLoadConfig_PollIntervalValidation(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		envVars     map[string]string
		expectError bool
		errorSubstr string
	}{
		{
			name:        "valid poll interval from flag",
			args:        []string{"-poll-interval", "5s"},
			expectError: false,
		},
		{
			name:        "zero poll interval from flag",
			args:        []string{"-poll-interval", "0s"},
			expectError: true,
			errorSubstr: "poll interval must be positive",
		},
		{
			name:        "negative poll interval from flag",
			args:        []string{"-poll-interval", "-5s"},
			expectError: true,
			errorSubstr: "poll interval must be positive",
		},
		{
			name:        "valid poll interval from env",
			envVars:     map[string]string{"RATELORD_POLL_INTERVAL": "5s"},
			expectError: false,
		},
		{
			name:        "zero poll interval from env",
			envVars:     map[string]string{"RATELORD_POLL_INTERVAL": "0s"},
			expectError: true,
			errorSubstr: "RATELORD_POLL_INTERVAL must be positive",
		},
		{
			name:        "negative poll interval from env",
			envVars:     map[string]string{"RATELORD_POLL_INTERVAL": "-5s"},
			expectError: true,
			errorSubstr: "RATELORD_POLL_INTERVAL must be positive",
		},
		{
			name:        "invalid poll interval format from flag",
			args:        []string{"-poll-interval", "invalid"},
			expectError: true,
			errorSubstr: "invalid poll interval",
		},
		{
			name:        "invalid poll interval format from env",
			envVars:     map[string]string{"RATELORD_POLL_INTERVAL": "invalid"},
			expectError: true,
			errorSubstr: "invalid RATELORD_POLL_INTERVAL",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set up environment variables
			for k, v := range tt.envVars {
				os.Setenv(k, v)
				defer os.Unsetenv(k)
			}

			cfg, err := LoadConfig(tt.args)

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error containing %q, got nil", tt.errorSubstr)
				} else if !strings.Contains(err.Error(), tt.errorSubstr) {
					t.Errorf("expected error containing %q, got %q", tt.errorSubstr, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				} else if cfg.PollInterval <= 0 {
					t.Errorf("expected positive poll interval, got %v", cfg.PollInterval)
				}
			}
		})
	}
}

func TestLoadConfig_DefaultPollInterval(t *testing.T) {
	cfg, err := LoadConfig([]string{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.PollInterval != 10*time.Second {
		t.Errorf("expected default poll interval of 10s, got %v", cfg.PollInterval)
	}
}
