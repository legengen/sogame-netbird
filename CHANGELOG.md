# Changelog

## [0.1.1] - 2026-07-19

### Fixed

- Corrected Windows system-tray initialization and managed NetBird profile validation.
- Added reproducible GitHub CI and tag-driven Windows packaging.

### Distribution

- Published as an unsigned Windows demo; SmartScreen warnings are expected.
- The bundled official NetBird v0.74.7 MSI remains unchanged and signed by NetBird GmbH.

## [0.1.0] - 2026-07-19

### Added

- Windows 10/11 x64 desktop client built with Wails, Go, React, and TypeScript.
- Single-room create, join, connect, disconnect, leave, and switch workflows.
- Official NetBird v0.74.7 daemon integration through local RPC with exact-version enforcement.
- P2P-preferred connection state with successful Relay fallback presentation.
- Windows DPAPI protection for Room Codes and enrollment-only Setup Key handling.
- Local-only anonymized diagnostics, service repair, and system tray behavior.
- Signed-ready Windows package containing the verified official NetBird v0.74.7 MSI prerequisite.

### Verified

- Same-room P2P and forced Relay data paths against the self-hosted control plane.
- Cross-room policy isolation, daemon restart recovery, and managed `sogame-room` profile isolation.
- Go, race, frontend, Wails, RPC contract, artifact integrity, and secret-redaction suites.

### Release Gates

- Clean Windows 10 and Windows 11 install, repair, upgrade, and uninstall matrix.
- Physical sleep, network-switch, and GUI-restart end-to-end verification.
- Final code signing, package checksum publication, and release-candidate smoke test.
