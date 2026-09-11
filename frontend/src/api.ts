export interface ModConf {
  name: string
  workshopId: string
}

export interface AppConfig {
  gameId: number
  gameName: string
  defaultServer: string
  mods: ModConf[]
  modParam: string
  launchParams: string
}

export interface GameServer {
  addr: string
  gameport: number
  queryport: number
  steamid: string
  name: string
  appid: number
  gamedir: string
  version: string
  map: string
  players: number
  max_players: number
  bots: number
  region: number
  secure: boolean
  dedicated: boolean
  os: string
  type: string
  proxy: boolean
}

export interface Player {
  index: number
  name: string
  score: number
  duration: number
}

export interface LiveServer {
  addr: string
  name?: string
  map?: string
  players?: number
  maxPlayers?: number
  bots?: number
  version?: string
  keywords?: string
  appId?: number
  ping?: number
  game?: string
  folder?: string
  password?: boolean
  playerList?: Player[]
  error?: string
}

export interface ModStatus {
  name: string
  workshopId: string
  resolvedWorkshopId: string
  installed: boolean
  valid: boolean
  path: string
  size: number
  hasMeta: boolean
  pboCount: number
  bikeyCount: number
  issues: string[]
}

export interface ServerMod {
  workshopId: string
  title: string
  installed: boolean
  valid: boolean
  path: string
  issues: string[]
}

export interface ServerMods {
  addr: string
  workshopDir: string
  keywords: string
  count: number
  mods: ServerMod[]
}

export interface WorkshopItem {
  publishedfileid: string
  title: string
  url: string
  creator: string
  short_description: string
  subscriptions: number
  file_size: number
}

async function request<T>(url: string, init?: RequestInit): Promise<T> {
  const res = await fetch(url, init)
  if (!res.ok) {
    const data = await res.json().catch(() => null)
    throw new Error(data?.error ?? `HTTP ${res.status}`)
  }
  return res.json() as Promise<T>
}

export function getConfig(): Promise<{ mods: ModConf[]; modParam: string; gameName: string; gameId: number; defaultServer: string; launchParams: string }> {
  return request<AppConfig>('/api/config')
}

export function getServers(): Promise<{ servers: GameServer[]; count: number }> {
  return request<{ servers: GameServer[]; count: number }>('/api/servers')
}

export function getServerLive(addr: string): Promise<LiveServer> {
  return request<LiveServer>(`/api/server?addr=${encodeURIComponent(addr)}`)
}

export function getServersLive(addrs: string[]): Promise<{ servers: LiveServer[] }> {
  return request<{ servers: LiveServer[] }>('/api/servers/live', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ addrs }),
  })
}

export function getMods(): Promise<{ workshopDir: string; missingDir: string; mods: ModStatus[] }> {
  return request<{ workshopDir: string; missingDir: string; mods: ModStatus[] }>('/api/mods')
}

export function searchWorkshop(q: string): Promise<{ results: WorkshopItem[] }> {
  return request<{ results: WorkshopItem[] }>(`/api/mods/search?q=${encodeURIComponent(q)}`)
}

export function getServerMods(addr: string): Promise<ServerMods> {
  return request<ServerMods>(`/api/server/mods?addr=${encodeURIComponent(addr)}`)
}

export function launchDayZ(server: string, password: string): Promise<{ ok: boolean; url: string }> {
  return request<{ ok: boolean; url: string }>('/api/launch', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ server, password }),
  })
}

export interface Settings {
  gameDir: string
  workshopDir: string
  overrideGameDir: string
  overrideWorkshopDir: string
  detectedGameDir: string
  detectedWorkshopDir: string
  gameInstalled: boolean
  workshopFound: boolean
  defaultServer: string
  launchParams: string
  gameId: number
}

export function getSettings(): Promise<Settings> {
  return request<Settings>('/api/settings')
}

export function saveSettings(gameDir: string, workshopDir: string, launchParams: string): Promise<Settings> {
  return request<Settings>('/api/settings', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ gameDir, workshopDir, launchParams }),
  })
}