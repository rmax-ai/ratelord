#!/bin/bash
cd "$(dirname "$0")"
export GITHUB_TOKEN=${GITHUB_TOKEN:-"your-token-here"}
go run ../../cmd/ratelord-d/main.go
