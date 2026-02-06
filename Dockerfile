# Minimal Dockerfile for Goreleaser (copies pre-built binary)
# Uses distroless/cc to support CGO (glibc) dependencies
FROM gcr.io/distroless/cc-debian12

COPY ratelord-d /usr/local/bin/ratelord-d

ENTRYPOINT ["/usr/local/bin/ratelord-d"]