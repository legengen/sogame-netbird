# NetBird self-hosted deployment

## Endpoints

- Dashboard: https://legengen.top
- Management URL: https://legengen.top
- Embedded OIDC issuer: https://legengen.top/oauth2
- Signal: multiplexed over HTTPS/gRPC on `legengen.top:443`
- Relay: `rels://legengen.top:443` via `/relay`
- STUN: `stun:legengen.top:3478` (UDP)

## Containers

- `netbird-traefik`: TLS termination, HTTP redirect, ACME renewal
- `netbird-dashboard`: web UI
- `netbird-server`: official combined Management, Signal, Relay, STUN, and embedded IdP server

## Published ports

- TCP 80: HTTP to HTTPS redirect and ACME support
- TCP 443: Dashboard, Management API/gRPC, Signal, Relay, and OIDC
- UDP 3478: STUN

## Files and data

- `docker-compose.yml`: Compose topology
- `config.yaml`: combined server configuration and secrets (mode 0600)
- `dashboard.env`: Dashboard endpoints and OIDC settings (mode 0600)
- `getting-started.sh`: official configuration generator used for this deployment
- `sogame-netbird_netbird_data`: SQLite and geolocation data
- `sogame-netbird_netbird_traefik_letsencrypt`: Let's Encrypt account and certificates

Do not run `docker compose down --volumes` unless all NetBird data and certificates
should be removed.

## Operations

```bash
docker compose ps
docker compose logs -f
docker compose restart
docker compose pull
docker compose up -d
```

Docker Hub was not directly reachable during installation. The official images were
downloaded through registry proxies and tagged with their original image names.

## Client enrollment

Interactive browser authentication:

```bash
netbird up --management-url https://legengen.top
```

Headless enrollment after creating a setup key in the Dashboard:

```bash
netbird up --management-url https://legengen.top --setup-key <SETUP_KEY>
```

Check the peer after enrollment:

```bash
netbird status -d
```

## Health note

The combined server multiplexes Relay over the Management HTTP listener. In NetBird
server v0.74.7, the legacy Relay `/health` handler reports `listeners: null` and HTTP
503 because it expects a separate Relay listener. Functional checks pass for the
Dashboard, OIDC discovery, Management REST and gRPC routes, Relay WebSocket route,
metrics listener, and STUN UDP listener.
