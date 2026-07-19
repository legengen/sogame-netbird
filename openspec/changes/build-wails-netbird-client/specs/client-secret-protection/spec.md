## ADDED Requirements

### Requirement: Setup Key is enrollment-only memory
The system MUST keep a Room API Setup Key only in Go backend memory for the enrollment operation and MUST NOT persist it, log it, expose it through frontend state, or include it in diagnostics.

#### Scenario: Successful enrollment
- **WHEN** the daemon confirms enrollment
- **THEN** the client clears its Setup Key reference and relies on the daemon's peer identity for future reconnects

#### Scenario: Enrollment failure
- **WHEN** enrollment fails or is cancelled
- **THEN** the client clears the Setup Key reference before returning an error

#### Scenario: Frontend inspection
- **WHEN** the frontend receives room or connection DTOs
- **THEN** no DTO contains the Setup Key or an equivalent enrollment credential

### Requirement: Room Code is a protected bearer secret
The system SHALL treat the saved Room Code as a long-lived bearer secret and SHALL protect it with Windows-native user-bound encryption at rest.

#### Scenario: Save active room
- **WHEN** room enrollment succeeds
- **THEN** the client stores the Room Code using Windows Credential Manager or DPAPI rather than plaintext application configuration

#### Scenario: Different Windows user
- **WHEN** another Windows account reads the application data directory
- **THEN** it cannot recover the saved Room Code from plaintext files

#### Scenario: Leave room
- **WHEN** the user completes the leave workflow
- **THEN** the client deletes the protected Room Code and associated local room metadata

### Requirement: Minimal local metadata
The system SHALL persist only the room ID, Management URL, managed profile ID, display name, last non-secret state, and protected Room Code needed to restore the single room.

#### Scenario: Application restart
- **WHEN** the client restarts with valid saved metadata
- **THEN** it restores the single room without requesting a Setup Key or storing NetBird private keys itself

#### Scenario: Partial metadata
- **WHEN** required metadata or protected Room Code cannot be read
- **THEN** the client enters a repair or leave flow without printing recovered fragments

### Requirement: Secret-redacted observability
The system MUST exclude Setup Keys, Room API administrator tokens, NetBird private keys, bearer authorization values, and plaintext Room Codes from application logs, RPC traces, crash reports, UI errors, and exported diagnostics.

#### Scenario: API failure
- **WHEN** a Room API or daemon operation fails
- **THEN** logs contain only a typed error, endpoint class, status code, and correlation metadata that cannot be used to join the room

#### Scenario: Diagnostic export
- **WHEN** a user exports diagnostics
- **THEN** the export anonymizes IP addresses, hostnames, peer identifiers, and nonessential network metadata in addition to removing secrets

#### Scenario: Unexpected error value
- **WHEN** an upstream error contains request or credential material
- **THEN** the application applies centralized redaction before writing or displaying it

### Requirement: No automatic diagnostic upload
The system SHALL create diagnostic bundles locally and SHALL NOT upload telemetry, crash reports, or NetBird debug bundles without a separate explicit user-approved operation.

#### Scenario: Export diagnostics
- **WHEN** the user requests diagnostic export
- **THEN** the client writes a local anonymized bundle and does not contact an upload endpoint

#### Scenario: Application crash
- **WHEN** the application records a local crash report
- **THEN** it remains local and follows the same secret-redaction rules

