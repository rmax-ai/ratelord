# Deployment Guide

`ratelord` is designed as a local-first, daemon-authoritative rate limit governor. While it defaults to `127.0.0.1:8090` for local development, production deployments require careful management of its state (SQLite) and configuration (Policy).

This guide covers deployment via Systemd, Docker, and Kubernetes.

## 1. Overview & Architecture

- **Binary**: `ratelord-d` is a standalone Go binary.
- **State**: Persisted to `ratelord.db` (SQLite WAL mode). **Requires a persistent filesystem.**
- **Configuration**:
  - `policy.yaml` / `policy.yaml`: Defines pools, windows, and burst limits.
  - Environment Variables: Store sensitive tokens (e.g., `GITHUB_TOKEN`, `OPENAI_API_KEY`).
- **Network**: Exposes an HTTP API on port `8090`.
- **Signals**:
  - `SIGINT` / `SIGTERM`: Graceful shutdown (checkpoints state).
  - `SIGHUP`: Hot-reload policy config.

## 2. Prerequisites

1.  **Persistent Storage**: The directory containing `ratelord.db` must persist across restarts. Losing this file resets all rate limit quotas (risk of exhaustion).
2.  **Network**:
    - **Local**: Bind to `127.0.0.1` if co-located on the same host/pod.
    - **Remote**: Bind to `0.0.0.0` if running in a separate container/VM (ensure firewall protection; `ratelord` has no built-in auth).

---

## 3. Systemd (Linux Bare Metal)

Run `ratelord-d` as a system service.

### 3.1. User & Directories

```bash
# Create user
sudo useradd -r -s /bin/false ratelord

# Create state directory
sudo mkdir -p /var/lib/ratelord
sudo chown ratelord:ratelord /var/lib/ratelord

# Place config
sudo mkdir -p /etc/ratelord
sudo cp policy.yaml /etc/ratelord/
```

### 3.2. Unit File (`/etc/systemd/system/ratelord.service`)

```ini
[Unit]
Description=Ratelord Daemon
Documentation=https://github.com/rmax-ai/ratelord
After=network.target

[Service]
Type=simple
User=ratelord
Group=ratelord
# Adjust path to binary
ExecStart=/usr/local/bin/ratelord-d --config /etc/ratelord/policy.yaml --db /var/lib/ratelord/ratelord.db --addr 127.0.0.1:8090
ExecReload=/bin/kill -HUP $MAINPID
KillMode=process
Restart=on-failure
# Environment variables for tokens
EnvironmentFile=/etc/ratelord/ratelord.env

# Hardening
ProtectSystem=full
PrivateTmp=true
NoNewPrivileges=true

[Install]
WantedBy=multi-user.target
```

### 3.3. Management

```bash
sudo systemctl daemon-reload
sudo systemctl enable --now ratelord
sudo journalctl -u ratelord -f
```

---

## 4. Docker

### 4.1. Dockerfile

Multi-stage build to keep the image minimal.

```dockerfile
# Build Stage
FROM golang:1.23-alpine AS builder
WORKDIR /app
COPY . .
RUN go build -o ratelord-d ./cmd/ratelord-d

# Runtime Stage
FROM alpine:3.19
WORKDIR /app
# Install ca-certificates if ratelord needs to make outbound TLS calls (e.g. to upstream APIs)
RUN apk add --no-cache ca-certificates
COPY --from=builder /app/ratelord-d /usr/local/bin/
COPY policy.yaml /etc/ratelord/policy.yaml

# State volume
VOLUME /data
ENV RATELORD_DB_PATH=/data/ratelord.db
ENV RATELORD_CONFIG_PATH=/etc/ratelord/policy.yaml
ENV RATELORD_ADDR=0.0.0.0:8090

CMD ["ratelord-d"]
```

### 4.2. Docker Compose

```yaml
version: '3.8'
services:
  ratelord:
    build: .
    image: ratelord:latest
    restart: always
    volumes:
      - ratelord_data:/data
      - ./policy.yaml:/etc/ratelord/policy.yaml:ro
    environment:
      - GITHUB_TOKEN=${GITHUB_TOKEN}
      - OPENAI_API_KEY=${OPENAI_API_KEY}
    ports:
      - "127.0.0.1:8090:8090" # Bind to host localhost

volumes:
  ratelord_data:
```

---

## 5. Kubernetes (Sidecar Pattern)

The recommended pattern is running `ratelord` as a **sidecar** in the same Pod as your application. This ensures:
1.  **Low Latency**: Application talks to `localhost:8090` over the pod loopback interface.
2.  **Fate Sharing**: They scale and restart together.

### 5.1. Pod Spec Example

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: my-app
spec:
  replicas: 3
  template:
    spec:
      containers:
        # --- Main Application ---
        - name: app
          image: my-app:latest
          env:
            - name: RATELORD_URL
              value: "http://localhost:8090"

        # --- Ratelord Sidecar ---
        - name: ratelord
          image: ratelord:latest
          args:
            - "--addr=127.0.0.1:8090"
            - "--db=/data/ratelord.db"
            - "--config=/etc/config/policy.yaml"
          volumeMounts:
            - name: ratelord-data
              mountPath: /data
            - name: ratelord-config
              mountPath: /etc/config
          envFrom:
            - secretRef:
                name: ratelord-secrets # Contains API tokens

      volumes:
        - name: ratelord-config
          configMap:
            name: ratelord-policy
        - name: ratelord-data
          # Ideally a PVC, or EmptyDir if state loss on restart is acceptable
          # For strict correctness, use a PVC or StatefulSet
          persistentVolumeClaim:
            claimName: ratelord-pvc
```

### 5.2. Note on Horizontal Scaling

If you run multiple replicas of `ratelord` (e.g., 3 sidecars for 3 app replicas), **state is isolated per pod**.
- **Pros**: Zero coordination latency.
- **Cons**: Global limits (e.g., "1000 req/min across all pods") effectively become `N * Limit`.
- **Mitigation**: Use `policy.yaml` to define *per-instance* limits, or divide your global quota by the expected replica count ($Limit_{local} = Limit_{global} / N$).

---

## 6. Configuration & Secrets Management

### Policy (`policy.yaml`)
- Treat as code. Version control it.
- **Updates**:
  - **K8s**: Update ConfigMap -> Wait for volume update -> Send `SIGHUP` to sidecar (or let K8s restart it).
  - **Systemd**: Update file -> `systemctl reload ratelord`.

### Secrets
- **NEVER** put tokens in `policy.yaml`.
- `ratelord-d` reads tokens from environment variables referenced in the policy (if feature supported) or injects them directly if acting as a proxy.
- Use Kubernetes Secrets or `.env` files.

### Monitoring
- Scrape `GET /metrics` (Prometheus format) if enabled.
- Watch logs for `level=error` indicating exhaustion or config errors.
