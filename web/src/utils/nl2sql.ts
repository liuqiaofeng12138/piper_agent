export interface NL2SQLChartSpec {
  chartType: 'line' | 'bar' | 'table'
  title?: string
  xField?: string
  yFields?: string[]
  seriesField?: string | null
  columns?: string[]
  rows?: Record<string, unknown>[]
}

export interface NL2SQLPayload {
  chart?: NL2SQLChartSpec
  columns?: string[]
  rows?: unknown[][]
  sql?: string
  category?: string
}

const TAG = 'piper-nl2sql-data:'

export function parseNL2SQLPayload(content: string): {
  markdown: string
  payload: NL2SQLPayload | null
} {
  const re = new RegExp(`<!--\\s*${TAG}(.+?)\\s*-->`, 's')
  const m = content.match(re)
  if (!m) {
    return { markdown: content, payload: null }
  }
  const markdown = content.replace(re, '').trim()
  try {
    const payload = JSON.parse(m[1]) as NL2SQLPayload
    return { markdown, payload }
  } catch {
    return { markdown: content, payload: null }
  }
}

export function rowsFromPayload(payload: NL2SQLPayload | null): Record<string, unknown>[] {
  if (!payload) return []
  if (payload.chart?.rows?.length) return payload.chart.rows
  const cols = payload.columns ?? []
  const raw = payload.rows ?? []
  return raw.map((row) => {
    const obj: Record<string, unknown> = {}
    cols.forEach((c, i) => {
      obj[c] = (row as unknown[])[i]
    })
    return obj
  })
}
