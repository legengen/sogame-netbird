# Release Evidence

## Reproducible local evidence

- `go test ./...`, `go test -race ./app ./internal/diagnostics ./internal/observability`, and `go vet ./...` pass on the development Windows host.
- Frontend `npm test` and `npm run build` pass.
- `wails build -clean` produces `build/bin/sogame.exe` for `windows/amd64`.
- `scripts/package-windows.ps1` verifies the official NetBird v0.74.7 MSI (size `37,974,016`, SHA-256 `1be9ce80767a728a8682bc3c114256b224b8d6657400ac031e458a05b5e5942d`) and produces a signed-ready staging directory plus ZIP.
- `scripts/verify-release-package.ps1` rechecks package contents, metadata, MSI size, digest, and signed-ready manifest.
- The installed official v0.74.7 daemon passed `TestOfficialV0747DaemonReadOnlyContract`, the official MSI passed signature and tamper verification, and read-only Windows service discovery passed.
- Two clients connected to the self-hosted control plane, exchanged traffic with 0% packet loss, and both reported `Peers count: 1/1 Connected` with `Connection type: P2P`.
- With direct peer UDP temporarily blocked on one client, the official daemon selected `rels://legengen.top:443`, reported `Connection type: Relayed`, and carried four successful ICMP replies with 0% packet loss. Removing the temporary rules restored P2P and 0% packet loss.
- A fresh Room API create/join flow was exercised through the self-hosted control plane: the peer endpoint reported exactly two connected peers, both initially observed as idle before traffic established the P2P path.
- Moving one client into a second room produced one peer in each room and blocked cross-room traffic. Returning it to the original room restored two peers and P2P. The desktop command surface has no room-disable operation, and the leave test rejects any administrative disable call.

## Environment-gated evidence

The following release gates remain pending because this workspace has no clean Windows 10/11 VM and cannot automate physical sleep or network switching.

- Clean Windows VM RPC probe and install lifecycle matrix.
- Clean install, repair, upgrade, GUI-only uninstall, and optional daemon removal.
- Managed `sogame-room` profile isolation against the self-hosted control plane (Room API create/join itself is verified above).
- Sleep/network-switch/daemon-restart end-to-end workflows.

These are explicit environment gates, not simulated local test results.
