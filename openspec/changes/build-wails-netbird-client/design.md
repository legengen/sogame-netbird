## Context

The repository currently deploys the official combined NetBird v0.74.7 server and a separate Room API. Two official clients have validated room enrollment, same-room policy assignment, Relay fallback, STUN discovery, and direct P2P WireGuard connectivity. The missing product layer is a simple Windows desktop client that hides Setup Key handling and presents room-oriented connection state.

The desktop application must never become a second networking implementation. Official NetBird remains the permanent owner of daemon identity, TUN and WireGuard interfaces, routing, DNS, ACL enforcement, Management and Signal streams, ICE/STUN discovery, Relay fallback, encryption, and reconnection. Wails owns UI and application orchestration only.

## Goals / Non-Goals

**Goals:**

- Deliver a Windows 10/11 x64 Wails application with a Go backend and React/TypeScript frontend.
- Bundle and verify the official signed NetBird v0.74.7 Windows distribution.
- Manage the privileged official daemon from an unprivileged GUI through local RPC.
- Support one saved room, one managed profile, and one active room connection.
- Make disconnect, reconnect, leave, switch, tray, and GUI-exit behavior predictable.
- Treat P2P and Relay as the only successful peer tunnel states while representing an empty room as waiting rather than failed.
- Keep Setup Keys out of persistent storage and frontend state and protect the Room Code with Windows-native encryption.

**Non-Goals:**

- Implement, fork, or modify NetBird networking protocols or the NetBird daemon.
- Support macOS, Linux, Windows ARM64, multiple saved rooms, or simultaneous room connections.
- Add chat, file transfer, game discovery, room ownership, or administrator room disablement.
- Add an independent auto-update framework in this change.
- Change the existing Room API HTTP contract.

## Decisions

### Official NetBird is a fixed external subsystem

Package the unmodified official signed NetBird v0.74.7 Windows artifact and pin the server image to the same version. The build pipeline records the artifact URL, digest, and expected Windows publisher identity. The installer verifies these values before invoking the official installation path.

Floating versions and daemon auto-update are disabled for this release. A later NetBird upgrade is a coordinated application change that updates the server pin, bundled artifact, RPC adapter, compatibility tests, and release version together.

This is preferred over importing or copying NetBird networking code because it preserves upstream security ownership, minimizes privileged code in this project, and makes the network boundary auditable.

### Unprivileged Wails process with a privileged official service

The Wails executable runs as the interactive user. The official NetBird daemon runs as its Windows system service. UAC is requested only by a small installation or repair operation when service state requires it; creating, joining, connecting, disconnecting, viewing peers, and exporting diagnostics do not elevate the GUI.

Closing the window moves the application to the tray. Exiting the GUI does not stop the daemon or change tunnel state. Network state changes only through explicit Connect, Disconnect, Leave, or installer actions.

### Local RPC behind a project-owned adapter

The Go backend communicates with the official daemon using the v0.74.7 local RPC contract bound to the local machine. NetBird protobuf and RPC response types remain inside `internal/netbird`; application services consume normalized project DTOs.

The adapter surface covers version, service status, enrollment, connect, disconnect, deregistration, normalized status, and status events. If the official endpoint lacks a required event stream, the adapter performs bounded two-second polling while the main window is active. The frontend never connects to daemon RPC directly.

This is preferred over shelling out to CLI JSON because RPC provides typed state and avoids process and command-line secret exposure. CLI commands remain development diagnostics, not the product integration contract.

### One room maps to one managed profile

The client owns exactly one profile named `sogame-room`. It records the concrete profile ID returned by NetBird and never selects, deletes, or mutates unrelated profiles.

Create or Join is allowed only without a saved room. Switching performs Leave first: deregister the existing peer, remove the managed profile, clear protected room data, then enroll the new room. Disconnect only asks the daemon to go down and preserves peer identity. Reconnect uses that identity without retrieving the Setup Key again.

Room disablement remains an administrator-side operation. Leaving a room never disables it or affects other peers.

### Setup Key is transient; Room Code is encrypted

The Room API client runs in the Go backend. A returned Setup Key is passed directly to the adapter in memory and cleared on every success, failure, timeout, or cancellation path. It is excluded from Wails bindings, persistent state, logs, spans, errors, and diagnostics.

