# ratelord

A local-first constraint control plane for agentic and human-driven software systems.

## Build Instructions

This project uses Make for building and requires Go 1.24 or later.

### Prerequisites

- Go 1.24+
- C compiler (for SQLite CGO bindings)
- Make

### Building

To build all binaries:

```bash
make build
```

To build individual components:

```bash
# Build the daemon
make ratelord-d

# Build the TUI client
make ratelord-tui

# Build the main CLI
make ratelord
```

The binaries will be created in the `bin/` directory.

### Other Commands

- `make install`: Install binaries to GOPATH/bin
- `make clean`: Remove built binaries
- `make test`: Run tests
- `make generate`: Generate code (run before build)