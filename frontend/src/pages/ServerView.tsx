import { createSignal, createEffect, Show, For } from 'solid-js'
import type { Component } from 'solid-js'
import { getServerLive, getServerMods } from '../api'
import type { AppConfig, LiveServer, ServerMod } from '../api'

const ServerView: Component<{ config?: AppConfig }> = (props) => {
  const addr = () => props.config()?.defaultServer || '146.66.12.4:2402'

  const [live, setLive] = createSignal<LiveServer>()
  const [liveErr, setLiveErr] = createSignal<string>()
  const [mods, setMods] = createSignal<ServerMod[]>()
  const [modsErr, setModsErr] = createSignal<string>()
  const [loading, setLoading] = createSignal(true)

  createEffect(() => {
    const a = addr()
    let stopped = false
    const fetchLive = async () => {
      try {
        const d = await getServerLive(a)
        if (!stopped) {
          setLive(d)
          setLiveErr(undefined)
        }
      } catch (e) {
        if (!stopped) setLiveErr(String(e.message || e))
      }
    }
    const fetchMods = async () => {
      try {
        const d = await getServerMods(a)
        if (!stopped) {
          setMods(d.mods)
          setModsErr(undefined)
        }
      } catch (e) {
        if (!stopped) setModsErr(String(e.message || e))
      } finally {
        if (!stopped) setLoading(false)
      }
    }
    fetchLive()
    fetchMods()
    const timer = setInterval(fetchLive, 10000)
    return () => {
      stopped = true
      clearInterval(timer)
    }
  })

  const workshopUrl = (id: string) => `https://steamcommunity.com/sharedfiles/filedetails/?id=${id}`

  return (
    <div class="panel server">
      <div class="server-header">
        <div class="server-title">
          <span class="pulse-dot" />
          {live()?.name || addr()}
        </div>
        <div class="server-stats">
          <span>Игроки: <b>{live()?.players ?? '—'} / {live()?.maxPlayers ?? '—'}</b></span>
          <span>Карта: <b>{live()?.map || '—'}</b></span>
          <span>Пинг: <b>{live()?.ping != null ? `${live()?.ping} ms` : '—'}</b></span>
          <span class="srv-addr">{addr()}</span>
          <Show when={live()?.password} fallback={null}>
            <span class="status-pill invalid">с паролем</span>
          </Show>
        </div>
        <Show when={liveErr()}>
          <div class="server-note warn">{liveErr()} — живая статистика недоступна (UDP может быть закрыт)</div>
        </Show>
      </div>

      <div class="server-mods">
        <div class="server-mods-head">
          <h3>Моды сервера ({mods()?.length ?? 0})</h3>
          <button
            class="btn"
            onClick={() => {
              setMods(undefined)
              setModsErr(undefined)
              setLoading(true)
              getServerMods(addr())
                .then((d) => {
                  setMods(d.mods)
                  setModsErr(undefined)
                })
                .catch((e) => setModsErr(String(e.message || e)))
                .finally(() => setLoading(false))
            }}
          >
            Обновить
          </button>
        </div>

        <Show
          when={!loading()}
          fallback={<div class="loading"><div class="spinner" />Загрузка модов сервера...</div>}
        >
          <Show when={modsErr()} fallback={null}>
            <div class="empty" style="color: var(--danger);">{modsErr()}</div>
          </Show>
          <Show
            when={mods() && mods()!.length > 0}
            fallback={
              <Show when={!modsErr()}>
                <div class="empty">Моды сервера не определены (нет данных A2S)</div>
              </Show>
            }
          >
            <div class="srv-mods-grid">
              <For each={mods()}>
                {(m) => (
                  <div class="srv-mod-card">
                    <div class="srv-mod-title" title={m.title}>{m.title || m.workshopId}</div>
                    <div class="srv-mod-meta">
                      <span class="srv-mod-id">{m.workshopId}</span>
                      <Show when={m.installed} fallback={<span class="status-pill missing">НЕТ</span>}>
                        <span class={`status-pill ${m.valid ? 'installed' : 'invalid'}`}>
                          {m.valid ? 'ЕСТЬ' : 'БИТЫЙ'}
                        </span>
                      </Show>
                    </div>
                    <Show when={m.issues && m.issues.length > 0}>
                      <div class="mod-issues">
                        <For each={m.issues}>{(i) => <div class="issue">⚠ {i}</div>}</For>
                      </div>
                    </Show>
                    <Show when={!m.installed}>
                      <a class="ws-link" href={workshopUrl(m.workshopId)} target="_blank" rel="noopener noreferrer">
                        Открыть в Steam Workshop ↗
                      </a>
                    </Show>
                  </div>
                )}
              </For>
            </div>
          </Show>
        </Show>
      </div>

      <div class="section players">
        <h3>Игроки ({live()?.playerList?.length ?? 0})</h3>
        <Show when={live()?.playerList?.length} fallback={<div class="empty">Нет данных об игроках</div>}>
          <div class="players-list">
            <For each={live()!.playerList}>
              {(p) => (
                <div class="player-row">
                  <span class="pname" title={p.name}>{p.name}</span>
                </div>
              )}
            </For>
          </div>
        </Show>
      </div>
    </div>
  )
}

export default ServerView