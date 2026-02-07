# Configuration Guide

Ratelord is configured through a combination of environment variables (for system settings) and a configuration file (for policies and providers).

## Environment Variables

These variables control the daemon's runtime environment, storage, and network settings.

| Variable | Description | Default | Required |
| :--- | :--- | :--- | :--- |
| `RATELORD_PORT` | The TCP port the daemon listens on. | `8090` | No |
| `RATELORD_DB_PATH` | Path to the SQLite database file (state store). | `./ratelord.db` | No |
| `RATELORD_LOG_LEVEL` | Logging verbosity (`debug`, `info`, `warn`, `error`). | `info` | No |
| `RATELORD_POLICY_PATH` | Path to the declarative policy file (`.json` or `.yaml`). | `./policy.json` | No |
| `RATELORD_TLS_CERT` | Path to TLS certificate for HTTPS. | (Disabled) | No |
| `RATELORD_TLS_KEY` | Path to TLS private key for HTTPS. | (Disabled) | No |
| `RATELORD_REDIS_URL` | Connection string for Redis (if using distributed mode). | (Disabled) | No |
| `RATELORD_WEB_DIR` | Directory containing the web UI assets (for `--web` flag). | (None) | No |
| `RATELORD_MODE` | Operating mode: `leader`, `follower`, or `standalone`. | `leader` | No |
| `RATELORD_LEADER_URL` | URL of the Leader node (only for followers). | `http://localhost:8090` | No |
| `RATELORD_FOLLOWER_ID` | Unique ID for this node when in follower mode. | `hostname` | No |
| `RATELORD_ADVERTISED_URL` | Public URL this node broadcasts to the cluster. | `http://localhost:{port}` | No |
| `RATELORD_ARCHIVE_ENABLED` | Enable cold storage archiving of events. | `false` | No |
| `RATELORD_ARCHIVE_RETENTION` | Retention period for hot events before archiving (e.g., `720h`). | `720h` | No |
| `RATELORD_BLOB_PATH` | Local filesystem path for blob storage (if using local blob store). | `./blobs` | No |

## Policy Configuration

The policy file (JSON or YAML) defines the "brain" of Ratelord: which providers to track and what rules to enforce.

### Structure

The configuration file has four main sections:
1.  **`providers`**: Defines external services to monitor (e.g., GitHub, OpenAI).
2.  **`policies`**: Defines the governance rules applied to intents.
3.  **`pricing`**: (Optional) Defines cost models for financial governance.
4.  **`retention`**: (Optional) Defines data lifecycle rules for events.

### Example: `policy.json`

This example tracks GitHub API limits and enforces a hard stop when the limit is exceeded.

```json
{
  "policies": [
    {
      "id": "github-core-hard-limit",
      "scope": "global",
      "type": "hard",
      "limit": 5000,
      "rules": [
        {
          "name": "deny-exceeded-limit",
          "condition": "remaining < 0",
          "action": "deny",
          "params": {
            "reason": "hard limit exceeded for GitHub core requests"
          }
        }
      ]
    }
  ],
  "providers": {
    "github": [
      {
        "id": "github-main",
        "token_env_var": "GITHUB_TOKEN"
      }
    ]
  }
}
```

### Policy Rules

A Policy is a collection of Rules. Each rule evaluates the current state (forecasts, metadata, time) and decides an action.

-   **`id`**: Unique identifier for the policy.
-   **`scope`**: The scope this policy applies to (e.g., `global`, `env:prod`, `pool:github-core`).
-   **`rules`**: A list of logic predicates.
    -   **`condition`**: A logical expression (e.g., `remaining < 0`, `risk.p_exhaustion > 0.5`).
    -   **`action`**: The outcome if the condition matches.
        -   `approve`: Allow the intent.
        -   `shape`: Delay the intent (`wait_seconds`).
        -   `defer`: Wait until the next reset window.
        -   `deny`: Block the intent immediately.
        -   `switch`: Fail over to a backup identity/pool.

## Provider Configuration

The `providers` section configures the "Ingestion Layer". It tells Ratelord how to connect to external services to poll their usage limits.

### GitHub

Tracks `core`, `search`, and `graphql` rate limits.

```json
"providers": {
  "github": [
    {
      "id": "my-org-github",
      "token_env_var": "GITHUB_TOKEN",
      "enterprise_url": "https://github.example.com/api/v3" 
    }
  ]
}
```

-   `token_env_var`: The environment variable containing the Personal Access Token (PAT).
-   `enterprise_url`: (Optional) Base URL for GitHub Enterprise instances.

### OpenAI

Tracks Request-Per-Minute (RPM) and Token-Per-Minute (TPM) limits.

```json
"providers": {
  "openai": [
    {
      "id": "openai-prod",
      "api_key_env_var": "OPENAI_API_KEY",
      "org_id": "org-12345"
    }
  ]
}
```

-   `api_key_env_var`: The environment variable containing the API Key.
-   `org_id`: (Optional) Organization ID for usage tracking.
-   `base_url`: (Optional) Custom API endpoint (e.g. for Azure OpenAI or proxies).
