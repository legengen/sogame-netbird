# Diagnostics Privacy

Sogame diagnostics are local-only. `internal/diagnostics` writes bundles and crash reports below the current user's application-data directory and has no HTTP client, upload endpoint, telemetry queue, or background exporter.

Before a bundle or crash report is written, credentials, Room Codes, Setup Keys, private keys, IP addresses, hostnames, and peer identifiers are anonymized. Export is explicit through the Wails `ExportDiagnostics` command.
