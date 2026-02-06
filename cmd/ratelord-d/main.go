package main

import (
	"context"
	"flag"
	"fmt"
	"io/fs"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"syscall"
	"time"

	"github.com/rmax-ai/ratelord/pkg/api"
	"github.com/rmax-ai/ratelord/pkg/engine"
	"github.com/rmax-ai/ratelord/pkg/engine/forecast"
	"github.com/rmax-ai/ratelord/pkg/provider"
	"github.com/rmax-ai/ratelord/pkg/provider/github"
	"github.com/rmax-ai/ratelord/pkg/provider/openai"
	"github.com/rmax-ai/ratelord/pkg/store"
	"github.com/rmax-ai/ratelord/web"
)

var (
	Version   = "v1.0.0"
	Commit    = "unknown"
	BuildTime = "unknown"
)

type Config struct {
	DBPath     string
	PolicyPath string
	Port       int
	WebDir     string
}

func LoadConfig() Config {
	// Defaults
	cwd, _ := os.Getwd()
	cfg := Config{
		DBPath:     filepath.Join(cwd, "ratelord.db"),
		PolicyPath: filepath.Join(cwd, "policy.json"),
		Port:       8090,
	}

	// Env Vars
	if val := os.Getenv("RATELORD_DB_PATH"); val != "" {
		cfg.DBPath = val
	}
	if val := os.Getenv("RATELORD_POLICY_PATH"); val != "" {
		cfg.PolicyPath = val
	}
	if val := os.Getenv("RATELORD_PORT"); val != "" {
		if p, err := strconv.Atoi(val); err == nil {
			cfg.Port = p
		}
	}
	if val := os.Getenv("RATELORD_WEB_DIR"); val != "" {
		cfg.WebDir = val
	}

	// Flags (override env vars)
	flag.StringVar(&cfg.DBPath, "db", cfg.DBPath, "Path to SQLite database")
	flag.StringVar(&cfg.PolicyPath, "policy", cfg.PolicyPath, "Path to policy file")
	flag.IntVar(&cfg.Port, "port", cfg.Port, "HTTP server port")
	flag.StringVar(&cfg.WebDir, "web-dir", cfg.WebDir, "Path to web assets directory (overrides embedded)")

	flag.Parse()

	return cfg
}

