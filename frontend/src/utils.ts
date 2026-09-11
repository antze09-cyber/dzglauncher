export function fmtBytes(bytes: number): string {
  if (!bytes) return '0 B'
  const units = ['B', 'KB', 'MB', 'GB']
  let i = 0
  let v = bytes
  while (v >= 1024 && i < units.length - 1) {
    v /= 1024
    i++
  }
  return `${v.toFixed(i === 0 ? 0 : 1)} ${units[i]}`
}

export function fmtDuration(sec: number): string {
  if (sec < 0) return '-'
  const h = Math.floor(sec / 3600)
  const m = Math.floor((sec % 3600) / 60)
  if (h > 0) return `${h}ч ${m}м`
  return `${m}м ${Math.floor(sec % 60)}с`
}

export function regionName(region: number): string {
  const map: Record<number, string> = {
    0: 'US-East',
    1: 'US-West',
    2: 'S.America',
    3: 'Europe',
    4: 'Asia',
    5: 'Australia',
    6: 'MiddleEast',
    7: 'Africa',
    255: 'World',
  }
  return map[region] ?? String(region)
}

export function pingClass(ping?: number): string {
  if (ping == null) return ''
  if (ping <= 60) return 'good'
  if (ping <= 120) return 'mid'
  return 'bad'
}