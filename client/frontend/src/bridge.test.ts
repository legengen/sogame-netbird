import { afterEach, describe, expect, it, vi } from 'vitest'
import { createRoom, joinRoom, repairService, revealRoomCode } from './bridge'
import type { StateSnapshot } from './types'

const connected: StateSnapshot = {
  revision: 1,
  state: 'WaitingForPeer',
  connectedPath: 'none',
  peers: [],
  peersStale: false,
  service: {
    installed: true,
    running: true,
    version: '0.74.7',
    expectedVersion: '0.74.7',
    repairRequired: false,
  },
}

afterEach(() => vi.unstubAllGlobals())

describe('Wails room workflow bridge', () => {
  it('sends create and join command DTOs to the backend', async () => {
    const create = vi.fn().mockResolvedValue(connected)
    const join = vi.fn().mockResolvedValue(connected)
    const reveal = vi.fn().mockResolvedValue({ roomCode: 'AAAA-BBBB-CCCC' })
    vi.stubGlobal('window', { go: { app: { Controller: { CreateRoom: create, JoinRoom: join, RevealRoomCode: reveal } } } })

    await expect(createRoom({ displayName: '' })).resolves.toEqual(connected)
    await expect(joinRoom({ roomCode: '7X4K-329B-YY95', displayName: '' })).resolves.toEqual(connected)
    expect(create).toHaveBeenCalledWith({ displayName: '' })
    expect(join).toHaveBeenCalledWith({ roomCode: '7X4K-329B-YY95', displayName: '' })
    await expect(revealRoomCode()).resolves.toEqual({ roomCode: 'AAAA-BBBB-CCCC' })
    expect(reveal).toHaveBeenCalledOnce()
  })

  it('returns a recoverable state when Wails bindings are unavailable', async () => {
    vi.stubGlobal('window', {})
    const state = await createRoom({ displayName: '' })
    expect(state.state).toBe('RecoverableError')
    expect(state.error?.code).toBe('NETBIRD_SERVICE_UNAVAILABLE')
    expect(state.busyCommand).toBeUndefined()
  })

  it('routes lifecycle commands through the Wails controller', async () => {
    const state = vi.fn().mockResolvedValue(connected)
    const switchRoom = vi.fn().mockResolvedValue(connected)
    vi.stubGlobal('window', { go: { app: { Controller: {
      ConnectRoom: state,
      DisconnectRoom: state,
      LeaveRoom: state,
      SwitchRoom: switchRoom,
    } } } })

    const { connectRoom, disconnectRoom, leaveRoom, switchRoom: invokeSwitch } = await import('./bridge')
    await connectRoom()
    await disconnectRoom()
    await leaveRoom()
    await invokeSwitch({ mode: 'join', roomCode: '7X4K-329B-YY95', displayName: '', confirmed: true })
    expect(state).toHaveBeenCalledTimes(3)
    expect(switchRoom).toHaveBeenCalledWith({ mode: 'join', roomCode: '7X4K-329B-YY95', displayName: '', confirmed: true })
  })

  it('routes service repair through the Wails controller', async () => {
    const repair = vi.fn().mockResolvedValue(connected)
    vi.stubGlobal('window', { go: { app: { Controller: { RepairService: repair } } } })
    await expect(repairService()).resolves.toEqual(connected)
    expect(repair).toHaveBeenCalledOnce()
  })
})
