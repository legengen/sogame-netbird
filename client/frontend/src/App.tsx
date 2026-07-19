import { useEffect, useState } from 'react'
import {
	Check,
	CircleHelp,
	Copy,
	Download,
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
import { connectRoom, createRoom, disconnectRoom, exportDiagnostics, getState, joinRoom, leaveRoom, repairService, revealRoomCode, switchRoom } from './bridge'
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
  const [switchOpen, setSwitchOpen] = useState(false)
  const [switchMode, setSwitchMode] = useState<EntryMode>('join')
  const [switchCode, setSwitchCode] = useState('')
  const [switchConfirmed, setSwitchConfirmed] = useState(false)
  const [diagnosticMessage, setDiagnosticMessage] = useState('')

  useEffect(() => {
    void getState().then(setSnapshot)
  }, [])

  const state = snapshot?.state ?? 'NoRoom'
  const busy = Boolean(snapshot?.busyCommand)
  const errorCode = snapshot?.error?.code
  const canRepair = Boolean(snapshot?.service.repairRequired) || errorCode === 'NETBIRD_SERVICE_MISSING' || errorCode === 'NETBIRD_VERSION_MISMATCH' || errorCode === 'NETBIRD_SERVICE_UNAVAILABLE'

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

  async function runLifecycle(action: 'connect' | 'disconnect' | 'leave') {
    if (busy) return
    if (action === 'leave' && !window.confirm('确认离开当前房间？这会移除本机房间配置。')) return
    setSnapshot((current) => current ? { ...current, busyCommand: action, error: undefined } : current)
    const next = action === 'connect' ? await connectRoom() : action === 'disconnect' ? await disconnectRoom() : await leaveRoom()
    setSnapshot(next)
    if (action === 'leave' && next.state === 'NoRoom') {
      setSwitchOpen(false)
      setSwitchConfirmed(false)
    }
  }

  async function submitSwitch() {
    if (busy || !switchConfirmed || (switchMode === 'join' && !isValidRoomCode(switchCode))) return
    setSnapshot((current) => current ? { ...current, busyCommand: 'switch', error: undefined } : current)
    const next = await switchRoom({ mode: switchMode, roomCode: switchCode, displayName: '', confirmed: true })
    setSnapshot(next)
    if (next.state !== 'RecoverableError') {
      setSwitchOpen(false)
      setSwitchConfirmed(false)
    }
  }

  async function runRepair() {
    if (busy) return
    setSnapshot((current) => current ? { ...current, busyCommand: 'repair', error: undefined } : current)
    setSnapshot(await repairService())
  }

  async function runDiagnostics() {
    if (busy) return
    const result = await exportDiagnostics()
    setDiagnosticMessage(result.path ? '诊断包已保存到本机' : result.error?.message || '诊断导出失败')
    window.setTimeout(() => setDiagnosticMessage(''), 4_000)
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

        {state === 'Reconnecting' && (
          <div className="recovery-banner reconnect-banner" role="status">
            <RefreshCw size={17} />
            <div><strong>正在恢复 NetBird 连接</strong><span>官方 daemon 正在重新连接控制面，房间身份会保留。</span></div>
          </div>
        )}
        {snapshot?.error && (state === 'RecoverableError' || canRepair) && (
          <div className="recovery-banner error-banner" role="alert">
            <div>
              <strong>{snapshot.error.message}</strong>
              <span>{snapshot.error.action || '请稍后重试'}</span>
            </div>
            {canRepair && <button type="button" className="secondary-action" disabled={busy} onClick={() => void runRepair()}>{busy ? '正在修复...' : '修复服务'}</button>}
          </div>
        )}

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
            <div className="lifecycle-actions">
              <button type="button" className="secondary-action" disabled={busy} onClick={() => void runLifecycle('connect')}>连接</button>
              <button type="button" className="secondary-action" disabled={busy} onClick={() => void runLifecycle('disconnect')}>断开</button>
              <button type="button" className="secondary-action" disabled={busy} onClick={() => void runLifecycle('leave')}>离开</button>
              <button type="button" className="secondary-action" disabled={busy} onClick={() => setSwitchOpen((open) => !open)}>切换房间</button>
              <button type="button" className="secondary-action" disabled={busy} onClick={() => void runDiagnostics()}><Download size={15} />导出诊断</button>
            </div>
            {switchOpen && (
              <div className="switch-panel">
                <div className="segmented" aria-label="切换方式">
                  <button className={switchMode === 'create' ? 'selected' : ''} type="button" onClick={() => setSwitchMode('create')}>创建新房间</button>
                  <button className={switchMode === 'join' ? 'selected' : ''} type="button" onClick={() => setSwitchMode('join')}>加入其他房间</button>
                </div>
                {switchMode === 'join' && (
                  <input className="switch-input" value={switchCode} onChange={(event) => setSwitchCode(normalizeRoomCode(event.target.value))} placeholder="XXXX-XXXX-XXXX" maxLength={14} autoComplete="off" spellCheck={false} />
                )}
                <label className="confirm-row"><input type="checkbox" checked={switchConfirmed} onChange={(event) => setSwitchConfirmed(event.target.checked)} />确认先离开当前房间</label>
                <button className="primary-action" type="button" disabled={busy || !switchConfirmed || (switchMode === 'join' && !isValidRoomCode(switchCode))} onClick={() => void submitSwitch()}>切换并连接</button>
              </div>
            )}
            {copyState === 'failed' && <div className="inline-error" role="alert">无法访问系统剪贴板，请使用显示按钮查看房间码。</div>}
            {diagnosticMessage && <div className="inline-note" role="status">{diagnosticMessage}</div>}
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
