## Why

The self-hosted NetBird control plane and Room API are usable from the official CLI, but users do not yet have a focused Windows application for creating or joining a room, managing the single active room lifecycle, and understanding whether peer traffic is using P2P or Relay. A Wails desktop client can provide that experience while preserving the official NetBird client and daemon as the permanent networking foundation.

## What Changes

- Add a Windows 10/11 x64 desktop application built with Wails, Go, React, and TypeScript.
- Bundle the official signed NetBird v0.74.7 Windows build and manage its privileged daemon without elevating the desktop UI.
- Integrate with the official NetBird daemon through its local RPC interface; do not implement WireGuard, TUN, routing, Management, Signal, ICE, STUN, or Relay behavior in this project.
- Support creating or joining one saved room and connecting only that room at a time.
- Define explicit disconnect, reconnect, leave, and room-switch behavior around one managed NetBird profile.
- Distinguish control-plane readiness, waiting for another peer, peer connection establishment, P2P success, Relay success, and recoverable failure states.
- Protect room credentials and enrollment material using Windows-native secure storage and strict secret-redaction rules.
- Provide a small connection-focused UI, system tray behavior, peer visibility, and anonymized diagnostics.

## Capabilities

### New Capabilities

- `desktop-room-client`: Windows desktop room creation, joining, single-room persistence, lifecycle commands, peer presentation, tray behavior, and user-visible connection states.
- `netbird-daemon-integration`: Bundling, installation, version enforcement, local RPC control, status normalization, and recovery for the official NetBird v0.74.7 daemon.
- `client-secret-protection`: Secure handling of room codes, Setup Keys, local identity metadata, logs, diagnostics, and frontend/backend data boundaries.

### Modified Capabilities

None.

## Impact

- Adds a new Wails application and Windows packaging surface to the repository.
- Adds an official NetBird v0.74.7 binary/installer supply-chain dependency and a local RPC compatibility boundary.
- Consumes the existing public Room API without changing its current HTTP contract.
- Requires Windows service installation and repair flows with narrowly scoped UAC elevation.
- Requires the deployed NetBird server and packaged client daemon to remain pinned to v0.74.7 for this release.