The Room Code is effectively a bearer secret because it can obtain the reusable Setup Key. Store it with DPAPI or Windows Credential Manager under the current Windows user. Store non-secret metadata in a small versioned JSON document written atomically. NetBird private keys and daemon credentials remain owned by NetBird storage.

### Two-layer state model

The application combines, but does not conflate, two sources:

```text
Room API: membership and peer metadata
NetBird RPC: local daemon, control-plane, and tunnel truth
```

The normalized state machine is:

```text
NoRoom
  -> Enrolling
  -> ControlPlaneConnected
       -> WaitingForPeer
       -> ConnectingPeer
            -> ConnectedP2P
            -> ConnectedRelay
  -> Reconnecting
  -> RecoverableError
```

`WaitingForPeer` is healthy because an empty room cannot form a tunnel. `ConnectedP2P` is preferred. `ConnectedRelay` is successful fallback. Management or Signal connectivity alone never produces a peer-connected label. NetBird alone selects P2P or Relay; the application does not force candidates or implement fallback logic.

The Room API is polled every five seconds while visible and every thirty seconds in the tray. HTTP 429 and transient failures use exponential backoff and mark cached data stale. Daemon RPC events trigger immediate local state updates.

### Small application boundary

Use these major modules:

```text
client/
  app/                   Wails lifecycle and event publication
  internal/roomapi/      Typed Room API client
  internal/netbird/      Official daemon RPC adapter
  internal/session/      Single-room state machine and commands
  internal/securestore/  DPAPI/Credential Manager implementation
  internal/platform/     Windows service and installer integration
  frontend/              React and TypeScript room UI
  build/                 Windows packaging metadata
```

Wails bindings expose command DTOs and sanitized state snapshots only. The initial UI contains Create, Join, Copy Room Code, Connect, Disconnect, Leave, peer list, local NetBird IP, path type, stale indicators, service repair, and local diagnostic export.

### Local-only diagnostics

Application logs are structured and centrally redacted. Diagnostic export requests anonymized daemon output where supported, applies application-level redaction again, and writes a local bundle. No diagnostic, telemetry, or crash data is uploaded automatically.

## Risks / Trade-offs

- **The NetBird local RPC contract is not a stable public SDK** -> Pin v0.74.7 exactly, isolate all RPC types in one adapter, and add contract tests against the bundled daemon.
- **Bundling a privileged network service increases release risk** -> Preserve the official signed artifact unchanged, verify digest and publisher, and keep elevation outside the Wails process.
- **A Room Code is a reusable bearer credential** -> Encrypt it per Windows user, omit it from diagnostics, and delete it on Leave.
- **A crash cannot guarantee immediate zeroing of Go memory** -> Minimize Setup Key lifetime, avoid copies and formatting, and never persist or send it to the frontend.
- **One room at a time limits advanced use cases** -> Keep the state model simple; multi-room support requires a future server and daemon architecture decision.
- **P2P cannot be guaranteed on every network** -> Treat official Relay as successful fallback and explain the active path without implementing custom traversal.
- **Room API polling can be rate-limited** -> Use conservative intervals, stale cached views, and exponential backoff.

## Migration Plan

1. Pin the deployed official NetBird server image to v0.74.7 and verify the existing control plane.
2. Record and verify the official Windows v0.74.7 distribution digest and publisher identity in release metadata.
3. Implement an RPC contract spike against a clean Windows VM and the bundled daemon before building UI workflows.
4. Build the adapter, secure store, Room API client, and state machine behind tests.
5. Add the Wails UI and Windows installer/repair flow.
6. Validate create, join, waiting, P2P, Relay, reconnect, leave, mismatch, and secret-redaction scenarios on Windows 10 and 11 x64.
7. Release without automatic update and retain the official CLI as an operator diagnostic tool.

Rollback uninstalls the Wails GUI. The installer offers an explicit choice to retain or remove the official NetBird service; it never disables server-side rooms. Server rollback restores the previously backed-up pinned Compose configuration.

## Open Questions

- Which exact v0.74.7 RPC definitions and service methods are required for profile creation, enrollment, deregistration, and event delivery on Windows?
- Does the official signed Windows distribution expose every required component for redistribution as one unchanged artifact, or must the application bundle and invoke the official MSI as a nested prerequisite?
