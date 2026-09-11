import { createSignal, onMount, Show } from 'solid-js'
import type { Component, JSX } from 'solid-js'
import { getSettings, saveSettings } from '../api'
import type { Settings } from '../api'

const SettingsPanel: Component<{ compact?: boolean; onSaved?: () => void }> = (props) => {
  const [settings, setSettings] = createSignal<Settings>()
  const [gameDir, setGameDir] = createSignal('')
  const [workshopDir, setWorkshopDir] = createSignal('')
  const [loading, setLoading] = createSignal(true)
  const [saving, setSaving] = createSignal(false)
  const [detecting, setDetecting] = createSignal(false)
  const [msg, setMsg] = createSignal<{ text: string; error?: boolean }>()

  onMount(() => {
    getSettings()
      .then((s) => {
        setSettings(s)
        setGameDir(s.overrideGameDir || s.detectedGameDir || '')
        setWorkshopDir(s.overrideWorkshopDir || s.detectedWorkshopDir || '')
      })
      .catch((e) => setMsg({ text: String(e.message || e), error: true }))
      .finally(() => setLoading(false))
  })

  const cur = (): Settings | undefined => settings()

  const applyDetect = () => {
    const s = cur()
    if (!s) return
    setDetecting(true)
    getSettings()
      .then((fresh) => {
        setSettings(fresh)
        if (!gameDir()) setGameDir(fresh.overrideGameDir || fresh.detectedGameDir || '')
        if (!workshopDir()) setWorkshopDir(fresh.overrideWorkshopDir || fresh.detectedWorkshopDir || '')
      })
      .catch((e) => setMsg({ text: String(e.message || e), error: true }))
      .finally(() => setDetecting(false))
  }

  const save: JSX.EventHandler<HTMLButtonElement, MouseEvent> = async () => {
    setSaving(true)
    setMsg(undefined)
    try {
      const s = await saveSettings(gameDir().trim(), workshopDir().trim())
      setSettings(s)
      setMsg({ text: 'Пути сохранены' })
      props.onSaved?.()
    } catch (e) {
      setMsg({ text: String(e.message || e), error: true })
    } finally {
      setSaving(false)
    }
  }

  const useAuto = (field: 'game' | 'workshop') => {
    const s = cur()
    if (!s) return
    if (field === 'game') {
      setGameDir(s.detectedGameDir || '')
    } else {
      setWorkshopDir(s.detectedWorkshopDir || '')
    }
  }

  return (
    <div class="panel settings">
      <Show when={!loading()} fallback={<div class="loading"><div class="spinner" />Загрузка…</div>}>
        <div class="settings-row">
          <label>Папка DayZ</label>
          <div class="settings-input">
            <input
              value={gameDir()}
              onChange={(e) => setGameDir(e.currentTarget.value)}
              placeholder="Путь к steamapps/common/DayZ (пусто — автоопределение)"
            />
            <button class="btn" onClick={() => useAuto('game')} disabled={!cur()?.detectedGameDir}>
              Автоопределение
            </button>
          </div>
          <Show when={cur()?.detectedGameDir}>
            <div class="settings-hint">
              Найдено: <code>{cur()!.detectedGameDir}</code>
            </div>
          </Show>
        </div>

        <div class="settings-row">
          <label>Папка мастерской</label>
          <div class="settings-input">
            <input
              value={workshopDir()}
              onChange={(e) => setWorkshopDir(e.currentTarget.value)}
              placeholder="Путь к steamapps/workshop/content/221100 (пусто — автоопределение)"
            />
            <button class="btn" onClick={() => useAuto('workshop')} disabled={!cur()?.detectedWorkshopDir}>
              Автоопределение
            </button>
          </div>
          <Show when={cur()?.detectedWorkshopDir}>
            <div class="settings-hint">
              Найдено: <code>{cur()!.detectedWorkshopDir}</code>
            </div>
          </Show>
        </div>

        <div class="settings-status">
          <Show when={cur()?.gameInstalled} fallback={<span class="badge warn">Игра не найдена</span>}>
            <span class="badge ok">Игра найдена</span>
          </Show>
          <Show when={cur()?.workshopFound} fallback={<span class="badge warn">Мастерская не найдена</span>}>
            <span class="badge ok">Мастерская найдена</span>
          </Show>
          <button class="btn" onClick={applyDetect} disabled={detecting()}>
            {detecting() ? 'Проверка…' : 'Проверить пути'}
          </button>
          <button class="btn primary" onClick={save} disabled={saving()}>
            {saving() ? 'Сохранение…' : 'Сохранить'}
          </button>
        </div>

        <Show when={msg()}>
          <div class={`settings-msg ${msg()!.error ? 'error' : ''}`}>{msg()!.text}</div>
        </Show>
      </Show>
    </div>
  )
}

export default SettingsPanel