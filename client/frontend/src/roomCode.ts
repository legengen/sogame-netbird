const roomCodePattern = /^[A-Z0-9]{4}(?:-[A-Z0-9]{4}){2}$/

export function normalizeRoomCode(value: string): string {
  const compact = value.toUpperCase().replace(/[^A-Z0-9]/g, '').slice(0, 12)
  return compact.match(/.{1,4}/g)?.join('-') ?? ''
}

export function isValidRoomCode(value: string): boolean {
  return roomCodePattern.test(value)
}
