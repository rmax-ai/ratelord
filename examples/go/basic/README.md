# Ratelord Go SDK Example

This directory contains a simple example of how to use the `ratelord` Go SDK to negotiate intents with a running daemon.

## Prerequisites

1. A running `ratelord-d` instance on `http://127.0.0.1:8090`.
2. Go 1.22+ installed.

## Usage

Run the example:

```bash
go run main.go
```

## Expected Output

If the daemon is running and policy permits:

```
Daemon Status: ok (Version: v1.0.0)
Asking for permission...
✅ Access Granted! (Intent ID: 550e8400-e29b-41d4-a716-446655440000)
   Doing work...
   Work complete.
```

If the daemon denies the request (e.g., rate limit exceeded):

```
Daemon Status: ok (Version: v1.0.0)
Asking for permission...
❌ Access Denied. Reason: rate_limit_exceeded
```
