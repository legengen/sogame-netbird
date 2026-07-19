import { useEffect, useState } from 'react'
import {
	Check,
	CircleHelp,
	Copy,
	Eye,
	Gamepad2,
	LogIn,
	Plus,
	RefreshCw,
  Server,
  Settings,
  ShieldCheck,
  Wifi,
} from 'lucide-react'
import { createRoom, getState, joinRoom, revealRoomCode } from './bridge'
import { isValidRoomCode, normalizeRoomCode } from './roomCode'
import type { StateSnapshot } from './types'

type EntryMode = 'create' | 'join'

const stateLabel: Record<StateSnapshot['state'], string> = {
  NoRoom: '未加入房间',
  Enrolling: '正在加入',
  ControlPlaneConnected: '控制面已连接',
  WaitingForPeer: '等待其他玩家',
  ConnectingPeer: '正在建立通道',
  ConnectedP2P: 'P2P 已连接',
  ConnectedRelay: 'Relay 已连接',
  Reconnecting: '正在重连',
  RecoverableError: '需要处理',
}

function App() {
  const [mode, setMode] = useState<EntryMode>('create')
  const [roomCode, setRoomCode] = useState('')
  const [snapshot, setSnapshot] = useState<StateSnapshot | null>(null)
  const [revealedRoomCode, setRevealedRoomCode] = useState('')
  const [copyState, setCopyState] = useState<'idle' | 'copied' | 'failed'>('idle')

  useEffect(() => {
    void getState().then(setSnapshot)
  }, [])

  const state = snapshot?.state ?? 'NoRoom'
  const busy = Boolean(snapshot?.busyCommand)

  async function showRoomCode() {
    const result = await revealRoomCode()
    if (result.roomCode) {
      setRevealedRoomCode(result.roomCode)
      setCopyState('idle')
      window.setTimeout(() => setRevealedRoomCode(''), 20_000)
    }
  }

  async function copyRoomCode() {
    const code = revealedRoomCode
    if (!code) {
      await showRoomCode()
      return
    }
    try {
      await navigator.clipboard.writeText(code)
      setCopyState('copied')
      window.setTimeout(() => setCopyState('idle'), 2_000)
    } catch {
      setCopyState('failed')
    }
  }

  async function submitRoom() {
    if (busy || (mode === 'join' && !isValidRoomCode(roomCode))) {
      return
    }
    setSnapshot((current) => ({
      ...(current ?? {
        revision: 0,
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
      }),
      state: 'Enrolling',
      busyCommand: mode,
      error: undefined,
    }))
    const next = mode === 'create'
      ? await createRoom({ displayName: '' })
      : await joinRoom({ roomCode, displayName: '' })
    setSnapshot(next)
  }

  return (
    <div className="app-shell">
      <aside className="sidebar">
        <div className="brand" aria-label="Sogame">
          <span className="brand-mark"><Gamepad2 size={19} strokeWidth={2.2} /></span>
          <span>Sogame</span>
        </div>

        <nav className="nav" aria-label="主导航">
          <button className="nav-item active" type="button">
            <Wifi size={17} />
            <span>连接</span>
          </button>
          <button className="nav-item" type="button" disabled>
            <Settings size={17} />
            <span>设置</span>
          </button>
        </nav>

        <div className="service-status">
          <div className="service-heading">
            <Server size={15} />
            <span>NetBird 服务</span>
          </div>
          <div className="service-value">
            <span className={`status-dot ${snapshot?.service.running ? 'online' : ''}`} />
            {snapshot?.service.running ? '运行中' : '等待检测'}
          </div>
          <div className="service-version">目标版本 v{snapshot?.service.expectedVersion ?? '0.74.7'}</div>
        </div>
      </aside>

      <main className="workspace">
        <header className="topbar">
          <div>
            <div className="eyebrow">房间连接</div>
            <h1>{stateLabel[state]}</h1>
          </div>
          <button className="icon-button" type="button" title="帮助" aria-label="帮助">
            <CircleHelp size={18} />
          </button>
        </header>

        {snapshot?.roomId ? (
          <section className="active-panel" aria-labelledby="room-title">
            <div className="room-heading">
              <div>
                <div className="eyebrow">当前房间</div>
                <h2 id="room-title">{snapshot.roomId}</h2>
              </div>
              <span className={`path-pill path-${snapshot.connectedPath}`}>
                {snapshot.connectedPath === 'p2p' ? 'P2P' : snapshot.connectedPath === 'relay' ? 'Relay' : '未建立通道'}
              </span>
            </div>

            <div className="room-code-row">
              <div>
                <span className="detail-label">房间码</span>
                <code>{revealedRoomCode || snapshot.roomCodeMasked || '********-****'}</code>
              </div>
              <div className="room-code-actions">
                <button className="icon-button" type="button" title="显示房间码" aria-label="显示房间码" onClick={() => void showRoomCode()}>
                  <Eye size={17} />
                </button>
                <button className="icon-button" type="button" title="复制房间码" aria-label="复制房间码" onClick={() => void copyRoomCode()}>
                  {copyState === 'copied' ? <Check size={17} /> : <Copy size={17} />}
                </button>
              </div>
            </div>

            <div className="detail-grid">
              <div className="detail-block">
                <span className="detail-label">本机 NetBird IP</span>
                <strong>{snapshot.localNetbirdIp || '等待分配'}</strong>
              </div>
              <div className="detail-block">
                <span className="detail-label">连接状态</span>
                <strong>{stateLabel[state]}</strong>
              </div>
            </div>

            <div className="peer-section">
              <div className="section-heading">
                <div>
                  <h3>房间成员</h3>
                  {snapshot.peersStale && <span className="stale-label"><RefreshCw size={13} />数据可能已过期</span>}
                </div>
                {snapshot.lastPeerRefreshAt && <time>{new Date(snapshot.lastPeerRefreshAt).toLocaleTimeString()}</time>}
              </div>
              {snapshot.peers.length === 0 ? (
                <div className="empty-peers">等待其他玩家加入房间</div>
              ) : (
                <div className="peer-list">
                  {snapshot.peers.map((peer) => (
                    <div className="peer-row" key={peer.id}>
                      <span className={`peer-dot ${peer.connected ? 'online' : ''}`} />
                      <div className="peer-copy">
                        <strong>{peer.name || '未命名设备'}</strong>
                        <span>{peer.netbirdIp}</span>
                      </div>
                      <span className="peer-path">{peer.path === 'p2p' ? 'P2P' : peer.path === 'relay' ? 'Relay' : peer.connected ? '已连接' : '离线'}</span>
                    </div>
                  ))}
                </div>
              )}
            </div>
            {copyState === 'failed' && <div className="inline-error" role="alert">无法访问系统剪贴板，请使用显示按钮查看房间码。</div>}
          </section>
        ) : (
        <section className="entry-panel" aria-labelledby="entry-title">
          <div className="secure-note">
            <ShieldCheck size={17} />
            <span>连接由官方 NetBird v0.74.7 提供</span>
          </div>

          <div className="entry-content">
            <h2 id="entry-title">进入游戏房间</h2>

            <div className="segmented" aria-label="进入方式">
              <button
                className={mode === 'create' ? 'selected' : ''}
                type="button"
                onClick={() => setMode('create')}
              >
                创建房间
              </button>
              <button
                className={mode === 'join' ? 'selected' : ''}
                type="button"
                onClick={() => setMode('join')}
              >
                加入房间
              </button>
            </div>

            {mode === 'join' && (
              <label className="field">
                <span>房间码</span>
                <input
                  value={roomCode}
                  onChange={(event) => setRoomCode(normalizeRoomCode(event.target.value))}
                  placeholder="XXXX-XXXX-XXXX"
                  maxLength={14}
                  autoComplete="off"
                  spellCheck={false}
                />
              </label>
            )}

            <button
              className="primary-action"
              type="button"
              disabled={busy || (mode === 'join' && !isValidRoomCode(roomCode))}
              onClick={() => void submitRoom()}
            >
              {mode === 'create' ? <Plus size={18} /> : <LogIn size={18} />}
              <span>{busy ? '正在连接...' : mode === 'create' ? '创建并连接' : '加入并连接'}</span>
            </button>

            {snapshot?.error && (
              <div className="inline-error" role="alert">
                <strong>{snapshot.error.message}</strong>
                {snapshot.error.action && <span>{snapshot.error.action}</span>}
              </div>
            )}
          </div>
        </section>
        )}
      </main>
    </div>
  )
}

export default App
