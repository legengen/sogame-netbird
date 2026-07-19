import type { CreateRoomRequest, JoinRoomRequest, RevealRoomCodeResult, StateSnapshot, SwitchRoomRequest } from './types'

declare global {
  interface Window {
    go?: {
      app?: {
        Controller?: {
          GetState: () => Promise<StateSnapshot>
          CreateRoom: (request: CreateRoomRequest) => Promise<StateSnapshot>
          JoinRoom: (request: JoinRoomRequest) => Promise<StateSnapshot>
          RevealRoomCode: () => Promise<RevealRoomCodeResult>
          ConnectRoom: () => Promise<StateSnapshot>
          DisconnectRoom: () => Promise<StateSnapshot>
          LeaveRoom: () => Promise<StateSnapshot>
          SwitchRoom: (request: SwitchRoomRequest) => Promise<StateSnapshot>
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

export async function revealRoomCode(): Promise<RevealRoomCodeResult> {
  const binding = window.go?.app?.Controller?.RevealRoomCode
  if (!binding) {
    return {
      error: {
        code: 'NETBIRD_SERVICE_UNAVAILABLE',
        message: '客户端后端尚未连接',
        retryable: true,
        action: '重新启动客户端后重试',
      },
    }
  }
  return binding()
}

export async function connectRoom(): Promise<StateSnapshot> {
  return invokeLifecycle('ConnectRoom', 'connect')
}

export async function disconnectRoom(): Promise<StateSnapshot> {
  return invokeLifecycle('DisconnectRoom', 'disconnect')
}

export async function leaveRoom(): Promise<StateSnapshot> {
  return invokeLifecycle('LeaveRoom', 'leave')
}

export async function switchRoom(request: SwitchRoomRequest): Promise<StateSnapshot> {
  const binding = window.go?.app?.Controller?.SwitchRoom
  if (!binding) {
    return unavailableState('switch')
  }
  return binding(request)
}

async function invokeLifecycle(command: 'ConnectRoom' | 'DisconnectRoom' | 'LeaveRoom', label: string): Promise<StateSnapshot> {
  const binding = window.go?.app?.Controller?.[command]
  if (!binding) {
    return unavailableState(label)
  }
  return binding()
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
