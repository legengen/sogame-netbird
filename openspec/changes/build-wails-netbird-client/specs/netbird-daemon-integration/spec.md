## ADDED Requirements

### Requirement: Official NetBird is the permanent network foundation
The system MUST use the unmodified official NetBird client and daemon for all networking behavior and MUST NOT implement or replace WireGuard, TUN, routing, DNS, ACL, Management, Signal, ICE, STUN, Relay, NAT traversal, or peer encryption.

#### Scenario: Network connection requested
- **WHEN** the desktop client needs to enroll, connect, disconnect, or inspect a peer
- **THEN** it performs the operation through the official NetBird daemon adapter

#### Scenario: New networking feature considered
- **WHEN** a future change requires behavior already owned by NetBird
- **THEN** the change integrates the corresponding official NetBird capability instead of implementing a project-specific network stack

### Requirement: Fixed NetBird version
The system SHALL bundle and support exactly the official NetBird v0.74.7 Windows build, and the deployed NetBird server for this release SHALL also be pinned to v0.74.7 rather than a floating tag.

#### Scenario: Expected daemon version
- **WHEN** the adapter connects to a v0.74.7 daemon
- **THEN** it permits normal operation

#### Scenario: Mismatched daemon version
- **WHEN** the adapter detects any daemon version other than v0.74.7
- **THEN** it refuses room networking operations and offers a repair flow

#### Scenario: Release packaging
- **WHEN** a client release is assembled
- **THEN** the release records and verifies the exact official NetBird artifact version and digest

### Requirement: Verified official distribution
The system SHALL distribute the official signed NetBird Windows artifact unchanged and SHALL verify its cryptographic digest and Windows publisher signature before installation or repair.

#### Scenario: Valid bundled artifact
- **WHEN** installation or repair needs the bundled NetBird artifact
- **THEN** the installer proceeds only after the expected digest and NetBird publisher signature are valid

#### Scenario: Invalid bundled artifact
- **WHEN** the digest or publisher signature does not match
- **THEN** installation stops and reports an integrity failure without executing the artifact

### Requirement: Narrow privilege boundary
The Wails GUI SHALL run without administrator rights, and elevation SHALL occur only for installing, repairing, starting, stopping, or removing the official NetBird system service when Windows requires it.

#### Scenario: Normal room operation
- **WHEN** the user creates, joins, connects, disconnects, or views a room after installation
- **THEN** the GUI completes the operation without an elevation prompt

#### Scenario: Service installation required
- **WHEN** the verified NetBird service is absent or damaged
- **THEN** the client requests one narrowly scoped UAC elevation for the repair operation

### Requirement: Local RPC adapter
The system SHALL control and observe the official daemon through its local RPC interface, SHALL bind communication to the local machine, and SHALL isolate NetBird RPC types behind project-owned adapter DTOs.

#### Scenario: Adapter operation
- **WHEN** the application invokes a daemon action
- **THEN** the Go application layer calls a project-owned adapter interface rather than importing RPC types into Wails bindings or frontend models

#### Scenario: RPC unavailable
- **WHEN** the local daemon RPC endpoint cannot be reached
- **THEN** the adapter reports a typed service-unavailable result and starts bounded recovery or repair behavior

#### Scenario: Remote RPC access
- **WHEN** another machine attempts to reach the desktop daemon control endpoint
- **THEN** the endpoint is not exposed beyond the local machine

### Requirement: Managed NetBird profile
The system SHALL manage one dedicated NetBird profile named `sogame-room` and SHALL not alter unrelated user profiles.

#### Scenario: Existing unrelated profile
- **WHEN** NetBird already contains a profile not owned by this application
- **THEN** the adapter leaves that profile and its credentials unchanged

#### Scenario: Managed profile corruption
- **WHEN** the `sogame-room` profile is missing or inconsistent with saved client state
- **THEN** the client enters a recoverable repair state instead of silently using another profile

### Requirement: NetBird-owned path selection
The system SHALL prefer P2P in its presentation but SHALL allow the official NetBird daemon to select P2P or Relay without client-side forcing or custom candidate logic.

#### Scenario: STUN and direct path available
- **WHEN** NetBird establishes a P2P connection
- **THEN** the client reports the P2P path as preferred

#### Scenario: Direct path unavailable
- **WHEN** NetBird falls back to a functioning Relay connection
- **THEN** the client accepts the tunnel as connected and reports Relay fallback

### Requirement: Bounded recovery
The system SHALL recover from daemon restart, Windows sleep, and network changes using the official daemon identity and bounded retry with backoff.

#### Scenario: Daemon restarts
- **WHEN** the NetBird service restarts while a room is saved
- **THEN** the client reconnects to RPC, reloads normalized state, and does not request a new Setup Key

#### Scenario: Network returns
- **WHEN** Windows connectivity returns after an outage
- **THEN** the client prompts the official daemon to resume and refreshes control-plane and peer state

