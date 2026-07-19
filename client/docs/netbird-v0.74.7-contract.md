# NetBird v0.74.7 Windows contract

This application redistributes the unchanged official Windows x64 MSI as an
installer prerequisite. The MSI itself is not stored in Git. The release build
downloads it from the URL in `client/build/netbird-release.json` and verifies it
before packaging or execution.

## Official distribution

| Field | Confirmed value |
| --- | --- |
| Upstream tag | `v0.74.7` |
| Upstream commit | `a1c9427d8004576e2cbb9e546d409847fa9df318` |
| Artifact | `netbird_installer_0.74.7_windows_amd64.msi` |
| Size | `37,974,016` bytes |
| SHA-256 | `1be9ce80767a728a8682bc3c114256b224b8d6657400ac031e458a05b5e5942d` |
| Authenticode status | Valid |
| Publisher | `CN=NetBird GmbH, O=NetBird GmbH` |
| Release certificate thumbprint | `7B41FCCAFCB794720FE07D381F9CBDF18AB5900F` |
| Certificate issuer | `GlobalSign GCC R45 EV CodeSigning CA 2020` |

The digest above is published by the official GitHub Release API and was also
confirmed against the downloaded MSI with `Get-FileHash -Algorithm SHA256`.
The publisher identity and valid signature were confirmed with
`Get-AuthenticodeSignature`. The thumbprint is recorded as release evidence;
runtime verification trusts a valid Windows signature and the expected subject
identity so a legitimate NetBird certificate renewal does not break repairs.

The MSI declares product `NetBird` version `0.74.7`, manufacturer `NetBird GmbH`,
and installs these files:

- `netbird.exe` version 0.74.7.0
- `netbird-ui.exe` version 0.74.7.0
- `wintun.dll` version 0.14.1.0
- `netbird.png`
- `opengl32.dll` on AMD64

It creates the automatic LocalSystem service `NetBird`. The application sets
`AUTOSTART=0` so the official tray UI does not compete with the Sogame UI. The
silent elevated install command is:

```text
msiexec.exe /i <artifact.msi> /quiet /qn /norestart /l*v <install.log> AUTOSTART=0
```

## Redistribution and license

NetBird client code and binaries are covered by the upstream BSD 3-Clause
license. Binary redistribution must reproduce that license and disclaimer in
the installer documentation. The source for the exact redistributed client is
the official `netbirdio/netbird` repository at the tag and commit recorded
above. No NetBird executable, library, protocol implementation, or generated
binary is modified by this project.

## Coordinated upgrades

The Compose server image, this metadata, vendored RPC bindings, adapter contract
tests, and desktop release version form one compatibility unit. Upgrade all of
them together; never substitute `latest` or allow the official daemon to update
independently.

## Local daemon RPC

The Windows service listens on `tcp://127.0.0.1:41731`. Official CLI code removes
the `tcp://` prefix and opens an insecure gRPC connection to
`127.0.0.1:41731` with a ten-second dial timeout. The service listener is bound
to loopback, so the lack of TLS does not expose the control API remotely.

All calls use `daemon.DaemonService` from upstream
`client/proto/daemon.proto` at tag `v0.74.7`:

| Project operation | RPC and required fields | Confirmed behavior |
| --- | --- | --- |
| Health and version | `Status(StatusRequest)` | Reachability proves the service RPC is ready. `StatusResponse.daemonVersion` must equal `0.74.7`. |
| Full status | `Status({getFullPeerStatus:true})` | Returns daemon state, Management and Signal state, local IP, peers, Relay state, and event history. |
| List profiles | `ListProfiles({username})` | Returns exact profile ID, display name, and active flag. |
| Active profile | `GetActiveProfile({})` | Returns exact active ID, display name, and Windows username. |
| Create profile | `AddProfile({username,profileName:"sogame-room"})` | Creates a random on-disk ID and returns it. Duplicate display names are possible, so the returned ID must be persisted and used thereafter. |
| Select profile | `SwitchProfile({profileName:<id>,username})` | Resolves the handle in the daemon, persists daemon active state, loads its config, and returns the resolved ID. |
| Enroll | `Login({setupKey,managementUrl,profileName:<id>,username})` | Selects the managed profile, persists its Management URL, authenticates the Setup Key, and stores identity in NetBird-owned state. |
| Connect | `Up({profileName:<id>,username})` | Selects and starts the managed profile. Reuses existing identity after disconnect. |
| Disconnect | `Down({})` | Stops only the currently active engine and preserves identity. Because this request has no profile field, the adapter must verify/select the managed ID first. |
| Deregister | `Logout({profileName:<id>,username})` | Deregisters that profile peer and changes an active profile to `NeedsLogin`. |
| Remove profile | `RemoveProfile({profileName:<id>,username})` | Resolves the exact managed ID, logs it out defensively, then removes only its config/state. The active profile must be changed before removal. |
| Events | `SubscribeEvents({})` | Streams system events until cancellation. It is not a complete connection-state stream, so normalized status also uses bounded polling. |

The daemon-level states are `Idle`, `Connecting`, `Connected`, `NeedsLogin`,
`LoginFailed`, and `SessionExpired`. Peer tunnel truth comes from each
`FullStatus.peers` entry: `connStatus == "Connected"` plus `relayed == false`
means P2P, while the same status plus `relayed == true` means Relay. Management
or Signal connectivity alone is never a peer tunnel success.

`SubscribeEvents` metadata and messages are untrusted observability input. The
adapter must normalize or redact them before publishing application events.

## Permissions boundary

Normal RPC calls are made by the unprivileged interactive user. MSI install,
service repair, and optional service removal require elevation. The GUI process
itself is never relaunched elevated.

## Probe evidence

The repository needs a final clean-VM contract run before release. A provisional
read-only probe was run as a non-administrator against an already-installed
Windows daemon v0.74.6 on 2026-07-19. It confirmed that the ordinary user can
dial the daemon and call full `Status`; the response contained daemon version,
Management/Signal connectivity, local NetBird address, peer tunnel state, Relay
availability, and system events. It also demonstrated the intended exact-version
gate because the response reported `0.74.6`, not the required `0.74.7`.

The same restricted session could not query `Get-NetTCPConnection` or WMI service
details. Service discovery must therefore classify access-denied separately and
use RPC health plus executable version where possible. This provisional run does
not satisfy the clean Windows v0.74.7 VM release gate.

## Packaging decision

Use the official MSI as an unchanged nested prerequisite. Do not install the
signed tar archive by hand: the MSI is the upstream-owned unit that registers
the LocalSystem service, Wintun dependency, PATH entry, upgrade code, service
start/stop/removal actions, and uninstall behavior. The Sogame installer verifies
the MSI digest and Authenticode publisher, invokes it with the documented silent
arguments, and retains the original MSI product lifecycle.
