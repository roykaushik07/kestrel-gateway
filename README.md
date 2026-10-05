# kestrel-gateway

Public API gateway for the Kestrel medical drone delivery platform. It sits between
the ops console (`kestrel-console`) and the backend services (`kestrel-dispatch`,
`kestrel-telemetry`, `kestrel-compliance`).

## What it does

- Validates Okta-issued JWTs against the JWKS endpoint and enforces per-route scopes
- Applies Redis token-bucket rate limits per caller and route
- Reverse-proxies to upstreams declared in `config/routes.yaml`
- Fans the telemetry SSE stream out to console clients at `/v1/stream`
- `plugins/edge-auth` is a Cloudflare Worker that checks mTLS client certificates from
  hospital devices before traffic reaches the gateway

## Local development

```sh
cp .env.example .env
go test ./...
go run ./cmd/gateway
```

Shared types come from `kestrel-contracts` (Go module and the `@kestrel/contracts` npm package).

## Deploy

Merges to `main` build and push `ghcr.io/roykaushik07/kestrel-gateway:<sha>`. Releases are
rolled out by `kestrel-platform`; manifests in `deploy/k8s` are the reference for the Helm chart.
