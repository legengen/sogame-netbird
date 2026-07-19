## ADDED Requirements

### Requirement: Windows desktop scope
The system SHALL provide a Wails desktop application for 64-bit Windows 10 and Windows 11, using a Go backend and a React with TypeScript frontend.

#### Scenario: Supported Windows launch
- **WHEN** a user launches the installed application on a supported 64-bit Windows system
- **THEN** the application starts without requiring the GUI process to run as an administrator

#### Scenario: Unsupported platform build
- **WHEN** the project release pipeline builds this change
- **THEN** it produces only the documented Windows x64 client artifact

### Requirement: Room creation and joining
The system SHALL let a user create a room through `POST /rooms` or join an active room through `POST /rooms/join`, then enroll the managed NetBird profile with the returned Management URL and Setup Key.

#### Scenario: Create a room
- **WHEN** a user chooses to create a room and the Room API succeeds
- **THEN** the client saves the returned room as its only room and starts NetBird enrollment

#### Scenario: Join a room
- **WHEN** a user submits a valid room code and the Room API succeeds
- **THEN** the client saves that room as its only room and starts NetBird enrollment

#### Scenario: Room API rejection
- **WHEN** room creation or joining returns a validation, unavailable, or rate-limit error
- **THEN** the client presents a non-secret error state and does not create a partially enrolled local room

### Requirement: Single saved and active room
The system SHALL store at most one room and SHALL operate at most one managed NetBird profile at a time.

#### Scenario: Join with no saved room
- **WHEN** the user creates or joins a room while no room is saved
- **THEN** the client creates and selects the single managed profile named `sogame-room`

#### Scenario: Attempt to switch rooms
- **WHEN** the user attempts to create or join another room while a room is saved
- **THEN** the client requires the user to leave the current room before enrolling in the new room

#### Scenario: No room history
- **WHEN** the user leaves the current room
- **THEN** the client removes the saved room rather than retaining a selectable room history

### Requirement: Explicit room lifecycle
The system SHALL distinguish disconnecting, reconnecting, leaving, switching, closing the window, exiting the GUI, and uninstalling.

#### Scenario: Disconnect
- **WHEN** the user disconnects
- **THEN** the client asks the official daemon to go down while retaining the room and peer identity for later reconnection

#### Scenario: Reconnect
- **WHEN** a disconnected user reconnects
- **THEN** the client reuses the existing NetBird profile identity without requesting or persisting the Setup Key again

#### Scenario: Leave room
- **WHEN** the user confirms leaving the room
- **THEN** the client deregisters the peer, removes the managed profile, and clears all saved room data without disabling the server-side room

#### Scenario: Switch room
- **WHEN** the user confirms switching to another room
- **THEN** the client completes the leave workflow before beginning the new room workflow

#### Scenario: Close window
- **WHEN** the user closes the main window
- **THEN** the application remains available in the system tray and leaves the daemon connection unchanged

#### Scenario: Exit GUI
- **WHEN** the user exits the desktop GUI from the tray
- **THEN** the GUI terminates without implicitly disconnecting or deregistering the NetBird peer

### Requirement: Layered connection states
The system SHALL present control-plane readiness separately from peer tunnel connectivity and SHALL call a peer connection successful only when the official daemon reports P2P or Relay connectivity.

#### Scenario: Control plane ready without another peer
- **WHEN** Management and Signal are connected but the room has no other peer
- **THEN** the client displays `WaitingForPeer` as a normal non-error state

#### Scenario: Peer discovered without tunnel
- **WHEN** another room peer exists but the daemon has not established a tunnel
- **THEN** the client displays `ConnectingPeer` rather than `Connected`

#### Scenario: P2P tunnel established
- **WHEN** the daemon reports a connected peer with connection type P2P
- **THEN** the client displays `ConnectedP2P` as the preferred success state

#### Scenario: Relay tunnel established
- **WHEN** the daemon reports a connected peer with connection type Relay
- **THEN** the client displays `ConnectedRelay` as a successful fallback state

#### Scenario: Control plane only is not tunnel success
- **WHEN** Management and Signal are connected but no peer is connected by P2P or Relay
- **THEN** the client does not display the room as peer-connected

### Requirement: Peer visibility and refresh
The system SHALL show peer membership from the Room API and local tunnel details from the daemon without treating either source as authoritative for the other's state.

#### Scenario: Foreground refresh
- **WHEN** the main window is visible
- **THEN** the client refreshes Room API peer membership no more frequently than every five seconds and updates daemon state from RPC events or bounded polling

#### Scenario: Background refresh
- **WHEN** the application is only in the system tray
- **THEN** the client reduces Room API peer polling to no more frequently than every thirty seconds

#### Scenario: Rate limited peer query
- **WHEN** the Room API returns HTTP 429
- **THEN** the client applies exponential backoff and continues showing the last known peer list as stale

### Requirement: Minimal room interface
The system SHALL provide room code, local NetBird IP, peer list, P2P or Relay state, connection commands, and anonymized diagnostic export without exposing administrative room controls.

#### Scenario: Connected room display
- **WHEN** a room is active
- **THEN** the main interface shows the room code, local NetBird IP, current peers, and each available connection type

#### Scenario: Administrative action omitted
- **WHEN** a normal user operates the desktop client
- **THEN** the client provides no room disable endpoint or Room API administrator token input

