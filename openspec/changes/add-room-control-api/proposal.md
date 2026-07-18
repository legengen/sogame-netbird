## Why

The current NetBird deployment requires an administrator to create a Group and Setup Key for every room. A public room-control API will let users create or join permanent rooms with a room code while keeping the NetBird administrator credential on the server and preserving NetBird as the networking control plane.

## What Changes

- Add a `/rooms` HTTP API for public room creation, room-code joining, and same-room Peer listing.
- Provision one NetBird Group, permanent reusable unlimited Setup Key, and same-Group allow Policy per room.
- Store room metadata and encrypted Setup Key material in a local persistence store.
- Add idempotent provisioning, compensation/reconciliation for partial NetBird API failures, and request rate limits.
- Disable the default `All -> All` allow Policy before enabling room isolation; the existing test Peer is intentionally not migrated.
- Keep the official NetBird Server and Client protocols unchanged; the initial validation client remains the official CLI.

## Capabilities

### New Capabilities

- `room-control-api`: Public room creation/joining, room-code handling, Peer listing, NetBird resource provisioning, lifecycle state, and abuse controls.

### Modified Capabilities

None.

## Impact

- Adds a Room Control API service and persistent room metadata store to the Docker deployment.
- Integrates with the existing NetBird Management REST API using a server-side administrator credential.
- Adds Traefik routing for `/rooms` alongside the existing Dashboard and NetBird routes.
- Changes effective network policy by disabling the default `All -> All` rule; existing unassigned test Peers may lose connectivity.
- Requires secret management for the NetBird PAT, room-code hashes, and encrypted Setup Keys.
- Adds operational concerns for permanently valid, unlimited-use rooms: rate limiting, resource growth, audit logging, and manual room disablement.
