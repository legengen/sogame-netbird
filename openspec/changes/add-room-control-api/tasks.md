## 1. Service Foundation

- [x] 1.1 Choose the Room API runtime and add its Docker Compose service without changing the official NetBird containers.
- [x] 1.2 Add configuration loading for the NetBird Management URL, PAT secret, encryption key, database path, and rate limits.
- [x] 1.3 Add SQLite migrations for rooms and provisioning operations with unique room-code hashes and NetBird resource IDs.
- [x] 1.4 Add structured audit events and secret-redacting request/error logging.

## 2. NetBird Integration

- [x] 2.1 Implement a typed NetBird REST client for Groups, Setup Keys, Policies, and Peer/Group queries.
- [x] 2.2 Implement deterministic resource names and idempotent lookup for room provisioning retries.
- [x] 2.3 Implement room provisioning as a compensating workflow: Group, permanent reusable unlimited Setup Key, same-Group Policy, then active state.
- [x] 2.4 Implement Setup Key encryption/decryption and ensure the PAT and clear Setup Key never enter logs or responses outside intended room operations.
- [x] 2.5 Implement reconciliation for creating/error rooms and safe Setup Key revocation on failed provisioning.

## 3. Room API

- [x] 3.1 Implement `POST /rooms` with high-entropy code generation, idempotency, public response, and rate/concurrency limits.
- [x] 3.2 Implement `POST /rooms/join` with constant-shape invalid-code handling and encrypted Setup Key retrieval.
- [x] 3.3 Implement `GET /rooms/{roomCode}/peers` using Group membership and NetBird Peer metadata.
- [x] 3.4 Implement explicit room disablement and ensure disabled rooms cannot be joined.
- [x] 3.5 Add request validation, size limits, rate-limit responses, and audit metrics for all public endpoints.

## 4. Policy and Deployment Migration

- [x] 4.1 Add an explicit migration command or guarded startup step to disable the existing `Default All -> All` policy.
- [x] 4.2 Preserve the existing `lzh66` Peer without assigning it to a room and document its expected loss of access.
- [x] 4.3 Add Traefik routing for `/rooms` with priority above the Dashboard catch-all and verify no collision with NetBird `/api`, `/relay`, or `/oauth2` routes.
- [x] 4.4 Add service secret and SQLite volume wiring with restrictive permissions and no secret values committed to the repository.

## 5. Verification

- [x] 5.1 Test room creation and retry idempotency with Postman or automated HTTP tests.
- [ ] 5.2 Enroll two Peers with one room Setup Key and verify both are auto-assigned to the room Group.
- [ ] 5.3 Create a second room and verify same-room traffic is allowed while cross-room traffic is denied after the default policy migration.
- [x] 5.4 Verify invalid/disabled room codes, rate limits, secret redaction, and failed NetBird API compensation.
- [x] 5.5 Verify restart/reconciliation behavior and persistent room/Setup Key state across service restarts.
- [x] 5.6 Document the API contract, PAT rotation, backup/rollback procedure, and the future graphical-client integration boundary.
