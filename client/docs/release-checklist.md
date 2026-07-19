# Sogame Client v0.1.1 Demo Release Checklist

## Version and source

- [x] Hotfix branch is based on the published `main` branch.
- [x] Frontend, package manifest, archive name, and Windows executable metadata use `0.1.1`.
- [x] NetBird client artifact and server image are pinned to `0.74.7`.
- [x] Official MSI digest and NetBird GmbH publisher signature are verified before packaging.
- [ ] OpenSpec change `build-wails-netbird-client` has all production-distribution gates complete and is archived.

## Automated verification

- [x] `go test ./...`
- [x] `go test -race ./app ./internal/diagnostics ./internal/observability`
- [x] `go vet ./...`
- [x] Frontend `npm test` and `npm run build`
- [x] `wails build -clean`
- [x] Official v0.74.7 daemon RPC contract probe
- [x] MSI signature, digest, and tamper checks
- [x] P2P, forced Relay, cross-room isolation, and daemon restart tests

## Release-candidate gates

- [ ] Run clean install, repair, upgrade, GUI-only uninstall, and optional daemon removal on Windows 10 x64.
- [ ] Repeat the install lifecycle matrix on Windows 11 x64.
- [ ] Verify physical sleep resume, network switching, and GUI restart behavior.
- [ ] Build the final package from a clean checkout and run `scripts/verify-release-package.ps1`.
- [x] Verify `sogame.exe` and `sogame-helper.exe` are explicitly marked unsigned for the demo channel.
- [ ] Sign `sogame.exe` and `sogame-helper.exe` before distribution to end users.
- [ ] Publish SHA-256 checksums for the ZIP and all shipped executable artifacts.
- [ ] Confirm no Setup Key, access token, Room Code, private key, or local environment file is present in the package.

## GitFlow completion

- [ ] Merge `hotfix/publish-demo-v0.1.1` into `main` with `--no-ff`.
- [ ] Tag the resulting `main` commit as `v0.1.1`.
- [ ] Merge `hotfix/publish-demo-v0.1.1` into `develop`.
- [ ] Delete the hotfix branch after both merges are complete.
