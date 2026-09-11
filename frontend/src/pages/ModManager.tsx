import { createSignal, Show, For, onMount } from 'solid-js'
import type { Component } from 'solid-js'
import { getMods, searchWorkshop } from '../api'
import type { AppConfig, ModStatus, WorkshopItem } from '../api'
import { fmtBytes } from '../utils'

const ModManager: Component<{ config?: AppConfig }> = () => {
  const [mods, setMods] = createSignal<ModStatus[]>([])
  const [dir, setDir] = createSignal('')
  const [missingDir, setMissingDir] = createSignal('')
  const [loading, setLoading] = createSignal(true)
  const [searches, setSearches] = createSignal<Record<string, WorkshopItem[]>>({})
  const [searching, setSearching] = createSignal<string>()

  onMount(refresh)

  async function refresh() {
    setLoading(true)
    try {
      const res = await getMods()
      setMods(res.mods)
      setDir(res.workshopDir)
      setMissingDir(res.missingDir)
    } catch {
      // ignore
    } finally {
      setLoading(false)
    }
  }

  async function search(name: string) {
    setSearching(name)
    try {
      const res = await searchWorkshop(name)
      setSearches((s) => ({ ...s, [name]: res.results }))
    } catch {
      setSearches((s) => ({ ...s, [name]: [] }))
    } finally {
      setSearching(undefined)
    }
  }

  const subId = (m: ModStatus) => m.workshopId || m.resolvedWorkshopId

  return (
    <div class="panel mods">
      <div class="toolbar">
        <div style="color: var(--text-dim); flex: 1; overflow: hidden; text-overflow: ellipsis; white-space: nowrap;">
          {dir() || missingDir()}
        </div>
        <button class="btn" onClick={refresh} disabled={loading()}>
          {loading() ? 'Проверка...' : 'Проверить'}
        </button>
      </div>
      <Show
        when={!loading()}
        fallback={<div class="loading"><div class="spinner" />Проверка модов...</div>}
      >
        <div class="mods-grid">
          <For each={mods()}>
            {(m) => (
              <div class="mod-card">
                <div class="mod-name">{m.name}</div>
                <div class="mod-meta">
                  <span class={`status-pill ${m.installed ? 'installed' : 'missing'}`}>
                    {m.installed ? 'УСТАНОВЛЕН' : 'НЕ УСТАНОВЛЕН'}
                  </span>
                  <Show when={m.installed} fallback={null}>
                    <>
                      <span class={`status-pill ${m.valid ? 'ok' : 'invalid'}`}>
                        {m.valid ? 'ВАЛИДЕН' : 'ПРОБЛЕМА'}
                      </span>
                      <span class="mod-size">{fmtBytes(m.size)}</span>
                      <span class="mod-ver" title="PBO / bikey">
                        {m.pboCount} PBO · {m.bikeyCount} bikey
                      </span>
                      <Show when={m.issues && m.issues.length > 0}>
                        <div class="mod-issues">
                          <For each={m.issues}>{(i) => <div class="issue">⚠ {i}</div>}</For>
                        </div>
                      </Show>
                    </>
                  </Show>
                </div>
                <Show when={!m.installed} fallback={null}>
                  <Show when={subId(m)} fallback={null}>
                    <div class="search-result">
                      <a class="btn" href={`steam://url/CommunityFilePage/${subId(m)}`}>
                        Подписаться в Steam
                      </a>
                      <a
                        class="ws-link"
                        href={`https://steamcommunity.com/sharedfiles/filedetails/?id=${subId(m)}`}
                        target="_blank"
                        rel="noopener noreferrer"
                      >
                        Страница в мастерской
                      </a>
                      <Show when={!m.workshopId && m.resolvedWorkshopId}>
                        <div class="mod-note">ID подобрано автоматически по модам сервера: {m.resolvedWorkshopId}</div>
                      </Show>
                    </div>
                  </Show>
                  <Show when={!subId(m)} fallback={null}>
                    <div class="search-result">
                      <Show when={searching() === m.name} fallback={null}>
                        <div style="color: var(--text-dim);">Поиск в мастерской...</div>
                      </Show>
                      <Show when={searches()[m.name] && searches()[m.name]!.length > 0} fallback={null}>
                        <For each={searches()[m.name]}>
                          {(item) => (
                            <Show when={!searching()}>
                              <a href={item.url} target="_blank" rel="noopener noreferrer">
                                ▼ {item.title}
                              </a>
                            </Show>
                          )}
                        </For>
                      </Show>
                      <Show
                        when={!searches()[m.name] && searching() !== m.name}
                        fallback={null}
                      >
                        <button class="btn" onClick={() => search(m.name)}>
                          Найти в мастерской
                        </button>
                      </Show>
                    </div>
                  </Show>
                </Show>
              </div>
            )}
          </For>
        </div>
      </Show>
    </div>
  )
}

export default ModManager