import type { CreateRoomRequest, JoinRoomRequest, StateSnapshot } from './types'

declare global {
  interface Window {
    go?: {
      app?: {
        Controller?: {
          GetState: () => Promise<StateSnapshot>
          CreateRoom: (request: CreateRoomRequest) => Promise<StateSnapshot>
          JoinRoom: (request: JoinRoomRequest) => Promise<StateSnapshot>
        }
      }
    }
  }
}

const initialState: StateSnapshot = {
  revision: 0,
  state: 'NoRoom',
  connectedPath: 'none',
  peers: [],
  peersStale: false,
  service: {
    installed: false,
    running: false,
    version: '',
    expectedVersion: '0.74.7',
    repairRequired: false,
  },
}

export async function getState(): Promise<StateSnapshot> {
  const binding = window.go?.app?.Controller?.GetState
  if (!binding) {
    return initialState
  }
  return binding()
}

export async function createRoom(request: CreateRoomRequest): Promise<StateSnapshot> {
  const binding = window.go?.app?.Controller?.CreateRoom
  if (!binding) {
    return unavailableState('create')
  }
  return binding(request)
}

export async function joinRoom(request: JoinRoomRequest): Promise<StateSnapshot> {
  const binding = window.go?.app?.Controller?.JoinRoom
  if (!binding) {
    return unavailableState('join')
  }
  return binding(request)
}

function unavailableState(command: string): StateSnapshot {
  return {
    ...initialState,
    state: 'RecoverableError',
    error: {
      code: 'NETBIRD_SERVICE_UNAVAILABLE',
      message: '客户端后端尚未连接',
      retryable: true,
      action: `重新启动客户端后重试 ${command === 'create' ? '创建' : '加入'}`,
    },
  }
}
