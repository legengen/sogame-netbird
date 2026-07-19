import type { StateSnapshot } from './types'

declare global {
  interface Window {
    go?: {
      app?: {
        Controller?: {
          GetState: () => Promise<StateSnapshot>
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
