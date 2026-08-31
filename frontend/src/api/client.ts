const base = '/api/v1'

export class APIError extends Error {
  constructor(message: string, readonly detail: string, readonly status: number) { super(message) }
}

async function request<T>(path: string, init?: RequestInit): Promise<T> {
  const response = await fetch(base + path, { ...init, credentials: 'same-origin', headers: { 'Content-Type': 'application/json', ...init?.headers } })
  const contentType = response.headers.get('Content-Type') ?? ''
  const isHTML = contentType.includes('text/html')
  if (!response.ok) {
    let body: { error?: string; detail?: string } = {}
    if (!isHTML) {
      try { body = await response.json() } catch { /* malformed error response */ }
    }
    if (response.status === 401 && path !== '/auth/login') window.dispatchEvent(new Event('pgsentinel:unauthorized'))
    throw new APIError(body.error ?? 'Request failed', body.detail ?? '', response.status)
  }
  if (response.status === 204) return undefined as T
  if (isHTML) {
    throw new APIError('Invalid API response', 'The server returned a non-JSON response. Check the API or reverse-proxy configuration.', response.status)
  }
  return response.json() as Promise<T>
}

export const api = {
  get: <T>(path: string, init?: RequestInit) => request<T>(path, init),
  post: <T>(path: string, value?: unknown) => request<T>(path, { method: 'POST', body: value === undefined ? undefined : JSON.stringify(value) }),
  put: <T>(path: string, value: unknown) => request<T>(path, { method: 'PUT', body: JSON.stringify(value) }),
  delete: (path: string) => request<void>(path, { method: 'DELETE' }),
}
