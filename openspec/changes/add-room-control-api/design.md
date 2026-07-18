## Context

The server currently runs the official NetBird combined server behind Traefik at `https://legengen.top`. A validated NetBird v0.74.7 flow creates a reusable, never-expiring, unlimited-use Setup Key with `auto_groups`, and a newly enrolled Peer is assigned to that Group. The current account also has an enabled `Default` policy allowing `All -> All`, which must be disabled before room isolation is enabled. The existing test Peer `lzh66` is intentionally not migrated.

The new service is a public, passwordless room-control layer. It does not replace NetBird Management, Signal, Relay, STUN, WireGuard, or the future client. It uses a server-side NetBird administrator PAT and exposes `/rooms` through the existing Traefik host.

## Goals / Non-Goals

**Goals:**

- Let an unauthenticated user create a permanent room and receive its room code and Setup Key.
- Let an unauthenticated user exchange a room code for the room Setup Key.
- Provision one isolated NetBird Group, reusable unlimited Setup Key, and same-Group allow Policy per room.
- List Peer state for a room using its room code.
- Protect the public creation/join surface with rate limits, high-entropy codes, secret handling, idempotency, and audit events.
- Keep provisioning recoverable when NetBird API calls partially fail.

**Non-Goals:**

- Implement a custom NetBird client or graphical client in this change.
- Replace or modify NetBird source code, Signal, Relay, STUN, or WireGuard.
- Add user accounts, passwords, OAuth, room ownership, automatic room expiration, or automatic room destruction.
- Allow cross-room communication or retain the default `All -> All` allow policy.

## Decisions

### Public Room API on `/rooms`

Use the existing domain and a high-priority Traefik path route. `POST /rooms` creates a room; `POST /rooms/join` resolves a code; `GET /rooms/{roomCode}/peers` lists peers. This avoids a DNS change and keeps the public UX passwordless. NetBird's existing `/api`, `/relay`, and `/oauth2` paths remain untouched.

### Server-side NetBird API orchestration

The service calls NetBird REST using a PAT held only in a runtime secret. It creates resources in this order: local room intent, NetBird Group, Setup Key, same-Group Policy, then active room record. The Setup Key request uses `type=reusable`, `expires_in=0`, `usage_limit=0`, `ephemeral=false`, and `auto_groups=[groupID]`. The service stores the returned clear key encrypted because NetBird only returns it during creation.

An account-wide PAT is a high-impact credential; the service must never return it, log it, or expose it through errors. If the deployed NetBird version later supports scoped service credentials, the PAT integration should be replaced with the narrowest available scope.

### SQLite persistence with encrypted secrets

Use SQLite for the learning deployment, stored in a dedicated volume. Store only a hash of the room code for lookup and an encrypted Setup Key ciphertext. Keep `room_id`, NetBird resource IDs, lifecycle state, and timestamps. The encryption key is injected separately from the database and never committed.

### Saga state machine and idempotency

Cross-system creation cannot be atomic. Persist an operation id before external calls and use an idempotency key for `POST /rooms`. Mark rooms `creating`, `active`, `error`, or `disabled`. On failure, revoke/delete resources already created where safe and retain an error record for reconciliation. Retries must discover existing resources by deterministic names or operation ID rather than create duplicates.

### Policy migration before room activation

Disable the current `Default All -> All` policy before enabling room creation. Each room receives a bidirectional `protocol=all` allow rule from its Group to itself. The unassigned test Peer is not migrated; it may lose connectivity after the default rule is disabled. A failed Policy creation prevents the room from becoming active.

### Abuse controls without user authentication

Public creation is intentional, but permanent unlimited rooms create an unbounded resource-abuse risk. Apply IP and endpoint rate limits, request size limits, concurrent provisioning limits, high-entropy non-sequential codes, generic invalid-code responses, and audit logging. Do not impose room expiration or peer-count limits in this change, but expose operational metrics and a manual disable endpoint.

### Peer visibility through the control API

`GET /rooms/{roomCode}/peers` resolves the stored Group ID, queries NetBird Group/Peer APIs, and returns only the room's Peer metadata. It does not grant an additional network permission; WireGuard reachability is governed by the room Policy.

## Risks / Trade-offs

- **Public creation can exhaust NetBird resources** -> Rate limit, cap concurrent provisioning, alert on room/group/key counts, and provide manual disablement.
- **Permanent room codes cannot be recovered if lost** -> Return the code only at creation, use high entropy, and provide no predictable enumeration.
- **Setup Key leakage permits room entry** -> Encrypt at rest, redact logs, use HTTPS, and never expose the PAT; a future client should receive the key only in memory.
- **PAT compromise controls the whole account** -> Keep it out of the database and client, restrict filesystem permissions, rotate it after testing, and migrate to scoped credentials when available.
- **NetBird API and SQLite can diverge** -> Use deterministic names, idempotency records, compensation, and a reconciliation command.
- **Disabling `All -> All` affects existing Peers** -> Perform the policy migration explicitly and document that `lzh66` is not migrated.
- **Unlimited permanent rooms grow data indefinitely** -> Emit counts/metrics and document manual cleanup; do not silently delete rooms.

## Migration Plan

1. Back up the existing NetBird SQLite and configuration volumes.
2. Deploy the Room API in a stopped or private mode and run schema migrations.
3. Disable the `Default All -> All` policy and verify the intended test Peer impact.
4. Deploy the `/rooms` Traefik route and service with PAT and encryption secrets.
5. Create one test room, enroll two test Peers, and verify same-room connectivity and Group membership.
6. Create a second room and verify cross-room traffic is denied.
7. Enable public room creation after rate limits and reconciliation checks pass.

Rollback disables the `/rooms` route and stops new provisioning. It does not delete room resources or re-enable `All -> All` automatically. Re-enabling the default policy is an explicit operator decision because it removes isolation.

## Open Questions

- Should `POST /rooms` return the Setup Key directly, or return a short-lived join response that can be exchanged once by the client?
- What IP/rate limits are acceptable for the learning deployment while preserving passwordless creation?
- Should the first implementation expose only room creation/join/peer listing, or also manual disable and reconciliation endpoints?
- Which service-secret mechanism will be used when this is moved beyond the learning server?