func main() {
	// M21.1: Load Configuration
	cfg := LoadConfig()

	// M1.3: Emit system_started log on boot (structured)
	fmt.Println(`{"level":"info","msg":"system_started","component":"ratelord-d"}`)

	// M2.1: Initialize SQLite Store
	st, err := store.NewStore(cfg.DBPath)
	if err != nil {
		fmt.Printf(`{"level":"fatal","msg":"failed_to_init_store","error":"%v"}`+"\n", err)
		os.Exit(1)
	}
	fmt.Printf(`{"level":"info","msg":"store_initialized","path":"%s"}`+"\n", cfg.DBPath)

	// M4.2: Initialize Identity Projection
	identityProj := engine.NewIdentityProjection()

	// M5.1: Initialize Usage Projection
	usageProj := engine.NewUsageProjection()

	// Initialize Provider Projection
	providerProj := engine.NewProviderProjection()

	// M7.3: Initialize Forecast Projection and Forecaster
	forecastProj := forecast.NewForecastProjection(20) // Window size of 20 points
	linearModel := &forecast.LinearModel{}
	forecaster := forecast.NewForecaster(st, forecastProj, linearModel)

	// Replay events to build projection
	// NOTE: This blocks startup, but safe for small event logs
	events, err := st.ReadEvents(context.Background(), time.Time{}, 10000) // arbitrary large limit, from beginning
	if err == nil {
		// Replay identity events
		if err := identityProj.Replay(events); err != nil {
			fmt.Printf(`{"level":"error","msg":"failed_to_replay_identity_events","error":"%v"}`+"\n", err)
		} else {
			fmt.Printf(`{"level":"info","msg":"identity_projection_replayed","events_count":%d}`+"\n", len(events))
		}
		// Replay usage events
		if err := usageProj.Replay(events); err != nil {
			fmt.Printf(`{"level":"error","msg":"failed_to_replay_usage_events","error":"%v"}`+"\n", err)
		} else {
			fmt.Printf(`{"level":"info","msg":"usage_projection_replayed","events_count":%d}`+"\n", len(events))
		}
		// Replay provider events
		providerProj.Replay(events)
		fmt.Printf(`{"level":"info","msg":"provider_projection_replayed","events_count":%d}`+"\n", len(events))
		// Replay forecast projection
		for _, event := range events {
			if event.EventType == store.EventTypeUsageObserved {
				forecaster.OnUsageObserved(context.Background(), event)
			}
		}
		fmt.Printf(`{"level":"info","msg":"forecast_projection_replayed"}` + "\n")
	} else {
		fmt.Printf(`{"level":"error","msg":"failed_to_read_events","error":"%v"}`+"\n", err)
	}

	// M5.2: Initialize Policy Engine
	policyEngine := engine.NewPolicyEngine(usageProj)

	// M9.3: Initial Policy Load
	var policyCfg *engine.PolicyConfig
	if cfgLoader, err := engine.LoadPolicyConfig(cfg.PolicyPath); err == nil {
		policyCfg = cfgLoader
		policyEngine.UpdatePolicies(cfgLoader)
		fmt.Printf(`{"level":"info","msg":"policy_loaded","path":"%s","policies_count":%d}`+"\n", cfg.PolicyPath, len(cfgLoader.Policies))
	} else if !os.IsNotExist(err) {
		// Log error if file exists but failed to load; ignore if missing (default mode)
		fmt.Printf(`{"level":"error","msg":"failed_to_load_policy","error":"%v"}`+"\n", err)
	}

	// M6.3: Initialize Polling Orchestrator
	// Use the new Poller to drive the provider loop
	poller := engine.NewPoller(st, 10*time.Second, forecaster) // Poll every 10s for demo
	// Register the mock provider (M6.2)
	// IMPORTANT: For the demo, we assume the mock provider is available in the 'pkg/provider' package via a factory or similar,
	// but currently it resides in 'pkg/provider/mock.go' which is in package 'provider'.
	// So we can instantiate it directly.
	mockProv := provider.NewMockProvider("mock-provider-1")
	poller.Register(mockProv)

	// Register GitHub Providers (M14.2)
	if policyCfg != nil {
		for _, ghCfg := range policyCfg.Providers.GitHub {
			token := ""
			if ghCfg.TokenEnvVar != "" {
				token = os.Getenv(ghCfg.TokenEnvVar)
				if token == "" {
					fmt.Printf(`{"level":"warn","msg":"github_token_env_var_empty","env_var":"%s","provider_id":"%s"}`+"\n", ghCfg.TokenEnvVar, ghCfg.ID)
				}
			}
			ghProv := github.NewGitHubProvider(provider.ProviderID(ghCfg.ID), token, ghCfg.EnterpriseURL)
			poller.Register(ghProv)
			fmt.Printf(`{"level":"info","msg":"github_provider_registered","id":"%s"}`+"\n", ghCfg.ID)
		}
		// Register OpenAI Providers
		for _, oaCfg := range policyCfg.Providers.OpenAI {
			token := ""
			if oaCfg.TokenEnvVar != "" {
				token = os.Getenv(oaCfg.TokenEnvVar)
				if token == "" {
					fmt.Printf(`{"level":"warn","msg":"openai_token_env_var_empty","env_var":"%s","provider_id":"%s"}`+"\n", oaCfg.TokenEnvVar, oaCfg.ID)
				}
			}
			oaProv := openai.NewOpenAIProvider(provider.ProviderID(oaCfg.ID), token, oaCfg.OrgID, oaCfg.BaseURL)
			poller.Register(oaProv)
			fmt.Printf(`{"level":"info","msg":"openai_provider_registered","id":"%s"}`+"\n", oaCfg.ID)
		}
	}

	// Restore provider state from event stream
	poller.RestoreProviders(providerProj.GetState)
	fmt.Println(`{"level":"info","msg":"restored_provider_state_from_event_stream"}`)

	// Start Poller in background
	pollerCtx, pollerCancel := context.WithCancel(context.Background())
	defer pollerCancel()
	go poller.Start(pollerCtx)

	// M3.1: Start HTTP Server (in background)
	// Use NewServerWithPoller to enable debug endpoints
	addr := fmt.Sprintf(":%d", cfg.Port)
	srv := api.NewServerWithPoller(st, identityProj, usageProj, policyEngine, poller, addr)

	// Load and set web assets
	var webAssets fs.FS
	if cfg.WebDir != "" {
		webAssets = os.DirFS(cfg.WebDir)
		fmt.Printf(`{"level":"info","msg":"serving_web_assets_from_dir","path":"%s"}`+"\n", cfg.WebDir)
	} else {
		webAssets, err = web.Assets()
		if err != nil {
			fmt.Printf(`{"level":"error","msg":"failed_to_load_web_assets","error":"%v"}`+"\n", err)
		} else {
			fmt.Println(`{"level":"info","msg":"serving_embedded_web_assets"}`)
		}
	}

	if webAssets != nil {
		srv.SetStaticFS(webAssets)
	}

	go func() {
		if err := srv.Start(); err != nil {
			fmt.Printf(`{"level":"error","msg":"server_error","error":"%v"}`+"\n", err)
		}
	}()

	// M1.2: Handle SIGINT/SIGTERM for graceful shutdown
	// M9.3: Handle SIGHUP for policy reload
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)

	// Block until a shutdown signal is received
	var shutdownSig os.Signal
	for {
		sig := <-sigs
		if sig == syscall.SIGHUP {
			fmt.Println(`{"level":"info","msg":"reload_signal_received"}`)
			if cfgReloader, err := engine.LoadPolicyConfig(cfg.PolicyPath); err == nil {
				policyEngine.UpdatePolicies(cfgReloader)
				fmt.Printf(`{"level":"info","msg":"policy_reloaded","policies_count":%d}`+"\n", len(cfgReloader.Policies))
			} else {
				fmt.Printf(`{"level":"error","msg":"failed_to_reload_policy","error":"%v"}`+"\n", err)
			}
			continue
		}

		// If not SIGHUP, it's a shutdown signal
		shutdownSig = sig
		break
	}

	fmt.Printf(`{"level":"info","msg":"shutdown_initiated","signal":"%s"}`+"\n", shutdownSig)

	// Shutdown Server
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Stop(ctx); err != nil {
		fmt.Printf(`{"level":"error","msg":"server_shutdown_error","error":"%v"}`+"\n", err)
	} else {
		fmt.Println(`{"level":"info","msg":"server_stopped"}`)
	}

	// Cleanup
	if err := st.Close(); err != nil {
		fmt.Printf(`{"level":"error","msg":"failed_to_close_store","error":"%v"}`+"\n", err)
	} else {
		fmt.Println(`{"level":"info","msg":"store_closed"}`)
	}

	fmt.Println(`{"level":"info","msg":"shutdown_complete"}`)
}
