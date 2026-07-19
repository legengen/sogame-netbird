import { describe, expect, it } from 'vitest'
import { isValidRoomCode, normalizeRoomCode } from './roomCode'

describe('room code input', () => {
  it('normalizes pasted lowercase and separators', () => {
    expect(normalizeRoomCode('7x4k 329b_yy95')).toBe('7X4K-329B-YY95')
  })

  it('rejects incomplete or malformed codes', () => {
    expect(isValidRoomCode('7X4K-329B-YY95')).toBe(true)
    expect(isValidRoomCode('7X4K-329B')).toBe(false)
    expect(isValidRoomCode('7X4K-329B-YY9!')).toBe(false)
  })
})
