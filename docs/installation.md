# Installation & Deployment

`ratelord` is a local-first, single-binary daemon. It is designed to run close to your workloads—on the same host, sidecar, or within your cluster.

## 1. Installation

### From Source (Go)

If you have Go installed (1.23+), you can install the latest release directly:

```bash
go install github.com/rmax-ai/ratelord/cmd/ratelord-d@latest
```

This places the `ratelord-d` binary in your `$GOPATH/bin` (usually `~/go/bin`).

### From Release

Download pre-compiled binaries from the [Releases](https://github.com/rmax-ai/ratelord/releases) page (if available).

## 2. Running the Daemon

The daemon requires no external dependencies other than a persistent filesystem for its SQLite event log.

To run it with default settings (listening on port 8090, database in current directory):

```bash
ratelord-d
```

### Basic Flags

```bash
ratelord-d \
  --port 8090 \
  --db /var/lib/ratelord/ratelord.db \
  --policy /etc/ratelord/policy.json
```

## 3. Configuration

`ratelord` can be configured via flags or environment variables. Environment variables take precedence over defaults, but flags override environment variables.

### Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `RATELORD_PORT` | HTTP server port | `8090` |
| `RATELORD_DB_PATH` | Path to the SQLite event log | `./ratelord.db` |
| `RATELORD_POLICY_PATH` | Path to the policy configuration | `./policy.json` |
| `RATELORD_ADVERTISED_URL` | Public URL for this node | `http://localhost:8090` |
| `RATELORD_REDIS_URL` | Optional Redis URL for usage storage | (none) |

*Note: Sensitive tokens (like `GITHUB_TOKEN`) should be passed as standard environment variables and referenced in your `policy.json`.*

## 4. Production Deployment

### Systemd (Linux)

For bare-metal or VM deployments, run `ratelord-d` as a system service.

Create `/etc/systemd/system/ratelord.service`:

```ini
[Unit]
Description=Ratelord Daemon
After=network.target

[Service]
Type=simple
User=ratelord
Group=ratelord
ExecStart=/usr/local/bin/ratelord-d
Environment="RATELORD_PORT=8090"
Environment="RATELORD_DB_PATH=/var/lib/ratelord/ratelord.db"
Environment="RATELORD_POLICY_PATH=/etc/ratelord/policy.json"
# Add provider tokens here or via EnvironmentFile
EnvironmentFile=-/etc/ratelord/ratelord.env
Restart=on-failure

[Install]
WantedBy=multi-user.target
```

Reload and start:

```bash
sudo systemctl daemon-reload
sudo systemctl enable --now ratelord
```

### Docker

You can run `ratelord` as a container. Ensure you mount a volume for the database to persist state.

```bash
docker run -d \
  --name ratelord \
  -p 8090:8090 \
  -v $(pwd)/ratelord-data:/data \
  -v $(pwd)/policy.json:/app/policy.json \
  -e RATELORD_DB_PATH=/data/ratelord.db \
  -e RATELORD_POLICY_PATH=/app/policy.json \
  -e GITHUB_TOKEN=ghp_... \
  ghcr.io/rmax-ai/ratelord:latest
```

*Note: Replace `ghcr.io/rmax-ai/ratelord:latest` with the actual image path if building locally or pulling from a registry.*

#### Docker Compose Example

```yaml
version: '3'
services:
  ratelord:
    image: ghcr.io/rmax-ai/ratelord:latest
    ports:
      - "8090:8090"
    volumes:
      - ./data:/data
      - ./policy.json:/app/policy.json
    environment:
      - RATELORD_DB_PATH=/data/ratelord.db
      - RATELORD_PORT=8090
      - GITHUB_TOKEN=${GITHUB_TOKEN}
```
