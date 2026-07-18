## ADDED Requirements

### Requirement: Public room creation

The service SHALL expose `POST /rooms` without requiring a user account or password and SHALL return a newly generated high-entropy room code and the corresponding NetBird Setup Key only after provisioning completes successfully.

#### Scenario: Create a room successfully

- **WHEN** a valid client sends `POST /rooms` within its rate limit
- **THEN** the service creates one active room, one NetBird Group, one permanent reusable unlimited-use Setup Key, and one same-Group allow Policy
- **AND** the response contains the room code, room ID, Management URL, and clear Setup Key

#### Scenario: Duplicate creation retry

- **WHEN** the same idempotency key is submitted again
- **THEN** the service returns the original room result without creating duplicate NetBird resources

#### Scenario: NetBird provisioning fails

- **WHEN** any required Group, Setup Key, or Policy operation fails
- **THEN** the service does not return an active room and records a recoverable error with compensation status

### Requirement: Room-code joining

The service SHALL expose `POST /rooms/join` without requiring a user account or password and SHALL return the encrypted-at-rest room's clear Setup Key only when the supplied room code matches an active room.

#### Scenario: Join an active room

- **WHEN** a client submits a valid active room code
- **THEN** the service returns the room ID, Management URL, and Setup Key for that room

#### Scenario: Invalid or disabled room code

- **WHEN** a client submits an unknown, malformed, or disabled room code
- **THEN** the service returns a generic not-found or unavailable response without revealing which condition occurred

### Requirement: NetBird resource provisioning

The service SHALL create Setup Keys with reusable type, `expires_in=0`, `usage_limit=0`, non-ephemeral behavior, and `auto_groups` containing exactly the room Group ID. The service SHALL create a bidirectional allow Policy from the room Group to itself.

#### Scenario: Peer auto-assignment

- **WHEN** a NetBird Client enrolls with a room Setup Key
- **THEN** NetBird assigns the Peer to the room Group automatically

#### Scenario: Same-room communication

- **WHEN** two enrolled Peers belong to the same room Group
- **THEN** the room Policy permits their configured WireGuard traffic through P2P or Relay

#### Scenario: Cross-room isolation

- **WHEN** two enrolled Peers belong to different room Groups and the default `All -> All` policy is disabled
- **THEN** their traffic is denied by the absence of a cross-room allow Policy

### Requirement: Room Peer listing

The service SHALL expose `GET /rooms/{roomCode}/peers` and SHALL return only Peer metadata belonging to the room's NetBird Group.

#### Scenario: List room Peers

- **WHEN** a valid active room code is supplied
- **THEN** the response contains the room's Peer IDs, names, NetBird IPs, connection state, and last-known activity where available

#### Scenario: Peer from another room

- **WHEN** a Peer belongs to another Group
- **THEN** that Peer is absent from the room listing

### Requirement: Policy baseline migration

The deployment SHALL disable the account-wide `Default All -> All` allow Policy before activating room isolation, and SHALL NOT automatically migrate the existing test Peer `lzh66`.

#### Scenario: Disable default policy

- **WHEN** the control plane is enabled
- **THEN** the default All-to-All rule is disabled before the first isolated room is marked active

#### Scenario: Unassigned existing Peer

- **WHEN** `lzh66` remains only in the system `All` Group
- **THEN** it is not granted access to newly created rooms

### Requirement: Secret protection

The service SHALL keep the NetBird PAT server-side, SHALL encrypt Setup Keys at rest, SHALL store only a one-way hash of room codes for lookup, and SHALL redact secrets from logs and error responses.

#### Scenario: Public response does not expose PAT

- **WHEN** a user creates or joins a room
- **THEN** the response contains no NetBird administrator credential

#### Scenario: Secret redaction

- **WHEN** an API request fails or is audited
- **THEN** room Setup Keys and PAT values are absent from logs and diagnostic responses

### Requirement: Public API abuse controls

The service SHALL apply per-client and global rate limits, bounded provisioning concurrency, request size limits, and audit metrics to unauthenticated room operations.

#### Scenario: Rate-limited creation

- **WHEN** a client exceeds the configured room creation rate
- **THEN** the service rejects further creation requests without invoking the NetBird API

#### Scenario: Permanent room retention

- **WHEN** an active room has no current Peers
- **THEN** the service retains the room and its Setup Key until an explicit disable operation is performed
