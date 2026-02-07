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

	redisclient "github.com/redis/go-redis/v9"
	"github.com/rmax-ai/ratelord/pkg/api"
	"github.com/rmax-ai/ratelord/pkg/engine"
	"github.com/rmax-ai/ratelord/pkg/engine/forecast"
	"github.com/rmax-ai/ratelord/pkg/graph"
	"github.com/rmax-ai/ratelord/pkg/provider"
	"github.com/rmax-ai/ratelord/pkg/provider/federated"
	"github.com/rmax-ai/ratelord/pkg/provider/github"
	"github.com/rmax-ai/ratelord/pkg/provider/openai"
	"github.com/rmax-ai/ratelord/pkg/store"
	"github.com/rmax-ai/ratelord/pkg/store/redis"
	"github.com/rmax-ai/ratelord/web"
)

var (
	Version   = "v1.0.0"
	Commit    = "unknown"
	BuildTime = "unknown"
)

type Config struct {
	DBPath        string
	PolicyPath    string
	Port          int
	WebDir        string
	TLSCert       string
	TLSKey        string
	Mode          string
	LeaderURL     string
	FollowerID    string
	RedisURL      string
	AdvertisedURL string
}

type LeaderServices struct {
	pollerCtx        context.Context
	pollerCancel     context.CancelFunc
	rollupCtx        context.Context
	rollupCancel     context.CancelFunc
	dispatcherCtx    context.Context
	dispatcherCancel context.CancelFunc
	snapshotCtx      context.Context
	snapshotCancel   context.CancelFunc
	pruneCtx         context.Context
	pruneCancel      context.CancelFunc
	poller           *engine.Poller
	rollup           *engine.RollupWorker
	dispatcher       *engine.Dispatcher
	snapshotWorker   *engine.SnapshotWorker
	pruneWorker      *engine.PruneWorker
}

func (ls *LeaderServices) Start() {
	ls.pollerCtx, ls.pollerCancel = context.WithCancel(context.Background())
	go ls.poller.Start(ls.pollerCtx)
	ls.rollupCtx, ls.rollupCancel = context.WithCancel(context.Background())
	go ls.rollup.Run(ls.rollupCtx)
	ls.dispatcherCtx, ls.dispatcherCancel = context.WithCancel(context.Background())
	go ls.dispatcher.Start(ls.dispatcherCtx)
	ls.snapshotCtx, ls.snapshotCancel = context.WithCancel(context.Background())
	go ls.snapshotWorker.Run(ls.snapshotCtx)
	ls.pruneCtx, ls.pruneCancel = context.WithCancel(context.Background())
	go ls.pruneWorker.Run(ls.pruneCtx)
}

func (ls *LeaderServices) Stop() {
	if ls.pollerCancel != nil {
		ls.pollerCancel()
	}
	if ls.rollupCancel != nil {
		ls.rollupCancel()
	}
	if ls.dispatcherCancel != nil {
		ls.dispatcherCancel()
	}
	if ls.snapshotCancel != nil {
		ls.snapshotCancel()
	}
	if ls.pruneCancel != nil {
		ls.pruneCancel()
	}
}

