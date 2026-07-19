# Release Evidence

## Reproducible local evidence

- `go test ./...`, `go test -race ./app ./internal/diagnostics ./internal/observability`, and `go vet ./...` pass on the development Windows host.
- Frontend `npm test` and `npm run build` pass.
- `wails build -clean` produces `build/bin/sogame.exe` for `windows/amd64`.
- `scripts/package-windows.ps1` verifies the official NetBird v0.74.7 MSI (size `37,974,016`, SHA-256 `1be9ce80767a728a8682bc3c114256b224b8d6657400ac031e458a05b5e5942d`) and produces a signed-ready staging directory plus ZIP.
- `scripts/verify-release-package.ps1` rechecks package contents, metadata, MSI size, digest, and signed-ready manifest.

## Environment-gated evidence

The following release gates remain pending because this workspace has no clean Windows 10/11 VM and no second public client. The active host has an installed NetBird v0.74.6 service; replacing it with v0.74.7 would disrupt the user's current network and is intentionally not attempted.

- Clean v0.74.7 RPC contract probe and bundled-daemon contract test.
- Clean install, repair, upgrade, GUI-only uninstall, and optional daemon removal.
- Room create/join with the managed profile against the self-hosted control plane.
- Same-room P2P and Relay fallback with direct connectivity blocked.
- Sleep/network-switch/daemon-restart end-to-end workflows.
- Cross-room isolation with multiple simultaneous room peers.

These are explicit environment gates, not simulated local test results.
