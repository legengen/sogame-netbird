## 1. NetBird v0.74.7 Contract Spikes

- [x] 1.1 Pin the Compose NetBird server image to v0.74.7 and document the coordinated client/server version rule
- [x] 1.2 Identify the official signed Windows x64 v0.74.7 distribution artifact, publisher identity, digest, redistribution contents, and installation command
- [x] 1.3 Map the v0.74.7 Windows local RPC definitions and methods for version, service status, profile management, enrollment, connect, disconnect, deregistration, status, and events
- [ ] 1.4 Build a throwaway RPC probe against a clean Windows VM and record confirmed endpoint, transport, permissions, and method behavior
- [x] 1.5 Decide from the spike whether packaging invokes the official MSI prerequisite or installs the unchanged signed binary distribution, and update release metadata accordingly

## 2. Wails Application Foundation

- [x] 2.1 Scaffold the Windows x64 Wails application with a Go backend and React/TypeScript frontend under `client/`
- [x] 2.2 Add the application, internal module, frontend, build, and test directory boundaries from the design
- [x] 2.3 Define sanitized Wails command DTOs, state snapshots, typed errors, and backend event names
- [x] 2.4 Add Windows-only build constraints and fail unsupported platform builds with a clear message
- [x] 2.5 Add structured local logging with centralized secret and identifier redaction

## 3. Official NetBird Distribution and Service

- [x] 3.1 Add pinned v0.74.7 artifact metadata and build-time digest verification without committing the binary to source control unless explicitly approved
- [x] 3.2 Implement Windows publisher-signature and digest verification before any install or repair action
- [x] 3.3 Implement unprivileged service discovery, version inspection, and health classification
- [x] 3.4 Implement the narrowly scoped elevated install and repair helper without elevating the Wails GUI
- [x] 3.5 Implement explicit service removal support for the uninstaller while preserving the user's choice to retain the official daemon
- [ ] 3.6 Add tests for valid, missing, mismatched, unsigned, and tampered NetBird artifacts and services

## 4. Local RPC Adapter

- [x] 4.1 Generate or vendor only the required official v0.74.7 RPC bindings with license and source attribution
- [x] 4.2 Implement the project-owned NetBird adapter interface and normalized DTO mapping
- [x] 4.3 Implement exact v0.74.7 daemon version enforcement and typed mismatch repair results
- [x] 4.4 Implement managed `sogame-room` profile creation, selection, validation, and removal without changing unrelated profiles
- [x] 4.5 Implement enrollment, connect, disconnect, deregistration, status, and local event delivery over RPC
- [x] 4.6 Implement bounded RPC reconnection and polling fallback for daemon restart, sleep resume, and network change
- [ ] 4.7 Add adapter contract tests against the bundled v0.74.7 daemon plus unit tests with a fake adapter

## 5. Room API and Secure Storage

- [x] 5.1 Implement typed Room API clients for create, join, and peer listing with request limits, timeouts, and typed HTTP errors
- [x] 5.2 Generate a unique idempotency key for each create intent and safely reuse it only for retries of that intent
- [x] 5.3 Implement Room API retry and exponential-backoff behavior for transient failures and HTTP 429
- [x] 5.4 Implement versioned atomic storage for non-secret single-room metadata
- [x] 5.5 Implement Windows user-bound Room Code protection with DPAPI or Credential Manager
- [x] 5.6 Implement enrollment-scoped Setup Key handling that never enters frontend DTOs, persistence, logs, errors, or diagnostics
- [x] 5.7 Add tests proving Setup Key and plaintext Room Code absence from files, logs, Wails events, errors, and diagnostic fixtures

## 6. Single-Room Session State Machine

- [x] 6.1 Implement the normalized NoRoom, Enrolling, ControlPlaneConnected, WaitingForPeer, ConnectingPeer, ConnectedP2P, ConnectedRelay, Reconnecting, and RecoverableError states
- [ ] 6.2 Implement Create and Join transactions with compensation for partial local enrollment
- [ ] 6.3 Implement Disconnect and Reconnect while preserving the existing daemon peer identity
- [ ] 6.4 Implement Leave by deregistering the peer, removing only the managed profile, and clearing protected local room data
- [ ] 6.5 Implement Switch as a confirmed complete Leave followed by a new Create or Join flow
- [ ] 6.6 Implement five-second foreground and thirty-second tray Room API refresh with stale data markers
- [ ] 6.7 Implement P2P-preferred and Relay-success presentation using daemon-reported path selection only
- [ ] 6.8 Add deterministic state-machine tests for empty rooms, peer appearance, P2P, Relay, outages, reconnect, leave, and conflicting stored state

## 7. Wails User Experience

- [ ] 7.1 Build the no-room view with Create Room and Join Room workflows
- [ ] 7.2 Build the active-room view with protected room-code reveal/copy, local NetBird IP, peer list, stale state, and P2P or Relay status
- [ ] 7.3 Add Connect, Disconnect, Leave, and confirmed Switch interactions with stable disabled and progress states
- [ ] 7.4 Add service missing, service repair, version mismatch, Room API error, and daemon recovery views
- [ ] 7.5 Add system tray behavior so closing the window retains connectivity and exiting the GUI does not mutate daemon state
- [ ] 7.6 Verify all text fits and controls remain usable across supported Windows scaling settings and compact desktop sizes

## 8. Diagnostics and Privacy

- [ ] 8.1 Implement local anonymized application and NetBird diagnostic bundle generation
- [ ] 8.2 Redact IP addresses, hostnames, peer identifiers, Room Codes, Setup Keys, authorization values, and private key material from exports
- [ ] 8.3 Ensure crash reports and logs remain local and add no automatic telemetry or upload path
- [ ] 8.4 Add adversarial tests with secrets embedded in upstream errors, RPC messages, and log fields

## 9. Packaging and End-to-End Verification

- [ ] 9.1 Build the signed-ready Windows x64 installer with Wails assets and the verified official NetBird prerequisite
- [ ] 9.2 Test clean install, service repair, application upgrade, GUI-only uninstall, and optional daemon removal on Windows 10 and 11 x64
- [ ] 9.3 Verify room creation, room joining, managed profile isolation, and WaitingForPeer behavior against the self-hosted control plane
- [ ] 9.4 Verify same-room P2P connectivity with UDP 3478 available and Relay fallback with direct connectivity blocked
- [ ] 9.5 Verify sleep resume, network switching, daemon restart, GUI restart, disconnect, reconnect, leave, and switch workflows
- [ ] 9.6 Verify cross-room isolation and confirm the client never offers server-side room disablement
- [ ] 9.7 Run Go, frontend, Wails, RPC contract, secret-leak, installer, and end-to-end test suites and document the release evidence
