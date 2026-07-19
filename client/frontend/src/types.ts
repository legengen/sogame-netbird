export type ConnectionState =
  | 'NoRoom'
  | 'Enrolling'
  | 'ControlPlaneConnected'
  | 'WaitingForPeer'
  | 'ConnectingPeer'
  | 'ConnectedP2P'
  | 'ConnectedRelay'
  | 'Reconnecting'
  | 'RecoverableError'

export type PathType = 'none' | 'p2p' | 'relay'

export interface PublicError {
  code: string
  message: string
  retryable: boolean
  action?: string
}

export interface PeerSnapshot {
  id: string
  name: string
  netbirdIp: string
  connected: boolean
  path: PathType
}

export interface StateSnapshot {
  revision: number
  state: ConnectionState
  roomId?: string
  roomCodeMasked?: string
  managementUrl?: string
  localNetbirdIp?: string
  profileId?: string
  connectedPath: PathType
  peers: PeerSnapshot[]
  peersStale: boolean
  lastPeerRefreshAt?: string
  service: {
    installed: boolean
    running: boolean
    version: string
    expectedVersion: string
    repairRequired: boolean
  }
  error?: PublicError
  busyCommand?: string
}
