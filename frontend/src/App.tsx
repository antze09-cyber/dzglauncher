import { createSignal, createEffect, Show, For } from 'solid-js'
import type { Component } from 'solid-js'
import ServerView from './pages/ServerView'
import ModManager from './pages/ModManager'
import SettingsPanel from './pages/SettingsPanel'
import { getConfig, getMods, getSettings, launchDayZ } from './api'
import type { AppConfig, ModStatus, Settings } from './api'

type Tab = 'server' | 'mods' | 'settings'

const App: Component = () => {
  const [config, setConfig] = createSignal<AppConfig>()
  const [tab, setTab] = createSignal<Tab>('server')
  const [launching, setLaunching] = createSignal(false)
  const [launchMsg, setLaunchMsg] = createSignal<{ text: string; error?: boolean }>()
  const [modal, setModal] = createSignal<{ missing: ModStatus[]; force: boolean }>()
  const [missingNow, setMissingNow] = createSignal<ModStatus[]>()
  const [settings, setSettings] = createSignal<Settings>()
  const [setupOpen, setSetupOpen] = createSignal(false)

  createEffect(() => {
    getConfig()
      .then(setConfig)
      .catch((e) => setLaunchMsg({ text: String(e.message || e), error: true }))
  })

  createEffect(() => {
    getSettings()
      .then(setSettings)
      .catch(() => {})
  })

  // Пути ещё не настроены (первый запуск) — показываем мастер настройки.
  const pathsNeedSetup = () => {
    const s = settings()
    if (!s) return false
    const anySet = s.overrideGameDir !== '' || s.overrideWorkshopDir !== ''
    return !s.gameInstalled || !s.workshopFound || !anySet
  }

  const reloadSettings = async () => {
    try {
      setSettings(await getSettings())
    } catch {
      // ignore
    }
  }

  const targetAddr = () => config()?.defaultServer || '146.66.12.4:2402'

  const actuallyLaunch = async () => {
    setLaunching(true)
    setLaunchMsg(undefined)
    try {
      const res = await launchDayZ(targetAddr(), '')
      setLaunchMsg({ text: `Steam запущен: ${res.url}` })
    } catch (e) {
      setLaunchMsg({ text: String(e), error: true })
    } finally {
      setLaunching(false)
    }
  }

  const doLaunch = async () => {
    // Автопроверка: если моды не найдены в мастерской — предупреждаем.
    try {
      const res = await getMods()
      if (!res.missingDir) {
        const missing = res.mods.filter((m) => !m.installed)
        setMissingNow(missing)
        if (missing.length > 0) {
          setModal({ missing, force: false })
          return
        }
      }
    } catch {
      // Если проверку не удалось выполнить — запускаем как есть.
    }
    await actuallyLaunch()
  }

  const forceLaunch = async () => {
    setModal(undefined)
    await actuallyLaunch()
  }

  return (
    <div class="app">
      <div class="topbar">
        <div class="logo">
          DZ<span>LAUNCHER</span>
          <small>DayZ // WOC</small>
        </div>
        <div class="tabs">
          <button class={`tab ${tab() === 'server' ? 'active' : ''}`} onClick={() => setTab('server')}>
            Сервер
          </button>
          <button class={`tab ${tab() === 'mods' ? 'active' : ''}`} onClick={() => setTab('mods')}>
            Моды
          </button>
          <button class={`tab ${tab() === 'settings' ? 'active' : ''}`} onClick={() => setTab('settings')}>
            Настройки
          </button>
        </div>
      </div>

      <div class="main">
        <Show when={tab() === 'server'}>
          <ServerView config={config()} />
        </Show>
        <Show when={tab() === 'mods'}>
          <ModManager config={config()} />
        </Show>
        <Show when={tab() === 'settings'}>
          <SettingsPanel onSaved={reloadSettings} />
        </Show>
      </div>

      <Show when={pathsNeedSetup() && setupOpen()}>
        <div class="modal-overlay">
          <div class="modal setup-modal">
            <h2>Настройка путей DayZ</h2>
            <p>Проверьте пути к игре и мастерской. Если Steam установлен в стандартное место — просто нажмите «Сохранить».</p>
            <SettingsPanel onSaved={reloadSettings} />
            <div class="modal-actions">
              <button class="btn" onClick={() => setSetupOpen(false)}>Позже</button>
            </div>
          </div>
        </div>
      </Show>

      <div class="launchbar">
        <div class="server-target">
          ▶ {config()?.gameName || 'DayZ'} <span class="fmt">{targetAddr()}</span>
        </div>
        <Show when={pathsNeedSetup() && !setupOpen()}>
          <button class="btn" onClick={() => setSetupOpen(true)}>
            Настроить пути
          </button>
        </Show>
        <Show
          when={missingNow() && missingNow()!.length > 0}
          fallback={
            <Show when={launchMsg()} fallback={<div class="status">Запуск через Steam с подключением к выбранному серверу</div>}>
              <div class={`status ${launchMsg()!.error ? 'error' : ''}`}>{launchMsg()!.text}</div>
            </Show>
          }
        >
          <div class="status warn">Не установлено модов: {missingNow()!.length}</div>
        </Show>
        <button class="btn primary" onClick={doLaunch} disabled={launching()}>
          {launching() ? 'Запуск...' : 'Играть'}
        </button>
      </div>

      <Show when={modal()}>
        <div class="modal-overlay">
          <div class="modal">
            <h2>Не хватает модов</h2>
            <p>Перед подключением к серверу нужно установить эти моды из мастерской Steam:</p>
            <ul class="modal-list">
              <For each={modal()!.missing}>
                {(m) => (
                  <li>
                    <span class="mod-name">{m.name}</span>
                    <a
                      href={`/api/mods/search?q=${encodeURIComponent(m.name.replace(/^@/, ''))}`}
                      class="open-search"
                      onClick={(e) => {
                        e.preventDefault()
                        setTab('mods')
                        setModal(undefined)
                      }}
                    >
                      Открыть в менеджере
                    </a>
                  </li>
                )}
              </For>
            </ul>
            <div class="modal-actions">
              <button class="btn" onClick={() => setModal(undefined)}>Отмена</button>
              <button class="btn" onClick={forceLaunch}>Запустить без модов</button>
            </div>
          </div>
        </div>
      </Show>
    </div>
  )
}

export default App