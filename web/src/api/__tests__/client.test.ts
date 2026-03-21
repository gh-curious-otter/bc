import { describe, it, expect, vi, beforeEach } from 'vitest';
import { api } from '../client';

const fetchMock = globalThis.fetch as ReturnType<typeof vi.fn>;

function jsonResponse(body: unknown, status = 200, statusText = 'OK') {
  return Promise.resolve({
    ok: status >= 200 && status < 300,
    status,
    statusText,
    json: () => Promise.resolve(body),
  } as Response);
}

beforeEach(() => {
  fetchMock.mockReset();
});

describe('api.request', () => {
  it('sends Content-Type header', async () => {
    fetchMock.mockReturnValue(jsonResponse([]));
    await api.listAgents();
    const [, init] = fetchMock.mock.calls[0] as [string, RequestInit];
    expect((init.headers as Record<string, string>)['Content-Type']).toBe('application/json');
  });

  it('throws on non-ok response', async () => {
    fetchMock.mockReturnValue(jsonResponse(null, 500, 'Internal Server Error'));
    await expect(api.listAgents()).rejects.toThrow('API error: 500 Internal Server Error');
  });

  it('formats URL with path', async () => {
    fetchMock.mockReturnValue(jsonResponse({}));
    await api.getAgent('test-agent');
    const [url] = fetchMock.mock.calls[0] as [string];
    expect(url).toBe('/api/agents/test-agent');
  });

  it('encodes agent name in URL', async () => {
    fetchMock.mockReturnValue(jsonResponse({}));
    await api.getAgent('agent with spaces');
    const [url] = fetchMock.mock.calls[0] as [string];
    expect(url).toBe('/api/agents/agent%20with%20spaces');
  });

  it('sends POST with JSON body for sendToAgent', async () => {
    fetchMock.mockReturnValue(jsonResponse(null));
    await api.sendToAgent('bot', 'hello');
    const [, init] = fetchMock.mock.calls[0] as [string, RequestInit];
    expect(init.method).toBe('POST');
    expect(JSON.parse(init.body as string)).toEqual({ message: 'hello' });
  });

  it('sends POST with JSON body for sendToChannel', async () => {
    fetchMock.mockReturnValue(jsonResponse({ id: 1, sender: 'web', content: 'hi', created_at: '' }));
    await api.sendToChannel('general', 'hi');
    const [url, init] = fetchMock.mock.calls[0] as [string, RequestInit];
    expect(url).toBe('/api/channels/general/messages');
    expect(init.method).toBe('POST');
    expect(JSON.parse(init.body as string)).toEqual({ sender: 'web', content: 'hi' });
  });

  it('passes query params for getLogs', async () => {
    fetchMock.mockReturnValue(jsonResponse([]));
    await api.getLogs(25);
    const [url] = fetchMock.mock.calls[0] as [string];
    expect(url).toContain('tail=25');
  });
});
