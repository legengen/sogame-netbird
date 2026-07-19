package app

import (
	"encoding/json"

	"github.com/legengen/sogame-netbird/client/internal/observability"
)

type ConnectionState string

const (
	StateNoRoom                ConnectionState = "NoRoom"
	StateEnrolling             ConnectionState = "Enrolling"
	StateControlPlaneConnected ConnectionState = "ControlPlaneConnected"
	StateWaitingForPeer        ConnectionState = "WaitingForPeer"
	StateConnectingPeer        ConnectionState = "ConnectingPeer"
	StateConnectedP2P          ConnectionState = "ConnectedP2P"
	StateConnectedRelay        ConnectionState = "ConnectedRelay"
	StateReconnecting          ConnectionState = "Reconnecting"
	StateRecoverableError      ConnectionState = "RecoverableError"
)

type PathType string

const (
	PathNone  PathType = "none"
	PathP2P   PathType = "p2p"
	PathRelay PathType = "relay"
)

type PeerSnapshot struct {
	ID        string   `json:"id"`
	Name      string   `json:"name"`
	NetBirdIP string   `json:"netbirdIp"`
	Connected bool     `json:"connected"`
	Path      PathType `json:"path"`
}

type ServiceSnapshot struct {
	Installed       bool   `json:"installed"`
	Running         bool   `json:"running"`
	Version         string `json:"version"`
	ExpectedVersion string `json:"expectedVersion"`
	RepairRequired  bool   `json:"repairRequired"`
}

type StateSnapshot struct {
	Revision          uint64          `json:"revision"`
	State             ConnectionState `json:"state"`
	RoomID            string          `json:"roomId,omitempty"`
	RoomCodeMasked    string          `json:"roomCodeMasked,omitempty"`
	ManagementURL     string          `json:"managementUrl,omitempty"`
	LocalNetBirdIP    string          `json:"localNetbirdIp,omitempty"`
	ProfileID         string          `json:"profileId,omitempty"`
	ConnectedPath     PathType        `json:"connectedPath"`
	Peers             []PeerSnapshot  `json:"peers"`
	PeersStale        bool            `json:"peersStale"`
	LastPeerRefreshAt string          `json:"lastPeerRefreshAt,omitempty"`
	Service           ServiceSnapshot `json:"service"`
	Error             *PublicError    `json:"error,omitempty"`
	BusyCommand       string          `json:"busyCommand,omitempty"`
}

func (s StateSnapshot) MarshalJSON() ([]byte, error) {
	type stateSnapshotJSON StateSnapshot
	clean := s
	clean.RoomID = observability.Redact(clean.RoomID)
	clean.RoomCodeMasked = observability.Redact(clean.RoomCodeMasked)
	clean.ManagementURL = observability.Redact(clean.ManagementURL)
	clean.LocalNetBirdIP = observability.Redact(clean.LocalNetBirdIP)
	clean.ProfileID = observability.Redact(clean.ProfileID)
	clean.LastPeerRefreshAt = observability.Redact(clean.LastPeerRefreshAt)
	clean.BusyCommand = observability.Redact(clean.BusyCommand)
	clean.Peers = append([]PeerSnapshot(nil), s.Peers...)
	for index := range clean.Peers {
		clean.Peers[index].ID = observability.Redact(clean.Peers[index].ID)
		clean.Peers[index].Name = observability.Redact(clean.Peers[index].Name)
		clean.Peers[index].NetBirdIP = observability.Redact(clean.Peers[index].NetBirdIP)
	}
	return json.Marshal(stateSnapshotJSON(clean))
}

type CreateRoomRequest struct {
	DisplayName string `json:"displayName"`
}

type JoinRoomRequest struct {
	RoomCode    string `json:"roomCode"`
	DisplayName string `json:"displayName"`
}

type SwitchRoomRequest struct {
	Mode        string `json:"mode"`
	RoomCode    string `json:"roomCode,omitempty"`
	DisplayName string `json:"displayName"`
}

type DiagnosticResult struct {
	Path string `json:"path"`
}

type RevealRoomCodeResult struct {
	RoomCode string       `json:"roomCode,omitempty"`
	Error    *PublicError `json:"error,omitempty"`
}
