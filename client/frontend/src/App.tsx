import { useEffect, useState } from 'react'
import {
  CircleHelp,
  Gamepad2,
  LogIn,
  Plus,
  Server,
  Settings,
  ShieldCheck,
  Wifi,
} from 'lucide-react'
import { getState } from './bridge'
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

  useEffect(() => {
    void getState().then(setSnapshot)
  }, [])

  const state = snapshot?.state ?? 'NoRoom'

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
              disabled={mode === 'join' && !isValidRoomCode(roomCode)}
            >
              {mode === 'create' ? <Plus size={18} /> : <LogIn size={18} />}
              <span>{mode === 'create' ? '创建并连接' : '加入并连接'}</span>
            </button>
          </div>
        </section>
      </main>
    </div>
  )
}

export default App
