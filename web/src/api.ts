import type { ApiError, MatchResponse } from './types'

const jsonHeaders = { 'Content-Type': 'application/json' }

async function parseError(res: Response): Promise<string> {
  try {
    const data = (await res.json()) as ApiError & { detail?: string }
    return data.error_fa || data.error || data.detail || res.statusText
  } catch {
    return res.statusText || `HTTP ${res.status}`
  }
}

export async function healthCheck(): Promise<boolean> {
  try {
    const r = await fetch('/api/health', { method: 'GET' })
    return r.ok
  } catch {
    return false
  }
}

export async function matchCustomer(input: {
  national_id: string
  visit_purpose?: string
  include_default_warning?: boolean
}): Promise<{ ok: true; data: MatchResponse } | { ok: false; status: number; message: string }> {
  const r = await fetch('/api/match', {
    method: 'POST',
    headers: jsonHeaders,
    body: JSON.stringify(input),
  })
  if (r.ok) return { ok: true, data: (await r.json()) as MatchResponse }
  return { ok: false, status: r.status, message: await parseError(r) }
}

export async function matchColdStart(input: {
  name: string
  age: number
  gender: string
  occupation: string
  employment_type: string
  approx_income: number
  visit_purpose?: string
  include_default_warning?: boolean
}): Promise<{ ok: true; data: MatchResponse } | { ok: false; status: number; message: string }> {
  const r = await fetch('/api/match/cold-start', {
    method: 'POST',
    headers: jsonHeaders,
    body: JSON.stringify(input),
  })
  if (r.ok) return { ok: true, data: (await r.json()) as MatchResponse }
  return { ok: false, status: r.status, message: await parseError(r) }
}

export async function agentChat(input: {
  message: string
  thread_id?: string
}): Promise<{ ok: true; reply: string; thread_id: string } | { ok: false; message: string }> {
  try {
    const r = await fetch('/api/agent/chat', {
      method: 'POST',
      headers: jsonHeaders,
      body: JSON.stringify(input),
    })
    if (!r.ok) return { ok: false, message: await parseError(r) }
    const data = (await r.json()) as { reply: string; thread_id: string }
    return { ok: true, reply: data.reply, thread_id: data.thread_id }
  } catch (e) {
    return { ok: false, message: e instanceof Error ? e.message : 'network error' }
  }
}
