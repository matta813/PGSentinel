import { afterEach, expect, test, vi } from 'vitest'
import { api, APIError } from './client'

afterEach(() => vi.restoreAllMocks())

test('reports an actionable error when an API route returns the SPA document', async () => {
  vi.spyOn(globalThis, 'fetch').mockResolvedValue(new Response('<!doctype html>', {
    status: 200,
    headers: { 'Content-Type': 'text/html; charset=utf-8' },
  }))

  await expect(api.get('/incidents')).rejects.toEqual(expect.objectContaining<Partial<APIError>>({
    message: 'Invalid API response',
    detail: expect.stringContaining('reverse-proxy'),
    status: 200,
  }))
})

test('accepts JSON when an intermediary omits the JSON content type', async () => {
  vi.spyOn(globalThis, 'fetch').mockResolvedValue(new Response(JSON.stringify([{ id: 'incident-1' }])))

  await expect(api.get('/incidents')).resolves.toEqual([{ id: 'incident-1' }])
})

test('downloads a binary response using the server filename', async () => {
  vi.spyOn(globalThis, 'fetch').mockResolvedValue(new Response('bundle', {
    status: 200,
    headers: { 'Content-Disposition': 'attachment; filename="diagnostics.zip"' },
  }))

  const result = await api.download('/diagnostic-bundle')
  expect(result.filename).toBe('diagnostics.zip')
  expect(await result.blob.text()).toBe('bundle')
})