func LoadConfig() Config {
	// Defaults
	cwd, _ := os.Getwd()
	hostname, _ := os.Hostname()
	cfg := Config{
		DBPath:     filepath.Join(cwd, "ratelord.db"),
		PolicyPath: filepath.Join(cwd, "policy.json"),
		Port:       8090,
		Mode:       "leader",
		LeaderURL:  "http://localhost:8090",
		FollowerID: hostname,
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
	if val := os.Getenv("RATELORD_TLS_CERT"); val != "" {
		cfg.TLSCert = val
	}
	if val := os.Getenv("RATELORD_TLS_KEY"); val != "" {
		cfg.TLSKey = val
	}
	if val := os.Getenv("RATELORD_MODE"); val != "" {
		cfg.Mode = val
	}
	if val := os.Getenv("RATELORD_LEADER_URL"); val != "" {
		cfg.LeaderURL = val
	}
	if val := os.Getenv("RATELORD_FOLLOWER_ID"); val != "" {
		cfg.FollowerID = val
	}
	if val := os.Getenv("RATELORD_REDIS_URL"); val != "" {
		cfg.RedisURL = val
	}
	if val := os.Getenv("RATELORD_ADVERTISED_URL"); val != "" {
		cfg.AdvertisedURL = val
	}

	// Flags (override env vars)
	flag.StringVar(&cfg.DBPath, "db", cfg.DBPath, "Path to SQLite database")
	flag.StringVar(&cfg.PolicyPath, "policy", cfg.PolicyPath, "Path to policy file")
	flag.IntVar(&cfg.Port, "port", cfg.Port, "HTTP server port")
	flag.StringVar(&cfg.WebDir, "web-dir", cfg.WebDir, "Path to web assets directory (overrides embedded)")
	flag.StringVar(&cfg.TLSCert, "tls-cert", cfg.TLSCert, "Path to TLS certificate file")
	flag.StringVar(&cfg.TLSKey, "tls-key", cfg.TLSKey, "Path to TLS key file")
	flag.StringVar(&cfg.Mode, "mode", cfg.Mode, "Operation mode: leader, follower")
	flag.StringVar(&cfg.LeaderURL, "leader-url", cfg.LeaderURL, "URL of the leader node (for follower mode)")
	flag.StringVar(&cfg.FollowerID, "follower-id", cfg.FollowerID, "Unique ID for this follower")
	flag.StringVar(&cfg.RedisURL, "redis-url", cfg.RedisURL, "Redis URL for usage storage")
	flag.StringVar(&cfg.AdvertisedURL, "advertised-url", cfg.AdvertisedURL, "Public URL of this node (for leader redirection)")

	flag.Parse()

	// Default AdvertisedURL if not set
	if cfg.AdvertisedURL == "" {
		protocol := "http"
		if cfg.TLSCert != "" {
			protocol = "https"
		}
		// Try to get IP, or just use localhost if not specified
		// In production, user MUST set this. For dev, localhost is fine.
		cfg.AdvertisedURL = fmt.Sprintf("%s://localhost:%d", protocol, cfg.Port)
	}

	return cfg
}

func main() {
	// M21.1: Load Configuration
	cfg := LoadConfig()

	var redisClient *redisclient.Client

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
	var usageStore engine.UsageStore
	if cfg.RedisURL != "" {
		opt, err := redisclient.ParseURL(cfg.RedisURL)
		if err != nil {
			fmt.Printf(`{"level":"fatal","msg":"failed_to_parse_redis_url","error":"%v"}`+"\n", err)
			os.Exit(1)
		}
		redisClient = redisclient.NewClient(opt)
		usageStore = redis.NewRedisUsageStore(redisClient)
		fmt.Printf(`{"level":"info","msg":"using_redis_usage_store","url":"%s"}`+"\n", cfg.RedisURL)
	} else {
		redisClient = nil
		usageStore = engine.NewMemoryUsageStore()
		fmt.Println(`{"level":"info","msg":"using_memory_usage_store"}`)
	}
	usageProj := engine.NewUsageProjectionWithStore(usageStore)

	var leaseStore store.LeaseStore
	if redisClient != nil {
		leaseStore = redis.NewRedisLeaseStore(redisClient)
	} else {
		leaseStore = st
	}

	// Initialize Provider Projection
	providerProj := engine.NewProviderProjection()

	// M34.1: Initialize Cluster Topology Projection
	clusterProj := engine.NewClusterTopology()

	// M35.2: Initialize Graph Projection
	graphProj := graph.NewProjection()

	// M7.3: Initialize Forecast Projection and Forecaster
	forecastProj := forecast.NewForecastProjection(20) // Window size of 20 points
	linearModel := &forecast.LinearModel{}
	forecaster := forecast.NewForecaster(st, forecastProj, linearModel)

	// Replay events to build projection
	// NOTE: This blocks startup, but safe for small event logs
	var since time.Time
	// Try loading from snapshot
	if checkpoint, err := engine.LoadLatestSnapshot(context.Background(), st, identityProj, usageProj, providerProj, forecastProj); err != nil {
		fmt.Printf(`{"level":"warn","msg":"failed_to_load_snapshot","error":"%v"}`+"\n", err)
	} else if !checkpoint.IsZero() {
		since = checkpoint
		fmt.Printf(`{"level":"info","msg":"snapshot_loaded","checkpoint_ts":"%s"}`+"\n", checkpoint)
	}

	events, err := st.ReadEvents(context.Background(), since, 10000) // arbitrary large limit, from beginning or snapshot
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
		// Replay cluster events
		clusterProj.Replay(events)
		fmt.Printf(`{"level":"info","msg":"cluster_projection_replayed"}` + "\n")
		// Replay graph events
		if err := graphProj.Replay(events); err != nil {
			fmt.Printf(`{"level":"error","msg":"failed_to_replay_graph_events","error":"%v"}`+"\n", err)
		} else {
			fmt.Printf(`{"level":"info","msg":"graph_projection_replayed"}` + "\n")
		}
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
	policyEngine := engine.NewPolicyEngine(usageProj, graphProj)

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
	poller := engine.NewPoller(st, 10*time.Second, forecaster, policyCfg) // Poll every 10s for demo

	// Federation: Usage Router
	var usageRouter *federated.UsageRouter

	if cfg.Mode == "follower" {
		fmt.Printf(`{"level":"info","msg":"starting_in_follower_mode","leader_url":"%s","follower_id":"%s"}`+"\n", cfg.LeaderURL, cfg.FollowerID)

		usageRouter = federated.NewUsageRouter()

		// Register Federated Mock Provider
		mockFed := federated.NewFederatedProvider("mock-provider-1", cfg.LeaderURL, cfg.FollowerID)
		mockFed.RegisterPool("default")
		poller.Register(mockFed)
		usageRouter.Register(mockFed)

		// Register Federated GitHub Providers
		if policyCfg != nil {
			for _, ghCfg := range policyCfg.Providers.GitHub {
				ghFed := federated.NewFederatedProvider(ghCfg.ID, cfg.LeaderURL, cfg.FollowerID)
				poller.Register(ghFed)
				usageRouter.Register(ghFed)
			}
			for _, oaCfg := range policyCfg.Providers.OpenAI {
				oaFed := federated.NewFederatedProvider(oaCfg.ID, cfg.LeaderURL, cfg.FollowerID)
				poller.Register(oaFed)
				usageRouter.Register(oaFed)
			}
		}

	} else {
		// Leader / Standalone Mode

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
	}

	// Restore provider state from event stream
	poller.RestoreProviders(providerProj.GetState)
	fmt.Println(`{"level":"info","msg":"restored_provider_state_from_event_stream"}`)

	// M25.2: Initialize Rollup Worker
	rollup := engine.NewRollupWorker(st)

	// M26.2: Initialize Webhook Dispatcher
	dispatcher := engine.NewDispatcher(st)

	// M27.2: Initialize Snapshot Worker
	// Run every 5 minutes by default
	snapshotWorker := engine.NewSnapshotWorker(st, identityProj, usageProj, providerProj, forecastProj, 5*time.Minute)

	// M36.1: Initialize Prune Worker
	// Defaults to disabled if no policy
	var retentionCfg *engine.RetentionConfig
	if policyCfg != nil {
		retentionCfg = policyCfg.Retention
	}
	pruneWorker := engine.NewPruneWorker(st, retentionCfg)

	leaderServices := &LeaderServices{
		poller:         poller,
		rollup:         rollup,
		dispatcher:     dispatcher,
		snapshotWorker: snapshotWorker,
		pruneWorker:    pruneWorker,
	}

	var em *engine.ElectionManager

	if cfg.Mode == "follower" {
		leaderServices.Start()
	} else {
		// Use AdvertisedURL as the holder ID so clients can redirect to it
		holderID := cfg.AdvertisedURL
		em = engine.NewElectionManager(leaseStore, holderID, "ratelord-leader", 5*time.Second, func() {
			fmt.Printf(`{"level":"info","msg":"promoted_to_leader","holder_id":"%s"}`+"\n", holderID)
			leaderServices.Start()
		}, func() {
			fmt.Printf(`{"level":"info","msg":"demoted_from_leader","holder_id":"%s"}`+"\n", holderID)
			leaderServices.Stop()
		})
		emCtx, emCancel := context.WithCancel(context.Background())
		go em.Start(emCtx)
		defer emCancel()
		defer em.Stop(context.Background())
	}

	// M3.1: Start HTTP Server (in background)
	// Use NewServerWithPoller to enable debug endpoints
	addr := fmt.Sprintf(":%d", cfg.Port)
	srv := api.NewServerWithPoller(st, identityProj, usageProj, policyEngine, clusterProj, graphProj, poller, addr)

	if em != nil {
		srv.SetElectionManager(em)
	}

	if usageRouter != nil {
		srv.SetUsageTracker(usageRouter)
	}

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

	// M23.1: Configure TLS if provided
	if cfg.TLSCert != "" && cfg.TLSKey != "" {
		srv.SetTLS(cfg.TLSCert, cfg.TLSKey)
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
				poller.UpdateConfig(cfgReloader)
				pruneWorker.UpdateConfig(cfgReloader.Retention)
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

	leaderServices.Stop()

	// Cleanup
	if err := st.Close(); err != nil {
		fmt.Printf(`{"level":"error","msg":"failed_to_close_store","error":"%v"}`+"\n", err)
	} else {
		fmt.Println(`{"level":"info","msg":"store_closed"}`)
	}

	fmt.Println(`{"level":"info","msg":"shutdown_complete"}`)
}
