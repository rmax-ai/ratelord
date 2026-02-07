# Installation & Quickstart

`ratelord` is a local-first, single-binary daemon designed to govern rate limits and constraints.

## 1. Install the Binary

### From Source (Go)

Requires Go 1.23+.

```bash
go install github.com/rmax-ai/ratelord/cmd/ratelord@latest
go install github.com/rmax-ai/ratelord/cmd/ratelord-d@latest
go install github.com/rmax-ai/ratelord/cmd/ratelord-tui@latest
```

### Verification

Verify the installation by checking the versions:

```bash
ratelord version
ratelord-d --version
ratelord-tui --version
```

### From Releases

Download pre-compiled binaries for your platform (macOS, Linux, Windows) from the [GitHub Releases](https://github.com/rmax-ai/ratelord/releases) page.

## 2. Quickstart (Local)

To run Ratelord locally for development:

1.  **Start the Daemon**:
    ```bash
    ratelord-d
    ```
    *It will create a `ratelord.db` in the current directory and listen on port 8090.*

2.  **Run the TUI** (in a separate terminal):
    ```bash
    ratelord-tui
    ```

3.  **Register an Identity**:
    ```bash
    ratelord identity add pat:my-user user
    ```
    *This creates a new identity and prints the access token.*

## 3. Production Deployment

For deploying to production (Systemd, Docker, Kubernetes), please refer to the **[Deployment Guide](guides/deployment.md)**.

## 4. Configuration

For details on `policy.yaml` and environment variables, see the **[Configuration Guide](configuration.md)**.
