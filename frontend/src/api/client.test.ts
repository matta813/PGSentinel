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
