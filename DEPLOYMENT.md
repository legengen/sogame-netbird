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
- `room-api.env`: local Room API secrets (ignored by Git, mode 0600)
- `sogame-netbird_room_api_data`: persistent Room API SQLite database

Do not run `docker compose down --volumes` unless all NetBird data and certificates
should be removed.

## Room API (optional)

The Room API is a separate Go service. It calls the official Management REST API
with a server-side PAT and does not modify NetBird source code or the official
containers. Public users can create a permanent room and exchange the returned
room code; joining returns the room's reusable Setup Key. The Setup Key is stored
encrypted in SQLite and is only returned by the create/join operations.

Create the local environment file and restrict it to root:

```bash
cp room-api.env.example room-api.env
sed -i "s#replace-with-a-server-side-netbird-pat#<PAT>#" room-api.env
sed -i "s#replace-with-a-random-secret#$(openssl rand -base64 32)#" room-api.env
sed -i "s#replace-with-a-separate-disable-token#$(openssl rand -hex 32)#" room-api.env
chmod 600 room-api.env
```

Start the API behind the existing Traefik HTTPS entrypoint:

```bash
docker compose --profile room-api up -d --build room-api
docker compose ps
curl -fsS https://legengen.top/rooms/healthz
```

The `/rooms` router has priority 200, above the Dashboard catch-all (priority 1),
and uses the same Let's Encrypt certificate resolver. It does not overlap the
official `/api`, `/relay`, `/oauth2`, or gRPC routes.

Before relying on room isolation, run the explicit migration once. This disables
the account-wide `Default All -> All` policy; the existing `lzh66` peer is left
outside every room and therefore loses access until it joins a room:

```bash
docker compose --profile room-api run --rm room-api --disable-default-policy
```

Room API endpoints:

```text
POST /rooms                  -> {room_id, room_code, management_url, setup_key}
POST /rooms/join             -> {room_id, management_url, setup_key}
GET  /rooms/{room_code}/peers
POST /rooms/{room_code}/disable   (X-Room-Admin-Token required)
GET  /rooms/healthz
```

Use an `Idempotency-Key` header on `POST /rooms` when a caller may retry. The
service rate-limits create, join, and peer-list requests and caps JSON bodies.
The room code is permanent until the admin disable endpoint is called.

Rotate the Management PAT by creating a replacement PAT, updating only the local
`room-api.env`, and recreating the service (`docker compose --profile room-api up
-d --force-recreate room-api`). Revoke the old PAT after the new instance is
healthy. Back up the named `room_api_data` volume together with the encryption
key; losing either makes stored Setup Keys unrecoverable. To roll back the API,
stop only the `room-api` service and leave the official NetBird volumes intact.

The future graphical client should call this API, receive `management_url` and a
Setup Key, then manage an embedded official NetBird daemon. It does not require
an alternate Management implementation or a custom WireGuard protocol.

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
